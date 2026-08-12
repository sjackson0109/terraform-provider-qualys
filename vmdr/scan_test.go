package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestListScansSendsConfirmedFiltersAndParsesFields(t *testing.T) {
	var gotForm url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.Form
		fmt.Fprint(w, `<SCAN_LIST_OUTPUT><RESPONSE><SCAN_LIST>
			<SCAN><REF>scan/12345.6789</REF><TITLE>Nightly</TITLE><TYPE>Scheduled</TYPE>
			<USER_LOGIN>acct1</USER_LOGIN><LAUNCH_DATETIME>2026-01-01T00:00:00Z</LAUNCH_DATETIME>
			<DURATION>01:02:03</DURATION><TARGET>10.0.0.0/24</TARGET><PROCESSED>1</PROCESSED>
			<STATUS><STATE>Finished</STATE></STATUS></SCAN>
			</SCAN_LIST></RESPONSE></SCAN_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	processed := true
	scans, err := c.ListScans(context.Background(), ScanFilter{
		State: "Finished", Type: "Scheduled", Target: "10.0.0.0/24",
		UserLogin: "acct1", LaunchedAfter: "2026-01-01T00:00:00Z",
		LaunchedBefore: "2026-02-01T00:00:00Z", Processed: &processed,
	})
	if err != nil {
		t.Fatalf("ListScans: %v", err)
	}
	if gotForm.Get("state") != "Finished" || gotForm.Get("type") != "Scheduled" ||
		gotForm.Get("target") != "10.0.0.0/24" || gotForm.Get("user_login") != "acct1" ||
		gotForm.Get("launched_after_datetime") != "2026-01-01T00:00:00Z" ||
		gotForm.Get("launched_before_datetime") != "2026-02-01T00:00:00Z" ||
		gotForm.Get("processed") != "1" {
		t.Errorf("form = %v", gotForm)
	}
	if len(scans) != 1 {
		t.Fatalf("got %d scans, want 1", len(scans))
	}
	s := scans[0]
	if s.Ref != "scan/12345.6789" || s.Title != "Nightly" || s.Type != "Scheduled" ||
		s.UserLogin != "acct1" || s.Duration != "01:02:03" || s.Target != "10.0.0.0/24" ||
		!s.Processed || s.Status != "Finished" {
		t.Errorf("scan = %+v", s)
	}
}
