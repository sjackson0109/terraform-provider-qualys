package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// mockGCPConnectorServer is a stateful, in-memory implementation of just
// enough of the Connector v3 GCP API (doc 12) to drive a full
// create/read/update/no-op-plan/import/delete cycle through a real
// Terraform plan/apply. It is deliberately narrow: only the fields this
// provider's schema exposes are round-tripped.
type mockGCPConnectorServer struct {
	mu     sync.Mutex
	nextID int
	byID   map[string]map[string]interface{}
}

func newMockGCPConnectorServer() *mockGCPConnectorServer {
	return &mockGCPConnectorServer{nextID: 1000, byID: map[string]map[string]interface{}{}}
}

func (m *mockGCPConnectorServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPost && matchPath(r.URL.Path, "/qps/rest/3.0/create/am/gcpassetdataconnector"):
		m.create(w, r)
	case r.Method == http.MethodGet && hasPrefix(r.URL.Path, "/qps/rest/3.0/get/am/gcpassetdataconnector/"):
		m.get(w, id(r.URL.Path))
	case r.Method == http.MethodPost && hasPrefix(r.URL.Path, "/qps/rest/3.0/update/am/gcpassetdataconnector/"):
		m.update(w, r, id(r.URL.Path))
	case r.Method == http.MethodPost && hasPrefix(r.URL.Path, "/qps/rest/3.0/delete/am/gcpassetdataconnector/"):
		m.delete(w, id(r.URL.Path))
	case r.Method == http.MethodPost && matchPath(r.URL.Path, "/qps/rest/3.0/search/am/gcpassetdataconnector"):
		m.search(w)
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"UNSUPPORTED_URL"}}`)
	}
}

func matchPath(got, want string) bool { return got == want }
func hasPrefix(got, prefix string) bool {
	return len(got) >= len(prefix) && got[:len(prefix)] == prefix
}
func id(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

type gcpEnvelope struct {
	ServiceRequest struct {
		Data struct {
			GcpAssetDataConnector map[string]interface{} `json:"GcpAssetDataConnector"`
		} `json:"data"`
	} `json:"ServiceRequest"`
}

func (m *mockGCPConnectorServer) create(w http.ResponseWriter, r *http.Request) {
	var env gcpEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"ServiceResponse":{"responseCode":"BAD_REQUEST","responseErrorDetails":{"errorMessage":%q}}}`, err.Error())
		return
	}
	idStr := strconv.Itoa(m.nextID)
	m.nextID++
	conn := env.ServiceRequest.Data.GcpAssetDataConnector
	conn["id"] = idStr
	conn["connectorState"] = "FINISHED_SUCCESS"
	conn["cloudviewUuid"] = ""
	m.byID[idStr] = conn
	m.writeConnector(w, conn)
}

func (m *mockGCPConnectorServer) get(w http.ResponseWriter, idStr string) {
	conn, ok := m.byID[idStr]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
		return
	}
	m.writeConnector(w, conn)
}

func (m *mockGCPConnectorServer) update(w http.ResponseWriter, r *http.Request, idStr string) {
	if _, ok := m.byID[idStr]; !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
		return
	}
	var env gcpEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"ServiceResponse":{"responseCode":"BAD_REQUEST","responseErrorDetails":{"errorMessage":%q}}}`, err.Error())
		return
	}
	conn := env.ServiceRequest.Data.GcpAssetDataConnector
	conn["id"] = idStr
	conn["connectorState"] = "FINISHED_SUCCESS"
	conn["cloudviewUuid"] = ""
	m.byID[idStr] = conn
	fmt.Fprintf(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"data":[{"GcpAssetDataConnector":{"id":%q}}]}}`, idStr)
}

func (m *mockGCPConnectorServer) delete(w http.ResponseWriter, idStr string) {
	if _, ok := m.byID[idStr]; !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
		return
	}
	delete(m.byID, idStr)
	fmt.Fprintf(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"data":[{"GcpAssetDataConnector":{"id":%q}}]}}`, idStr)
}

func (m *mockGCPConnectorServer) search(w http.ResponseWriter) {
	fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":0,"hasMoreRecords":"false","data":[]}}`)
}

