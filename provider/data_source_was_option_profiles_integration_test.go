package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// mockWASOptionProfileSearchServer is a stateless, in-memory implementation
// of just the WAS optionprofile search endpoint (qps/wasoptionprofile.go's
// SearchWASOptionProfiles), returning a fixed set of profiles on a single
// page (hasMoreRecords omitted/false) — enough to drive
// dataSourceWASOptionProfilesRead through a real Terraform read.
type mockWASOptionProfileSearchServer struct{}

func (m *mockWASOptionProfileSearchServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost || r.URL.Path != "/qps/rest/3.0/search/was/optionprofile" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"UNSUPPORTED_URL"}}`)
		return
	}

	fmt.Fprint(w, `{
		"ServiceResponse": {
			"responseCode": "SUCCESS",
			"count": 2,
			"data": [
				{"OptionProfile": {"id": "101", "name": "Initial WAS Options", "comments": "shipped by default",
					"maxCrawlRequests": 500, "performance": "MEDIUM", "bruteforceOption": "STANDARD",
					"timeoutErrorThreshold": 10, "unexpectedErrorThreshold": 5,
					"sensitiveContent": {"creditCardNumber": true, "socialSecurityNumber": false}}},
				{"OptionProfile": {"id": "202", "name": "Aggressive Scan", "comments": "",
					"maxCrawlRequests": 2000, "performance": "HIGH", "bruteforceOption": "MINIMAL",
					"timeoutErrorThreshold": 20, "unexpectedErrorThreshold": 10,
					"sensitiveContent": {"creditCardNumber": false, "socialSecurityNumber": true}}}
			]
		}
	}`)
}

// TestIntegrationWASOptionProfilesDataSource drives
// data.qualys_was_option_profiles through a real Terraform read against an
// in-memory mock of the WAS optionprofile search endpoint, confirming both
// the unfiltered list and the name_contains client-side filter.
//
// This is an INTEGRATION test, not an acceptance test — see
// TestIntegrationWASOptionProfileLifecycle's doc comment for what that
// distinction means.
func TestIntegrationWASOptionProfilesDataSource(t *testing.T) {
	srv := httptest.NewTLSServer(&mockWASOptionProfileSearchServer{})
	defer srv.Close()
	accMockServerEnv(t, srv)

	resource.Test(t, resource.TestCase{
		ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: `
					data "qualys_was_option_profiles" "all" {}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.qualys_was_option_profiles.all", "option_profiles.#", "2"),
					resource.TestCheckResourceAttr("data.qualys_was_option_profiles.all", "option_profiles.0.id", "101"),
					resource.TestCheckResourceAttr("data.qualys_was_option_profiles.all", "option_profiles.0.name", "Initial WAS Options"),
					resource.TestCheckResourceAttr("data.qualys_was_option_profiles.all", "option_profiles.0.max_crawl_requests", "500"),
					resource.TestCheckResourceAttr("data.qualys_was_option_profiles.all", "option_profiles.0.detect_credit_card_numbers", "true"),
					resource.TestCheckResourceAttr("data.qualys_was_option_profiles.all", "option_profiles.1.id", "202"),
					resource.TestCheckResourceAttr("data.qualys_was_option_profiles.all", "option_profiles.1.name", "Aggressive Scan"),
				),
			},
			{
				Config: `
					data "qualys_was_option_profiles" "filtered" {
						name_contains = "initial"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.qualys_was_option_profiles.filtered", "option_profiles.#", "1"),
					resource.TestCheckResourceAttr("data.qualys_was_option_profiles.filtered", "option_profiles.0.id", "101"),
				),
			},
		},
	})
}
