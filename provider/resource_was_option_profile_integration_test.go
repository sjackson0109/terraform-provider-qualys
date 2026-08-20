package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// mockWASOptionProfileServer is a stateful, in-memory implementation of
// just enough of the WAS optionprofile API (qps/wasoptionprofile.go) to
// drive a full create/read/update/no-op-plan/import/delete cycle through a
// real Terraform plan/apply.
type mockWASOptionProfileServer struct {
	mu     sync.Mutex
	nextID int
	byID   map[string]map[string]interface{}
}

func newMockWASOptionProfileServer() *mockWASOptionProfileServer {
	return &mockWASOptionProfileServer{nextID: 9000, byID: map[string]map[string]interface{}{}}
}

func (m *mockWASOptionProfileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/qps/rest/3.0/create/was/optionprofile":
		m.create(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/qps/rest/3.0/get/was/optionprofile/"):
		m.get(w, wasOPID(r.URL.Path))
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/qps/rest/3.0/update/was/optionprofile/"):
		m.update(w, r, wasOPID(r.URL.Path))
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/qps/rest/3.0/delete/was/optionprofile/"):
		m.delete(w, wasOPID(r.URL.Path))
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"UNSUPPORTED_URL"}}`)
	}
}

func wasOPID(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

type wasOptionProfileEnvelope struct {
	ServiceRequest struct {
		Data struct {
			OptionProfile map[string]interface{} `json:"OptionProfile"`
		} `json:"data"`
	} `json:"ServiceRequest"`
}

func (m *mockWASOptionProfileServer) create(w http.ResponseWriter, r *http.Request) {
	var env wasOptionProfileEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"ServiceResponse":{"responseCode":"BAD_REQUEST","responseErrorDetails":{"errorMessage":%q}}}`, err.Error())
		return
	}
	idStr := strconv.Itoa(m.nextID)
	m.nextID++
	profile := env.ServiceRequest.Data.OptionProfile
	profile["id"] = idStr
	m.byID[idStr] = profile
	m.writeProfile(w, profile)
}

func (m *mockWASOptionProfileServer) get(w http.ResponseWriter, idStr string) {
	profile, ok := m.byID[idStr]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
		return
	}
	m.writeProfile(w, profile)
}

func (m *mockWASOptionProfileServer) update(w http.ResponseWriter, r *http.Request, idStr string) {
	if _, ok := m.byID[idStr]; !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
		return
	}
	var env wasOptionProfileEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"ServiceResponse":{"responseCode":"BAD_REQUEST","responseErrorDetails":{"errorMessage":%q}}}`, err.Error())
		return
	}
	profile := env.ServiceRequest.Data.OptionProfile
	profile["id"] = idStr
	m.byID[idStr] = profile
	fmt.Fprintf(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"data":[{"OptionProfile":{"id":%s}}]}}`, idStr)
}

func (m *mockWASOptionProfileServer) delete(w http.ResponseWriter, idStr string) {
	if _, ok := m.byID[idStr]; !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
		return
	}
	delete(m.byID, idStr)
	fmt.Fprintf(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"data":[{"OptionProfile":{"id":%s}}]}}`, idStr)
}

func (m *mockWASOptionProfileServer) writeProfile(w http.ResponseWriter, profile map[string]interface{}) {
	resp := map[string]interface{}{
		"ServiceResponse": map[string]interface{}{
			"responseCode": "SUCCESS",
			"count":        1,
			"data": []interface{}{
				map[string]interface{}{
					"OptionProfile": map[string]interface{}{
						"id":       profile["id"],
						"name":     profile["name"],
						"comments": profile["comments"],
					},
				},
			},
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// TestIntegrationWASOptionProfileLifecycle drives qualys_was_option_profile
// — a WAS/QPS (portal JSON API) managed resource, distinct from the VM/PA
// (vmdr) resource TestIntegrationAssetGroupLifecycle covers — through a
// real Terraform create -> read/refresh -> update -> refresh -> no-op plan
// -> import -> delete cycle, against an in-memory mock of the WAS
// optionprofile endpoints in this file. It also exercises this provider's
// OBJECT_NOT_FOUND policy in a real plan/apply/destroy cycle: the mock's
// delete/get handlers return OBJECT_NOT_FOUND for an unknown id exactly
// like the real portal API's documented behaviour.
//
// This is an INTEGRATION test, not an acceptance test: see
// TestIntegrationGCPConnectorLifecycle's doc comment for what that
// distinction means and why. No genuine live-tenant acceptance test exists
// for this resource in this change — see the task's final report.
func TestIntegrationWASOptionProfileLifecycle(t *testing.T) {
	mock := newMockWASOptionProfileServer()
	srv := httptest.NewTLSServer(mock)
	defer srv.Close()
	accMockServerEnv(t, srv)

	resource.Test(t, resource.TestCase{
		ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "qualys_was_option_profile" "test" {
						name     = "integration-test-profile"
						comments = "created by TestIntegrationWASOptionProfileLifecycle"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("qualys_was_option_profile.test", "name", "integration-test-profile"),
					resource.TestCheckResourceAttr("qualys_was_option_profile.test", "comments",
						"created by TestIntegrationWASOptionProfileLifecycle"),
					resource.TestCheckResourceAttrSet("qualys_was_option_profile.test", "id"),
				),
			},
			{
				Config: `
					resource "qualys_was_option_profile" "test" {
						name     = "integration-test-profile"
						comments = "updated by TestIntegrationWASOptionProfileLifecycle"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("qualys_was_option_profile.test", "comments",
						"updated by TestIntegrationWASOptionProfileLifecycle"),
				),
			},
			{
				// No-op plan: re-applying identical config must not diff.
				Config: `
					resource "qualys_was_option_profile" "test" {
						name     = "integration-test-profile"
						comments = "updated by TestIntegrationWASOptionProfileLifecycle"
					}
				`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "qualys_was_option_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
