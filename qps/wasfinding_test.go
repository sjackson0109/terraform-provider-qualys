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
