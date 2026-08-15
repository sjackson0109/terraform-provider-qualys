package provider

import (
	"testing"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestWASFindingsSchemaValidatesSeverity(t *testing.T) {
	r := dataSourceWASFindings()
	for _, key := range []string{"severity", "minimum_severity", "maximum_severity"} {
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

func TestWASFindingsSchemaValidatesRFC3339DateFilters(t *testing.T) {
	r := dataSourceWASFindings()
	for _, key := range []string{
		"first_detected_after", "first_detected_before",
		"last_detected_after", "last_detected_before",
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

func TestWASFindingsWebAppAndFindingIDFiltersConflict(t *testing.T) {
	r := dataSourceWASFindings()
	webAppID := r.Schema["web_app_id"]
	webAppIDs := r.Schema["web_app_ids"]
	findingIDs := r.Schema["finding_ids"]

	wantConflicts := func(got []string, want ...string) bool {
		if len(got) != len(want) {
			return false
		}
		set := make(map[string]bool, len(got))
		for _, g := range got {
			set[g] = true
		}
		for _, w := range want {
			if !set[w] {
				return false
			}
		}
		return true
	}

	if !wantConflicts(webAppID.ConflictsWith, "web_app_ids", "finding_ids") {
		t.Errorf("web_app_id.ConflictsWith = %v", webAppID.ConflictsWith)
	}
	if !wantConflicts(webAppIDs.ConflictsWith, "web_app_id", "finding_ids") {
		t.Errorf("web_app_ids.ConflictsWith = %v", webAppIDs.ConflictsWith)
	}
	if !wantConflicts(findingIDs.ConflictsWith, "web_app_id", "web_app_ids") {
		t.Errorf("finding_ids.ConflictsWith = %v", findingIDs.ConflictsWith)
	}
}

func TestWASFindingsSeverityIsNowNumeric(t *testing.T) {
	r := dataSourceWASFindings()
	if r.Schema["severity"].Type != schema.TypeInt {
		t.Errorf("severity should be TypeInt, got %s", r.Schema["severity"].Type)
	}
	for _, key := range []string{"status", "type", "finding_type", "qids", "web_app_ids", "finding_ids"} {
		if r.Schema[key].Type != schema.TypeSet {
			t.Errorf("%s should be TypeSet, got %s", key, r.Schema[key].Type)
		}
	}
}

func TestWASFindingsFindingKeyEqualsID(t *testing.T) {
	r := dataSourceWASFindings()
	findingElem := r.Schema["findings"].Elem.(*schema.Resource)
	if findingElem.Schema["finding_key"] == nil || findingElem.Schema["id"] == nil {
		t.Fatal("findings[*] should expose both finding_key and id")
	}
}

func TestFilterWASFindingsByQID(t *testing.T) {
	findings := []*qps.WASFinding{
		{ID: "1", QID: "150001"},
		{ID: "2", QID: "150002"},
	}
	out, err := filterWASFindings(findings, wasFindingFilters{QIDs: []string{"150001"}})
	if err != nil {
		t.Fatalf("filterWASFindings: %v", err)
	}
	if len(out) != 1 || out[0].QID != "150001" {
		t.Errorf("out = %+v", out)
	}
}

func TestFilterWASFindingsByStatusSet(t *testing.T) {
	findings := []*qps.WASFinding{
		{ID: "1", Status: "NEW"},
		{ID: "2", Status: "ACTIVE"},
		{ID: "3", Status: "FIXED"},
	}
	out, err := filterWASFindings(findings, wasFindingFilters{Statuses: []string{"NEW", "ACTIVE"}})
	if err != nil {
		t.Fatalf("filterWASFindings: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("out = %+v, want NEW and ACTIVE only", out)
	}
}

func TestFilterWASFindingsBySeverityRange(t *testing.T) {
	findings := []*qps.WASFinding{
		{ID: "1", Severity: 1},
		{ID: "2", Severity: 3},
		{ID: "3", Severity: 5},
	}
	out, err := filterWASFindings(findings, wasFindingFilters{MinimumSeverity: 2, MaximumSeverity: 4})
	if err != nil {
		t.Fatalf("filterWASFindings: %v", err)
	}
	if len(out) != 1 || out[0].ID != "2" {
		t.Errorf("out = %+v, want only the severity-3 finding", out)
	}
}

func TestFilterWASFindingsByDateRange(t *testing.T) {
	findings := []*qps.WASFinding{
		{ID: "1", LastDetectedDate: "2026-01-01T00:00:00Z"},
		{ID: "2", LastDetectedDate: "2026-08-15T00:00:00Z"},
		{ID: "3", LastDetectedDate: ""},
	}
	out, err := filterWASFindings(findings, wasFindingFilters{LastDetectedAfter: "2026-08-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("filterWASFindings: %v", err)
	}
	if len(out) != 1 || out[0].ID != "2" {
		t.Errorf("out = %+v, want only the August finding; an empty datetime must not pass a configured filter", out)
	}
}

func TestFilterWASFindingsTakesFractionalSecondsDatetimes(t *testing.T) {
	findings := []*qps.WASFinding{
		{ID: "1", LastDetectedDate: "2026-08-15T00:00:00.123Z"},
	}
	out, err := filterWASFindings(findings, wasFindingFilters{LastDetectedAfter: "2026-08-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("filterWASFindings: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("out = %+v, want the fractional-second timestamp to parse", out)
	}
}

func TestFilterWASFindingsRejectsUnparseableStoredDatetime(t *testing.T) {
	findings := []*qps.WASFinding{{ID: "1", LastDetectedDate: "not-a-date"}}
	_, err := filterWASFindings(findings, wasFindingFilters{LastDetectedAfter: "2026-01-01T00:00:00Z"})
	if err == nil {
		t.Fatal("expected an error when a decoded finding's datetime cannot be parsed")
	}
}

func TestFilterWASFindingsWithNoFiltersReturnsEverything(t *testing.T) {
	findings := []*qps.WASFinding{{ID: "1"}, {ID: "2"}}
	out, err := filterWASFindings(findings, wasFindingFilters{})
	if err != nil {
		t.Fatalf("filterWASFindings: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("out = %+v, want both findings unfiltered", out)
	}
}

func TestSortWASFindingsOrdersNumericallyByID(t *testing.T) {
	findings := []*qps.WASFinding{
		{ID: "10"},
		{ID: "9"},
		{ID: "2"},
	}
	sortWASFindings(findings)
	if findings[0].ID != "2" || findings[1].ID != "9" || findings[2].ID != "10" {
		ids := []string{findings[0].ID, findings[1].ID, findings[2].ID}
		t.Errorf("order = %v, want numeric 2, 9, 10", ids)
	}
}

func TestWASFindingConflictDiagnosticsBuildsOneWarningPerConflict(t *testing.T) {
	conflicts := []qps.WASFindingConflict{
		{ID: "1", First: &qps.WASFinding{ID: "1", Status: "NEW"}, Other: &qps.WASFinding{ID: "1", Status: "FIXED"}},
	}
	diags := wasFindingConflictDiagnostics(conflicts)
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(diags))
	}
	if diags[0].Severity != diag.Warning {
		t.Errorf("severity = %v, want diag.Warning", diags[0].Severity)
	}
}
