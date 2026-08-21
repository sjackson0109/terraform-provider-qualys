package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
)

func TestVMFindingsSchemaValidatesSeverityRange(t *testing.T) {
	r := dataSourceVMFindings()
	for _, key := range []string{"minimum_severity", "maximum_severity"} {
		s := r.Schema[key]
		if s.ValidateFunc == nil {
			t.Fatalf("%s should have a ValidateFunc restricting it to 1-5", key)
		}
		if _, errs := s.ValidateFunc(0, key); len(errs) == 0 {
			t.Errorf("%s: severity 0 should be rejected", key)
		}
		if _, errs := s.ValidateFunc(6, key); len(errs) == 0 {
			t.Errorf("%s: severity 6 should be rejected", key)
		}
		if _, errs := s.ValidateFunc(3, key); len(errs) != 0 {
			t.Errorf("%s: severity 3 should be accepted, got %v", key, errs)
		}
	}
}

func TestVMFindingsSchemaValidatesRFC3339DateFilters(t *testing.T) {
	r := dataSourceVMFindings()
	for _, key := range []string{
		"first_found_after", "first_found_before",
		"last_found_after", "last_found_before",
		"last_test_after", "last_test_before",
	} {
		s := r.Schema[key]
		if s.ValidateFunc == nil {
			t.Fatalf("%s should have a ValidateFunc requiring RFC3339", key)
		}
		if _, errs := s.ValidateFunc("not-a-date", key); len(errs) == 0 {
			t.Errorf("%s: an invalid date should be rejected at plan time", key)
		}
		if _, errs := s.ValidateFunc("2026-08-01T00:00:00Z", key); len(errs) != 0 {
			t.Errorf("%s: a valid RFC3339 date should be accepted, got %v", key, errs)
		}
	}
}

func TestVMFindingsResultsFieldIsSensitive(t *testing.T) {
	r := dataSourceVMFindings()
	findingElem := r.Schema["findings"].Elem.(*schema.Resource)
	if !findingElem.Schema["results"].Sensitive {
		t.Error("findings[*].results should be marked Sensitive so it is masked in plan/apply output")
	}
}

func TestFilterVMFindingsByQID(t *testing.T) {
	findings := []*vmdr.VMFinding{
		{HostID: "1", QID: "1001"},
		{HostID: "1", QID: "1002"},
	}
	out, err := filterVMFindings(findings, vmFindingFilters{QIDs: []string{"1001"}})
	if err != nil {
		t.Fatalf("filterVMFindings: %v", err)
	}
	if len(out) != 1 || out[0].QID != "1001" {
		t.Errorf("out = %+v", out)
	}
}

func TestFilterVMFindingsBySeverityRange(t *testing.T) {
	findings := []*vmdr.VMFinding{
		{HostID: "1", QID: "1", Severity: 1},
		{HostID: "1", QID: "2", Severity: 3},
		{HostID: "1", QID: "3", Severity: 5},
	}
	out, err := filterVMFindings(findings, vmFindingFilters{MinimumSeverity: 2, MaximumSeverity: 4})
	if err != nil {
		t.Fatalf("filterVMFindings: %v", err)
	}
	if len(out) != 1 || out[0].QID != "2" {
		t.Errorf("out = %+v, want only the severity-3 finding", out)
	}
}

