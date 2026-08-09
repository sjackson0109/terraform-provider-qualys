package qps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestSearchWASFindingsSendsFilters(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"false",
		  "data":[{"Finding":{"id":1,"qid":150001,"name":"Reflected XSS","type":"VULNERABILITY",
		    "severity":3,"status":"ACTIVE","findingType":"QUALYS","url":"https://shop.example.com/search",
		    "webApp":{"id":555,"name":"storefront"},"cvssV3":{"base":6.1}}}]}}`)
	}))
	defer srv.Close()

	ignored := false
	findings, err := c.SearchWASFindings(context.Background(), WASFindingFilter{
		WebAppID: "555", Status: WASFindingStatusActive, IsIgnored: &ignored,
	})
	if err != nil {
		t.Fatalf("SearchWASFindings: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	f := findings[0]
	if f.ID != "1" || f.WebAppID != "555" || f.CVSSV3Base != 6.1 || f.WebAppName != "storefront" {
		t.Errorf("finding = %+v", f)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	filters, _ := sreq["filters"].(map[string]interface{})
	criteria, _ := filters["Criteria"].([]interface{})
	if len(criteria) != 3 {
		t.Fatalf("expected 3 filter criteria (webApp.id, status, isIgnored), got %v", criteria)
	}
}

func TestWASFindingNotFoundIsRecognised(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
	}))
	defer srv.Close()

	_, err := c.GetWASFinding(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestIgnoreWASFindingSendsCommentToDedicatedEndpoint(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	if err := c.IgnoreWASFinding(context.Background(), "9284112", "Accepted business risk."); err != nil {
		t.Fatalf("IgnoreWASFinding: %v", err)
	}
	if gotPath != "/qps/rest/3.0/ignore/was/finding/9284112" {
		t.Errorf("path = %q", gotPath)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	finding, _ := data["Finding"].(map[string]interface{})
	if finding["comment"] != "Accepted business risk." {
		t.Errorf("comment = %v", finding["comment"])
	}
}

func TestReopenAndFixWASFindingUseDedicatedEndpoints(t *testing.T) {
	var gotPaths []string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	if err := c.ReopenWASFinding(context.Background(), "9284112", "Reappeared."); err != nil {
		t.Fatalf("ReopenWASFinding: %v", err)
	}
	if err := c.FixWASFinding(context.Background(), "9284112", "Verified fixed."); err != nil {
		t.Fatalf("FixWASFinding: %v", err)
	}
	want := []string{
		"/qps/rest/3.0/reopen/was/finding/9284112",
		"/qps/rest/3.0/fix/was/finding/9284112",
	}
	if len(gotPaths) != 2 || gotPaths[0] != want[0] || gotPaths[1] != want[1] {
		t.Errorf("paths = %v, want %v", gotPaths, want)
	}
}

func TestSearchWASFindingsWithNoFilterOmitsFilters(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"false","data":[]}}`)
	}))
	defer srv.Close()

	if _, err := c.SearchWASFindings(context.Background(), WASFindingFilter{}); err != nil {
		t.Fatalf("SearchWASFindings: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	if _, present := sreq["filters"]; present {
		t.Errorf("filters should be omitted entirely when no criteria are set, got %v", sreq["filters"])
	}
}
