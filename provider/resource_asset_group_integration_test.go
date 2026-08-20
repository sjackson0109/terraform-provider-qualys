package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// mockAssetGroupServer is a stateful, in-memory implementation of just
// enough of the VM/PA asset/group/ API (vmdr/assetgroup.go) to drive a
// full create/read/update/no-op-plan/import/delete cycle through a real
// Terraform plan/apply.
type mockAssetGroupServer struct {
	mu     sync.Mutex
	nextID int
	byID   map[string]url.Values
}

func newMockAssetGroupServer() *mockAssetGroupServer {
	return &mockAssetGroupServer{nextID: 5000, byID: map[string]url.Values{}}
}

func (m *mockAssetGroupServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !strings.Contains(r.URL.Path, "asset/group/") {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>unsupported path</TEXT></RESPONSE></SIMPLE_RETURN>`)
		return
	}
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>bad request</TEXT></RESPONSE></SIMPLE_RETURN>`)
		return
	}

	switch r.Form.Get("action") {
	case "add":
		m.add(w, r)
	case "edit":
		m.edit(w, r)
	case "delete":
		m.delete(w, r)
	case "list":
		m.list(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<SIMPLE_RETURN><RESPONSE><TEXT>unsupported action %q</TEXT></RESPONSE></SIMPLE_RETURN>`, r.Form.Get("action"))
	}
}

func (m *mockAssetGroupServer) add(w http.ResponseWriter, r *http.Request) {
	id := strconv.Itoa(m.nextID)
	m.nextID++
	stored := url.Values{}
	for k, v := range r.Form {
		stored[k] = v
	}
	m.byID[id] = stored
	fmt.Fprintf(w, `<SIMPLE_RETURN><RESPONSE><TEXT>Asset Group Added Successfully</TEXT>
	  <ITEM_LIST><ITEM><KEY>ID</KEY><VALUE>%s</VALUE></ITEM></ITEM_LIST>
	</RESPONSE></SIMPLE_RETURN>`, id)
}

func (m *mockAssetGroupServer) edit(w http.ResponseWriter, r *http.Request) {
	id := r.Form.Get("id")
	if _, ok := m.byID[id]; !ok {
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><CODE>999</CODE><TEXT>not found</TEXT></RESPONSE></SIMPLE_RETURN>`)
		return
	}
	stored := url.Values{}
	for k, v := range r.Form {
		stored[k] = v
	}
	m.byID[id] = stored
	fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>Asset Group Updated Successfully</TEXT></RESPONSE></SIMPLE_RETURN>`)
}

func (m *mockAssetGroupServer) delete(w http.ResponseWriter, r *http.Request) {
	id := r.Form.Get("id")
	if _, ok := m.byID[id]; !ok {
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><CODE>999</CODE><TEXT>not found</TEXT></RESPONSE></SIMPLE_RETURN>`)
		return
	}
	delete(m.byID, id)
	fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>Asset Group Deleted Successfully</TEXT></RESPONSE></SIMPLE_RETURN>`)
}

func (m *mockAssetGroupServer) list(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if idParam := r.Form.Get("ids"); idParam != "" {
		ids = strings.Split(idParam, ",")
	} else {
		for id := range m.byID {
			ids = append(ids, id)
		}
	}

	var b strings.Builder
	b.WriteString(`<ASSET_GROUP_LIST_OUTPUT><RESPONSE><ASSET_GROUP_LIST>`)
	for _, id := range ids {
		stored, ok := m.byID[id]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, `<ASSET_GROUP>
		  <ID>%s</ID><TITLE>%s</TITLE><OWNER_ID>11</OWNER_ID><UNIT_ID>0</UNIT_ID>
		  <NETWORK_IDS>0</NETWORK_IDS><COMMENTS>%s</COMMENTS>
		</ASSET_GROUP>`, id, stored.Get("title"), stored.Get("comments"))
	}
	b.WriteString(`</ASSET_GROUP_LIST></RESPONSE></ASSET_GROUP_LIST_OUTPUT>`)
	fmt.Fprint(w, b.String())
}

// TestIntegrationAssetGroupLifecycle drives qualys_asset_group — a VM/PA
// (vmdr, legacy XML API) managed resource, distinct from every other
// lifecycle test in this package, which all cover qps (portal JSON API)
// resources — through a real Terraform create -> read/refresh -> update ->
// refresh -> no-op plan -> import -> delete cycle, against an in-memory
// mock of the asset/group/ endpoints in this file.
//
// This is an INTEGRATION test, not an acceptance test: see
// TestIntegrationGCPConnectorLifecycle's doc comment for what that
// distinction means and why. No genuine live-tenant acceptance test exists
// for this resource in this change — see the task's final report.
func TestIntegrationAssetGroupLifecycle(t *testing.T) {
	mock := newMockAssetGroupServer()
	srv := httptest.NewTLSServer(mock)
	defer srv.Close()
	accMockServerEnv(t, srv.URL)

	resource.Test(t, resource.TestCase{
		ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "qualys_asset_group" "test" {
						title    = "integration-test-group"
						comments = "created by TestIntegrationAssetGroupLifecycle"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("qualys_asset_group.test", "title", "integration-test-group"),
					resource.TestCheckResourceAttr("qualys_asset_group.test", "comments",
						"created by TestIntegrationAssetGroupLifecycle"),
					resource.TestCheckResourceAttrSet("qualys_asset_group.test", "id"),
				),
			},
			{
				Config: `
					resource "qualys_asset_group" "test" {
						title    = "integration-test-group"
						comments = "updated by TestIntegrationAssetGroupLifecycle"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("qualys_asset_group.test", "comments",
						"updated by TestIntegrationAssetGroupLifecycle"),
				),
			},
			{
				// No-op plan: re-applying identical config must not diff.
				Config: `
					resource "qualys_asset_group" "test" {
						title    = "integration-test-group"
						comments = "updated by TestIntegrationAssetGroupLifecycle"
					}
				`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "qualys_asset_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
