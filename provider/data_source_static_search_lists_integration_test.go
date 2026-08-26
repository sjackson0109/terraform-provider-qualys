package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// mockStaticSearchListServer is a stateless, in-memory implementation of
// just the VM/PA static search list list endpoint (vmdr/searchlist.go's
// ListStaticSearchLists), returning a fixed pair of lists on a single page
// — enough to drive dataSourceStaticSearchListsRead through a real
// Terraform read.
type mockStaticSearchListServer struct{}

func (m *mockStaticSearchListServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.URL.Path, "qid/search_list/static/") {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>unsupported path</TEXT></RESPONSE></SIMPLE_RETURN>`)
		return
	}
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>bad request</TEXT></RESPONSE></SIMPLE_RETURN>`)
		return
	}
	if r.Form.Get("action") != "list" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<SIMPLE_RETURN><RESPONSE><TEXT>unsupported action %q</TEXT></RESPONSE></SIMPLE_RETURN>`, r.Form.Get("action"))
		return
	}

	fmt.Fprint(w, `<SEARCH_LIST_OUTPUT><RESPONSE><STATIC_LISTS>
		<STATIC_LIST>
			<ID>501</ID><TITLE>PCI QIDs</TITLE><GLOBAL>1</GLOBAL>
			<COMMENTS>PCI DSS relevant vulnerabilities</COMMENTS>
			<QIDS><QID>12345</QID><QID>23456</QID></QIDS>
			<CREATED><DATETIME>2026-01-01T00:00:00Z</DATETIME></CREATED>
			<MODIFIED><DATETIME>2026-01-01T00:00:00Z</DATETIME></MODIFIED>
		</STATIC_LIST>
		<STATIC_LIST>
			<ID>502</ID><TITLE>Excluded Noise</TITLE><GLOBAL>0</GLOBAL>
			<COMMENTS></COMMENTS>
			<QIDS><QID>99999</QID></QIDS>
			<CREATED><DATETIME>2026-01-02T00:00:00Z</DATETIME></CREATED>
			<MODIFIED><DATETIME>2026-01-02T00:00:00Z</DATETIME></MODIFIED>
		</STATIC_LIST>
	</STATIC_LISTS></RESPONSE></SEARCH_LIST_OUTPUT>`)
}

// TestIntegrationStaticSearchListsDataSource drives
// data.qualys_static_search_lists through a real Terraform read against an
// in-memory mock of the VM/PA static search list list endpoint, confirming
// both the unfiltered list and the title_contains client-side filter.
//
// This is an INTEGRATION test, not an acceptance test — see
// TestIntegrationWASOptionProfileLifecycle's doc comment for what that
// distinction means.
func TestIntegrationStaticSearchListsDataSource(t *testing.T) {
	srv := httptest.NewTLSServer(&mockStaticSearchListServer{})
	defer srv.Close()
	accMockServerEnv(t, srv)

	resource.Test(t, resource.TestCase{
		ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: `
					data "qualys_static_search_lists" "all" {}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.qualys_static_search_lists.all", "search_lists.#", "2"),
					resource.TestCheckResourceAttr("data.qualys_static_search_lists.all", "search_lists.0.id", "501"),
					resource.TestCheckResourceAttr("data.qualys_static_search_lists.all", "search_lists.0.title", "PCI QIDs"),
					resource.TestCheckResourceAttr("data.qualys_static_search_lists.all", "search_lists.0.global", "true"),
					resource.TestCheckResourceAttr("data.qualys_static_search_lists.all", "search_lists.0.qids.#", "2"),
					resource.TestCheckResourceAttr("data.qualys_static_search_lists.all", "search_lists.1.id", "502"),
					resource.TestCheckResourceAttr("data.qualys_static_search_lists.all", "search_lists.1.title", "Excluded Noise"),
					resource.TestCheckResourceAttr("data.qualys_static_search_lists.all", "search_lists.1.global", "false"),
				),
			},
			{
				Config: `
					data "qualys_static_search_lists" "filtered" {
						title_contains = "pci"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.qualys_static_search_lists.filtered", "search_lists.#", "1"),
					resource.TestCheckResourceAttr("data.qualys_static_search_lists.filtered", "search_lists.0.id", "501"),
				),
			},
		},
	})
}
