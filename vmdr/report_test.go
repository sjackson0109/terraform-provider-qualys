package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestListReportsSendsConfirmedFiltersAndParsesFields(t *testing.T) {
	var gotForm url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.Form
		fmt.Fprint(w, `<REPORT_LIST_OUTPUT><RESPONSE><REPORT_LIST>
			<REPORT><ID>9001</ID><TITLE>Q1 Vulnerabilities</TITLE><TYPE>Scan</TYPE>
			<USER_LOGIN>acct1</USER_LOGIN><LAUNCH_DATETIME>2026-01-01T00:00:00Z</LAUNCH_DATETIME>
			<OUTPUT_FORMAT>pdf</OUTPUT_FORMAT><SIZE>2MB</SIZE>
			<STATUS><STATE>Finished</STATE></STATUS>
			<EXPIRATION_DATETIME>2026-02-01T00:00:00Z</EXPIRATION_DATETIME></REPORT>
			</REPORT_LIST></RESPONSE></REPORT_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	reports, err := c.ListReports(context.Background(), ReportFilter{ID: "9001", State: "Finished"})
	if err != nil {
		t.Fatalf("ListReports: %v", err)
	}
	if gotForm.Get("id") != "9001" || gotForm.Get("state") != "Finished" {
		t.Errorf("form = %v", gotForm)
	}
	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}
	r := reports[0]
	if r.ID != "9001" || r.Title != "Q1 Vulnerabilities" || r.Type != "Scan" ||
		r.UserLogin != "acct1" || r.OutputFormat != "pdf" || r.Size != "2MB" ||
		r.Status != "Finished" || r.ExpirationDatetime != "2026-02-01T00:00:00Z" {
		t.Errorf("report = %+v", r)
	}
}
