package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// mockWASAuthRecordSearchServer is a stateless, in-memory implementation of
// just the WAS webauthrecord search endpoint (qps/wasauthrecord.go's
// SearchWASAuthRecords), returning a fixed set of records on a single page
// — enough to drive dataSourceWASAuthRecordsRead through a real Terraform
// read. Credential blocks are never present in a search response (Qualys
// masks them), matching what the resource's own Read leaves unset.
type mockWASAuthRecordSearchServer struct{}

func (m *mockWASAuthRecordSearchServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost || r.URL.Path != "/qps/rest/3.0/search/was/webauthrecord" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"UNSUPPORTED_URL"}}`)
		return
	}

	fmt.Fprint(w, `{
		"ServiceResponse": {
			"responseCode": "SUCCESS",
			"count": 2,
			"data": [
				{"WebAppAuthRecord": {"id": "301", "name": "Staging Login", "comments": "form-based",
					"createdDate": "2026-01-01T00:00:00Z",
					"tags": {"list": {"TagSimple": [{"id": "9001", "name": "web"}]}}}},
				{"WebAppAuthRecord": {"id": "302", "name": "API Basic Auth", "comments": "",
					"createdDate": "2026-01-02T00:00:00Z"}}
			]
		}
	}`)
}

// TestIntegrationWASAuthRecordsDataSource drives data.qualys_was_auth_records
// through a real Terraform read against an in-memory mock of the WAS
// webauthrecord search endpoint, confirming both the unfiltered list and the
// name_contains client-side filter.
//
// This is an INTEGRATION test, not an acceptance test — see
// TestIntegrationWASOptionProfileLifecycle's doc comment for what that
// distinction means.
func TestIntegrationWASAuthRecordsDataSource(t *testing.T) {
	srv := httptest.NewTLSServer(&mockWASAuthRecordSearchServer{})
	defer srv.Close()
	accMockServerEnv(t, srv)

	resource.Test(t, resource.TestCase{
		ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: `
					data "qualys_was_auth_records" "all" {}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.qualys_was_auth_records.all", "auth_records.#", "2"),
					resource.TestCheckResourceAttr("data.qualys_was_auth_records.all", "auth_records.0.id", "301"),
					resource.TestCheckResourceAttr("data.qualys_was_auth_records.all", "auth_records.0.name", "Staging Login"),
					resource.TestCheckResourceAttr("data.qualys_was_auth_records.all", "auth_records.0.tag_ids.#", "1"),
					resource.TestCheckResourceAttr("data.qualys_was_auth_records.all", "auth_records.0.tag_ids.0", "9001"),
					resource.TestCheckResourceAttr("data.qualys_was_auth_records.all", "auth_records.1.id", "302"),
					resource.TestCheckResourceAttr("data.qualys_was_auth_records.all", "auth_records.1.name", "API Basic Auth"),
				),
			},
			{
				Config: `
					data "qualys_was_auth_records" "filtered" {
						name_contains = "staging"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.qualys_was_auth_records.filtered", "auth_records.#", "1"),
					resource.TestCheckResourceAttr("data.qualys_was_auth_records.filtered", "auth_records.0.id", "301"),
				),
			},
		},
	})
}
