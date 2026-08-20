package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAcceptanceGCPConnectorLifecycle is a genuine acceptance test: it runs
// a real Terraform create -> read/refresh -> update -> refresh -> no-op
// plan -> import -> delete cycle against an actual Qualys subscription,
// using a real GCP service-account key that subscription is pre-authorised
// to add as a connector. It is the live-tenant counterpart to
// TestIntegrationGCPConnectorLifecycle (resource_gcp_connector_integration_test.go),
// which proves the same lifecycle against an in-memory mock and cannot, by
// itself, prove anything about Qualys's real API.
//
// This provider was NOT exercised against a live tenant while building it
// — no Qualys credentials were available. This test exists so that a
// maintainer who does have an authorised tenant can run it; it is not
// something this change claims to have executed or verified itself.
//
// Requires (in addition to TF_ACC=1 and QUALYS_URL/QUALYS_USERNAME/
// QUALYS_PASSWORD, checked by requireRealTenantConfig):
//
//   - QUALYS_TEST_GCP_CREDENTIALS_JSON: a real GCP service-account key
//     (the full JSON, not a file path), for a project the test Qualys
//     subscription is authorised to add a CloudView/Connector v3
//     connector against. Never commit a real value for this — it is read
//     from the environment only.
//
// Skips (not fails) when any of the above is unset, so enabling
// acceptance tests in general does not require this specific
// configuration to be present.
func TestAcceptanceGCPConnectorLifecycle(t *testing.T) {
	credsJSON := requireRealTenantConfig(t, "QUALYS_TEST_GCP_CREDENTIALS_JSON")[0]
	name := "tf-acc-gcp-" + randomSuffix(t)

	resource.Test(t, resource.TestCase{
		ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "qualys_gcp_connector" "test" {
						name                 = %q
						description          = "created by TestAcceptanceGCPConnectorLifecycle"
						gcp_credentials_json = %q
						run_frequency        = 240
					}
				`, name, credsJSON),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "name", name),
					resource.TestCheckResourceAttrSet("qualys_gcp_connector.test", "connector_id"),
					resource.TestCheckResourceAttrSet("qualys_gcp_connector.test", "project_id"),
				),
			},
			{
				// Update: change a field that doesn't require rotating
				// credentials, then confirm a refresh agrees with it.
				Config: fmt.Sprintf(`
					resource "qualys_gcp_connector" "test" {
						name                 = %q
						description          = "updated by TestAcceptanceGCPConnectorLifecycle"
						gcp_credentials_json = %q
						run_frequency        = 480
					}
				`, name, credsJSON),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "description",
						"updated by TestAcceptanceGCPConnectorLifecycle"),
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "run_frequency", "480"),
				),
			},
			{
				// No-op plan: re-applying identical config against the real
				// API must not diff — the thing an in-memory mock cannot
				// prove, since it can only ever agree with itself.
				Config: fmt.Sprintf(`
					resource "qualys_gcp_connector" "test" {
						name                 = %q
						description          = "updated by TestAcceptanceGCPConnectorLifecycle"
						gcp_credentials_json = %q
						run_frequency        = 480
					}
				`, name, credsJSON),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:            "qualys_gcp_connector.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"gcp_credentials_json"}, // write-only, never re-read
			},
		},
		// No CheckDestroy: the real Delete call in the last step's implicit
		// cleanup already proves deletion succeeds against the live API,
		// which is what this test is for; a further existence check would
		// just be re-testing Read, already covered by every refresh above.
	})
}
