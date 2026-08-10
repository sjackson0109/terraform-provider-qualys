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

// hasIDCursor reports whether a decoded ServiceRequest body carries the
// "id GREATER <n>" pagination cursor SearchAll adds to every page after
// the first.
func hasIDCursor(body map[string]interface{}) bool {
	sreq, _ := body["ServiceRequest"].(map[string]interface{})
	filters, _ := sreq["filters"].(map[string]interface{})
	criteria, _ := filters["Criteria"].([]interface{})
	for _, raw := range criteria {
		c, _ := raw.(map[string]interface{})
		if c["field"] == "id" && c["operator"] == "GREATER" {
			return true
		}
	}
	return false
}

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
	findings, conflicts, err := c.SearchWASFindings(context.Background(), WASFindingFilter{
		WebAppID: "555", Status: WASFindingStatusActive, IsIgnored: &ignored,
	})
	if err != nil {
		t.Fatalf("SearchWASFindings: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %+v", conflicts)
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

func TestRetestWASFindingSendsFindingID(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	if err := c.RetestWASFinding(context.Background(), "1728792"); err != nil {
		t.Fatalf("RetestWASFinding: %v", err)
	}
	if gotPath != "/qps/rest/3.0/retest/was/finding/1728792" {
		t.Errorf("path = %q", gotPath)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	finding, _ := data["Finding"].(map[string]interface{})
	if finding["id"] != float64(1728792) {
		t.Errorf("data.Finding.id = %v", finding["id"])
	}
}

func TestGetWASFindingRetestStatusDecodesRetestDetail(t *testing.T) {
	var gotPath string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"Finding":{"id":2774812,"uniqueId":"af45db08-80c6-4527-a48a-9759450b21a2",
		  "retest":{"retestStatus":"RETESTED","retestedDate":"2020-10-30T09:03:11Z",
		  "findingStatus":"Finding has been detected","reason":"Finding was confirmed"}}}]}}`)
	}))
	defer srv.Close()

	status, err := c.GetWASFindingRetestStatus(context.Background(), "2774812")
	if err != nil {
		t.Fatalf("GetWASFindingRetestStatus: %v", err)
	}
	if gotPath != "/qps/rest/3.0/retestStatus/was/finding/2774812" {
		t.Errorf("path = %q", gotPath)
	}
	if status.RetestStatus != WASRetestStatusRetested || status.UniqueID != "af45db08-80c6-4527-a48a-9759450b21a2" ||
		status.Reason != "Finding was confirmed" {
		t.Errorf("status = %+v", status)
	}
}

func TestSearchWASFindingsSendsIDQIDAndDateCriteria(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"false","data":[]}}`)
	}))
	defer srv.Close()

	_, _, err := c.SearchWASFindings(context.Background(), WASFindingFilter{
		ID:                 "1",
		QID:                "150001",
		FirstDetectedAfter: "2026-01-01T00:00:00Z",
		LastDetectedBefore: "2026-12-31T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("SearchWASFindings: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	filters, _ := sreq["filters"].(map[string]interface{})
	criteria, _ := filters["Criteria"].([]interface{})
	if len(criteria) != 4 {
		t.Fatalf("expected 4 filter criteria (id, qid, firstDetectedDate GREATER, lastDetectedDate LESSER), got %v", criteria)
	}

	var sawGreater, sawLesser bool
	for _, raw := range criteria {
		c, _ := raw.(map[string]interface{})
		if c["field"] == "firstDetectedDate" && c["operator"] == "GREATER" {
			sawGreater = true
		}
		if c["field"] == "lastDetectedDate" && c["operator"] == "LESSER" {
			sawLesser = true
		}
	}
	if !sawGreater || !sawLesser {
		t.Errorf("criteria = %v, want GREATER on firstDetectedDate and LESSER on lastDetectedDate", criteria)
	}
}

// An exact duplicate finding returned on both sides of a pagination
// boundary must collapse to one, not appear twice.
func TestSearchWASFindingsCollapsesExactDuplicateAcrossPages(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var body map[string]interface{}
		_ = json.Unmarshal(b, &body)
		if !hasIDCursor(body) {
			fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"true","lastId":1,
			  "data":[{"Finding":{"id":1,"qid":150001,"severity":3}}]}}`)
			return
		}
		// Same finding repeated verbatim on page 2 (a legitimate overlap
		// pattern for id-cursor pagination), then the true final record.
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"false",
		  "data":[{"Finding":{"id":1,"qid":150001,"severity":3}},{"Finding":{"id":2,"qid":150002,"severity":2}}]}}`)
	}))
	defer srv.Close()

	findings, conflicts, err := c.SearchWASFindings(context.Background(), WASFindingFilter{})
	if err != nil {
		t.Fatalf("SearchWASFindings: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2 (the id=1 duplicate collapsed)", len(findings))
	}
	if len(conflicts) != 0 {
		t.Fatalf("an exact duplicate must not be reported as a conflict: %+v", conflicts)
	}
}

// Two records sharing an ID but disagreeing on other fields (e.g. status
// changed between pages) must be flagged, not silently merged.
func TestSearchWASFindingsReportsConflictOnDisagreement(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var body map[string]interface{}
		_ = json.Unmarshal(b, &body)
		if !hasIDCursor(body) {
			fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"true","lastId":1,
			  "data":[{"Finding":{"id":1,"qid":150001,"status":"NEW"}}]}}`)
			return
		}
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"false",
		  "data":[{"Finding":{"id":1,"qid":150001,"status":"FIXED"}}]}}`)
	}))
	defer srv.Close()

	findings, conflicts, err := c.SearchWASFindings(context.Background(), WASFindingFilter{})
	if err != nil {
		t.Fatalf("SearchWASFindings: %v", err)
	}
	if len(findings) != 1 || findings[0].Status != "NEW" {
		t.Fatalf("findings = %+v, want 1 with the first-seen status kept", findings)
	}
	if len(conflicts) != 1 || conflicts[0].ID != "1" || conflicts[0].First.Status != "NEW" || conflicts[0].Other.Status != "FIXED" {
		t.Fatalf("conflicts = %+v", conflicts)
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

	if _, _, err := c.SearchWASFindings(context.Background(), WASFindingFilter{}); err != nil {
		t.Fatalf("SearchWASFindings: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	if _, present := sreq["filters"]; present {
		t.Errorf("filters should be omitted entirely when no criteria are set, got %v", sreq["filters"])
	}
}
