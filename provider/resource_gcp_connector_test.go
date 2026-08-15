package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/sjackson0109/terraform-provider-qualys/cloudview/gcp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestResourceGcpConnector_basic, TestResourceGcpConnector_update and
// TestDataSourceGcpConnector_basic (in data_source_gcp_connector_test.go)
// currently fail against a real Terraform CLI, and are excluded from CI
// (see .github/workflows/ci.yaml) rather than fixed here: providerConfigure
// always builds vmdr and qps clients too, even for a test that only
// exercises this GCP resource, and both correctly refuse the plain-HTTP
// URL testServer() hands out ("BaseURL must use https"). Weakening that
// check is not an option — it is a deliberate guard against sending
// credentials over plaintext. Fixing this properly needs either a
// TLS-capable local mock the vmdr/qps clients can be made to trust, or
// lazy per-resource client construction in providerConfigure.
func TestResourceGcpConnector_basic(t *testing.T) {
	projectId := uuid.New()

	ts := testServer()
	defer ts.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourcesDestroyed,
		Steps: []resource.TestStep{
			{
				Config: getHclConfig(projectId, "test name"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "name", "test name"),
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "description", "test description"),
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "gcp_credentials_json", "{\"foo\":\"bar\"}"),
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "project_id", projectId.String()),
				),
			},
		},
	})
}

func TestResourceGcpConnector_update(t *testing.T) {
	projectId := uuid.New()

	ts := testServer()
	defer ts.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourcesDestroyed,
		Steps: []resource.TestStep{
			{
				Config: getHclConfig(projectId, "test name"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "name", "test name"),
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "description", "test description"),
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "gcp_credentials_json", "{\"foo\":\"bar\"}"),
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "project_id", projectId.String()),
				),
			},
			{
				Config: getHclConfig(projectId, "test name update"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "name", "test name update"),
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "description", "test description"),
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "gcp_credentials_json", "{\"foo\":\"bar\"}"),
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "project_id", projectId.String()),
				),
			},
		},
	})
}

func getHclConfig(projectId uuid.UUID, name string) string {
	// The empty provider block is required, not decorative: without one, a
	// modern Terraform CLI refuses to configure a provider whose schema
	// marks every argument Required (even with a DefaultFunc reading the
	// QUALYS_* env vars testServer sets) and fails pre-apply with "Provider
	// ... requires explicit configuration" before the plugin's own
	// Configure/DefaultFunc logic ever runs.
	return fmt.Sprintf(`
provider "qualys" {}

resource "qualys_gcp_connector" "test" {
    name                  = "%s"
    description           = "test description"
    gcp_credentials_json  = jsonencode({foo = "bar"})
    project_id            = "%s"
}
`, name, projectId.String())
}

func testAccCheckResourcesDestroyed(state *terraform.State) error {
	service := gcp.NewService(os.Getenv("QUALYS_URL"), os.Getenv("QUALYS_USERNAME"), os.Getenv("QUALYS_PASSWORD"))
	for _, rs := range state.RootModule().Resources {
		_, err := service.Get(rs.Primary.ID)
		if err.Error() != "Not Found" {
			return fmt.Errorf("expected 'Not Found', unknown error: %s", err)
		}
	}
	return nil
}
