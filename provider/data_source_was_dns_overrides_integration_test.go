package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// mockWASDNSOverrideSearchServer is a stateless, in-memory implementation of
// just the WAS dnsoverride search endpoint (qps/wasdnsoverride.go's
// SearchWASDNSOverrides), returning a fixed set of records on a single page
// — enough to drive dataSourceWASDNSOverridesRead through a real Terraform
// read.
type mockWASDNSOverrideSearchServer struct{}

func (m *mockWASDNSOverrideSearchServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost || r.URL.Path != "/qps/rest/3.0/search/was/dnsoverride" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"UNSUPPORTED_URL"}}`)
		return
	}

	fmt.Fprint(w, `{
		"ServiceResponse": {
			"responseCode": "SUCCESS",
			"count": 2,
			"data": [
				{"DnsOverride": {"id": "401", "name": "Staging Override",
					"mappings": {"list": {"DnsMapping": [
						{"hostName": "app.example.com", "ipAddress": "10.0.0.5"}
					]}},
					"comments": {"list": ["points at the staging cluster"]},
					"tags": {"list": {"TagSimple": [{"id": "9001", "name": "web"}]}},
					"createdDate": "2026-01-01T00:00:00Z", "updatedDate": "2026-01-01T00:00:00Z"}},
				{"DnsOverride": {"id": "402", "name": "UAT Override",
					"mappings": {"list": {"DnsMapping": [
						{"hostName": "uat.example.com", "ipAddress": "10.0.0.6"}
					]}},
					"createdDate": "2026-01-02T00:00:00Z", "updatedDate": "2026-01-02T00:00:00Z"}}
			]
		}
	}`)
}

// TestIntegrationWASDNSOverridesDataSource drives
// data.qualys_was_dns_overrides through a real Terraform read against an
// in-memory mock of the WAS dnsoverride search endpoint, confirming both
// the unfiltered list (including nested mapping blocks) and the
// name_contains client-side filter.
//
// This is an INTEGRATION test, not an acceptance test — see
// TestIntegrationWASOptionProfileLifecycle's doc comment for what that
// distinction means.
func TestIntegrationWASDNSOverridesDataSource(t *testing.T) {
	srv := httptest.NewTLSServer(&mockWASDNSOverrideSearchServer{})
	defer srv.Close()
	accMockServerEnv(t, srv)

	resource.Test(t, resource.TestCase{
		ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: `
					data "qualys_was_dns_overrides" "all" {}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.qualys_was_dns_overrides.all", "dns_overrides.#", "2"),
					resource.TestCheckResourceAttr("data.qualys_was_dns_overrides.all", "dns_overrides.0.id", "401"),
					resource.TestCheckResourceAttr("data.qualys_was_dns_overrides.all", "dns_overrides.0.name", "Staging Override"),
					resource.TestCheckResourceAttr("data.qualys_was_dns_overrides.all", "dns_overrides.0.mapping.#", "1"),
					resource.TestCheckResourceAttr("data.qualys_was_dns_overrides.all", "dns_overrides.0.mapping.0.host_name", "app.example.com"),
					resource.TestCheckResourceAttr("data.qualys_was_dns_overrides.all", "dns_overrides.0.mapping.0.ip_address", "10.0.0.5"),
					resource.TestCheckResourceAttr("data.qualys_was_dns_overrides.all", "dns_overrides.1.id", "402"),
					resource.TestCheckResourceAttr("data.qualys_was_dns_overrides.all", "dns_overrides.1.name", "UAT Override"),
				),
			},
			{
				Config: `
					data "qualys_was_dns_overrides" "filtered" {
						name_contains = "staging"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.qualys_was_dns_overrides.filtered", "dns_overrides.#", "1"),
					resource.TestCheckResourceAttr("data.qualys_was_dns_overrides.filtered", "dns_overrides.0.id", "401"),
				),
			},
		},
	})
}