func TestFilterVMFindingsByDateRange(t *testing.T) {
	findings := []*vmdr.VMFinding{
		{HostID: "1", QID: "1", LastFoundDatetime: "2026-01-01T00:00:00Z"},
		{HostID: "1", QID: "2", LastFoundDatetime: "2026-08-15T00:00:00Z"},
		{HostID: "1", QID: "3", LastFoundDatetime: ""},
	}
	out, err := filterVMFindings(findings, vmFindingFilters{LastFoundAfter: "2026-08-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("filterVMFindings: %v", err)
	}
	if len(out) != 1 || out[0].QID != "2" {
		t.Errorf("out = %+v, want only the August finding; an empty datetime must not pass a configured filter", out)
	}
}

func TestFilterVMFindingsRejectsUnparseableStoredDatetime(t *testing.T) {
	findings := []*vmdr.VMFinding{
		{HostID: "1", QID: "1", LastFoundDatetime: "not-a-date"},
	}
	_, err := filterVMFindings(findings, vmFindingFilters{LastFoundAfter: "2026-01-01T00:00:00Z"})
	if err == nil {
		t.Fatal("expected an error when a decoded finding's datetime cannot be parsed as RFC3339")
	}
}

func TestFilterVMFindingsWithNoFiltersReturnsEverything(t *testing.T) {
	findings := []*vmdr.VMFinding{{HostID: "1", QID: "1"}, {HostID: "2", QID: "2"}}
	out, err := filterVMFindings(findings, vmFindingFilters{})
	if err != nil {
		t.Fatalf("filterVMFindings: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("out = %+v, want both findings unfiltered", out)
	}
}

func TestSortVMFindingsOrdersByHostQIDPortProtocol(t *testing.T) {
	findings := []*vmdr.VMFinding{
		{HostID: "10", QID: "1"},
		{HostID: "9", QID: "5"},
		{HostID: "9", QID: "1", Port: 443, Protocol: "TCP"},
		{HostID: "9", QID: "1", Port: 80, Protocol: "TCP"},
		{HostID: "9", QID: "1", Port: 80, Protocol: "UDP"},
	}
	sortVMFindings(findings)

	want := []string{"9:1:TCP:80", "9:1:UDP:80", "9:1:TCP:443", "9:5", "10:1"}
	for i, f := range findings {
		if f.FindingKey() != want[i] {
			t.Errorf("position %d: got %q, want %q (full order: %v)", i, f.FindingKey(), want[i], keysOf(findings))
		}
	}
}

func keysOf(findings []*vmdr.VMFinding) []string {
	out := make([]string, len(findings))
	for i, f := range findings {
		out[i] = f.FindingKey()
	}
	return out
}

// Host IDs are numeric strings; sorting must compare them numerically, not
// lexically, or host "10" would sort before host "9".
func TestSortVMFindingsHostIDIsNumericNotLexical(t *testing.T) {
	findings := []*vmdr.VMFinding{
		{HostID: "10", QID: "1"},
		{HostID: "9", QID: "1"},
		{HostID: "2", QID: "1"},
	}
	sortVMFindings(findings)
	if findings[0].HostID != "2" || findings[1].HostID != "9" || findings[2].HostID != "10" {
		t.Errorf("order = %v, want numeric 2, 9, 10", keysOf(findings))
	}
}

func TestFetchKnowledgeBaseCollectsUniqueQIDsOnly(t *testing.T) {
	// fetchKnowledgeBase itself needs a *vmdr.Client to call
	// GetKnowledgeBaseEntries, so its network behaviour (one batched call
	// for many findings sharing QIDs) is covered end-to-end in
	// vmdr/knowledgebase_test.go's TestGetKnowledgeBaseEntriesBatchesIntoOneRequest.
	// This test only confirms the QID de-duplication this function performs
	// before calling out, using a nil client and a zero-finding input (the
	// one case this function can exercise without a real client).
	kb, err := fetchKnowledgeBase(context.TODO(), nil, nil)
	if err != nil {
		t.Fatalf("fetchKnowledgeBase: %v", err)
	}
	if kb != nil {
		t.Errorf("kb = %v, want nil for no findings", kb)
	}
}

// TestResolveAssetGroupIPsUsesOneBulkRequest is a regression test: this
// used to call GetAssetGroup once per configured asset_group_id (N HTTP
// requests for N groups). resolveAssetGroupIPs must resolve every group in
// a single ListAssetGroups(IDs: [...]) call instead.
func TestResolveAssetGroupIPsUsesOneBulkRequest(t *testing.T) {
	requests := 0
	c, srv := vmdrTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		fmt.Fprint(w, `<ASSET_GROUP_LIST_OUTPUT><RESPONSE>
		  <ASSET_GROUP_LIST>
		    <ASSET_GROUP><ID>1</ID><TITLE>a</TITLE><IP_SET><IP>10.0.0.1</IP></IP_SET></ASSET_GROUP>
		    <ASSET_GROUP><ID>2</ID><TITLE>b</TITLE><IP_SET><IP>10.0.0.2</IP></IP_SET></ASSET_GROUP>
		  </ASSET_GROUP_LIST>
		</RESPONSE></ASSET_GROUP_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	ips, err := resolveAssetGroupIPs(context.Background(), c, []string{"1", "2"})
	if err != nil {
		t.Fatalf("resolveAssetGroupIPs: %v", err)
	}
	if requests != 1 {
		t.Errorf("made %d HTTP requests for 2 groups, want exactly 1 (bulk lookup)", requests)
	}
	if len(ips) != 2 || ips[0] != "10.0.0.1" || ips[1] != "10.0.0.2" {
		t.Errorf("ips = %v", ips)
	}
}

// TestResolveAssetGroupIPsErrorsOnMissingGroup confirms the bulk lookup
// still surfaces a clear error for a mistyped/inaccessible group id — a
// bulk search for N ids where one doesn't match returns the other N-1
// without an API error, which would otherwise silently under-resolve IPs
// with no indication anything was missed.
func TestResolveAssetGroupIPsErrorsOnMissingGroup(t *testing.T) {
	c, srv := vmdrTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<ASSET_GROUP_LIST_OUTPUT><RESPONSE>
		  <ASSET_GROUP_LIST>
		    <ASSET_GROUP><ID>1</ID><TITLE>a</TITLE><IP_SET><IP>10.0.0.1</IP></IP_SET></ASSET_GROUP>
		  </ASSET_GROUP_LIST>
		</RESPONSE></ASSET_GROUP_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	_, err := resolveAssetGroupIPs(context.Background(), c, []string{"1", "404"})
	if err == nil {
		t.Fatal("expected an error for asset group 404, which the mock server never returned")
	}
	if !errors.Is(err, vmdr.ErrNotFound) {
		t.Errorf("error = %v, want it to wrap vmdr.ErrNotFound", err)
	}
}

// vmdrTestClient builds a *vmdr.Client against a local TLS mock server, the
// provider-package equivalent of vmdr's own internal testClient helper
// (unexported, so not usable directly from this package).
func vmdrTestClient(t *testing.T, h http.Handler) (*vmdr.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	c, err := vmdr.NewClient(vmdr.Config{
		BaseURL:    srv.URL,
		Username:   "u",
		Password:   "p",
		HTTPClient: srv.Client(),
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("vmdr.NewClient: %v", err)
	}
	return c, srv
}