// writeConnector renders conn in the same shape the real v3 API's response
// samples use (doc 12): booleans as strings, collections nested under
// "list", authRecord echoing only projectId.
func (m *mockGCPConnectorServer) writeConnector(w http.ResponseWriter, conn map[string]interface{}) {
	activation, _ := conn["activation"].(map[string]interface{})
	var modules []interface{}
	if activation != nil {
		if set, ok := activation["set"].(map[string]interface{}); ok {
			modules, _ = set["ActivationModule"].([]interface{})
		}
	}

	projectID := ""
	if auth, ok := conn["authRecord"].(map[string]interface{}); ok {
		if v, ok := auth["project_id"].(string); ok {
			projectID = v
		}
	}

	resp := map[string]interface{}{
		"ServiceResponse": map[string]interface{}{
			"responseCode": "SUCCESS",
			"count":        1,
			"data": []interface{}{
				map[string]interface{}{
					"GcpAssetDataConnector": map[string]interface{}{
						"id":                   conn["id"],
						"name":                 conn["name"],
						"description":          conn["description"],
						"connectorState":       conn["connectorState"],
						"disabled":             fmt.Sprintf("%v", conn["disabled"] == true),
						"runFrequency":         conn["runFrequency"],
						"isRemediationEnabled": fmt.Sprintf("%v", conn["isRemediationEnabled"] == true),
						"cloudviewUuid":        conn["cloudviewUuid"],
						"activation":           map[string]interface{}{"list": map[string]interface{}{"ActivationModule": modules}},
						"authRecord":           map[string]interface{}{"projectId": projectID},
					},
				},
			},
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// TestIntegrationGCPConnectorLifecycle drives the GCP connector through a
// real Terraform create -> update (including rotating credentials to a
// different project without setting project_id explicitly, the regression
// this test exists for — see validateGCPProjectMatchesKey) -> no-op plan
// -> import -> delete cycle, using the real terraform binary and the real
// provider binary talking over the real plugin protocol.
//
// This is an INTEGRATION test, not an acceptance test proving live Qualys
// behaviour: the "API" on the other end is mockGCPConnectorServer, an
// in-memory stand-in this file defines, not a real Qualys subscription. It
// proves the full chain from Terraform configuration through this
// provider's schema, CRUD functions and wire encoding down to an HTTP
// request — and that a genuine terraform plan/apply/refresh/import/destroy
// cycle behaves correctly against whatever that HTTP layer returns. It
// proves nothing about whether Qualys's real API actually accepts or
// returns what mockGCPConnectorServer assumes; that assumption is exactly
// what remains unverified against a live tenant (see doc 12 and the
// task's final report). TestAcceptanceGCPConnectorLifecycle, in
// resource_gcp_connector_acceptance_test.go, is the genuine live-tenant
// counterpart.
//
// This deliberately does NOT set IsUnitTest, even though it needs no real
// Qualys credentials, and so only runs under an explicit TF_ACC=1 like a
// real-tenant test would. Reason: the provider runs as a separate child
// process under the real terraform binary this test shells out to, and
// that process does not share this test's *http.Client — it builds its
// own via qps.NewClient, which trusts the OS certificate store, not
// httptest.NewTLSServer's self-signed certificate. Left alone, that makes
// every step fail on TLS verification in this environment — not a reason
// to weaken the provider's HTTPS enforcement, but a reason to give the
// provider subprocess a trust root for this one test run; see
// accMockServerEnv / trustMockServerCertificate in acctest_helpers_test.go
// for how. IsUnitTest would make a TLS failure run — and break — on every
// plain `go test ./...` if that mechanism were ever unavailable (e.g. on
// Windows, where it deliberately skips instead — see
// trustMockServerCertificate), which is worse than a test that requires
// deliberate opt-in and fails clearly when run.
func TestIntegrationGCPConnectorLifecycle(t *testing.T) {
	// No IsUnitTest here (see doc comment above), so resource.Test's own
	// TF_ACC gate applies — this is a mock-backed integration test, not a
	// real-tenant acceptance test, so requireRealTenant (which demands
	// real QUALYS_* credentials) does not apply; accMockServerEnv below
	// sets its own.

	mock := newMockGCPConnectorServer()
	srv := httptest.NewTLSServer(mock)
	defer srv.Close()
	accMockServerEnv(t, srv)

	keyA := `{"type":"service_account","project_id":"project-a","private_key":"x"}`
	keyB := `{"type":"service_account","project_id":"project-b","private_key":"y"}`

	resource.Test(t, resource.TestCase{
		ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				// project_id deliberately unset: derived from the key.
				Config: fmt.Sprintf(`
					resource "qualys_gcp_connector" "test" {
						name                 = "acctest-gcp"
						gcp_credentials_json = %q
						run_frequency        = 240
					}
				`, keyA),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "name", "acctest-gcp"),
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "project_id", "project-a"),
				),
			},
			{
				// Regression case: rotate credentials to a different
				// project without ever touching project_id. Before the
				// fix, this failed at plan time comparing the stale
				// project-a state value against the new key's project-b.
				Config: fmt.Sprintf(`
					resource "qualys_gcp_connector" "test" {
						name                 = "acctest-gcp"
						gcp_credentials_json = %q
						run_frequency        = 240
					}
				`, keyB),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("qualys_gcp_connector.test", "project_id", "project-b"),
				),
			},
			{
				// No-op plan: re-applying identical config must not diff.
				Config: fmt.Sprintf(`
					resource "qualys_gcp_connector" "test" {
						name                 = "acctest-gcp"
						gcp_credentials_json = %q
						run_frequency        = 240
					}
				`, keyB),
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
	})
}
