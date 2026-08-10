package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestListVMFindingsOneHostOneFinding(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>100</ID><IP>10.0.0.1</IP><TRACKING_METHOD>IP</TRACKING_METHOD>
		    <DNS>host1.example.com</DNS><NETBIOS>HOST1</NETBIOS><OS>Linux</OS>
		    <DETECTION_LIST>
		      <DETECTION>
		        <QID>105015</QID><TYPE>Confirmed</TYPE><SEVERITY>3</SEVERITY>
		        <PORT>443</PORT><PROTOCOL>tcp</PROTOCOL><SSL>1</SSL>
		        <RESULTS>TLSv1.0 enabled</RESULTS><STATUS>Active</STATUS>
		        <FIRST_FOUND_DATETIME>2026-01-01T00:00:00Z</FIRST_FOUND_DATETIME>
		        <LAST_FOUND_DATETIME>2026-06-01T00:00:00Z</LAST_FOUND_DATETIME>
		        <TIMES_FOUND>4</TIMES_FOUND>
		        <LAST_TEST_DATETIME>2026-06-01T00:00:00Z</LAST_TEST_DATETIME>
		        <IS_IGNORED>0</IS_IGNORED><IS_DISABLED>0</IS_DISABLED>
		      </DETECTION>
		    </DETECTION_LIST>
		  </HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	findings, conflicts, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err != nil {
		t.Fatalf("ListVMFindings: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %+v", conflicts)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	f := findings[0]
	if f.HostID != "100" || f.QID != "105015" || f.Port != 443 || f.Protocol != "tcp" ||
		f.Severity != 3 || f.Status != "Active" || !f.SSL || f.TimesFound != 4 {
		t.Errorf("finding = %+v", f)
	}
	if !f.HasPort {
		t.Error("HasPort should be true for a port-based detection")
	}
	if f.DNS != "host1.example.com" || f.OS != "Linux" || f.NetBIOS != "HOST1" {
		t.Errorf("host fields = %+v", f)
	}
}

func TestListVMFindingsOneHostMultipleFindings(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>100</ID><IP>10.0.0.1</IP>
		    <DETECTION_LIST>
		      <DETECTION><QID>1001</QID><SEVERITY>2</SEVERITY><STATUS>New</STATUS></DETECTION>
		      <DETECTION><QID>1002</QID><SEVERITY>4</SEVERITY><STATUS>Active</STATUS></DETECTION>
		    </DETECTION_LIST>
		  </HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	findings, _, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err != nil {
		t.Fatalf("ListVMFindings: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2 (no aggregation by host)", len(findings))
	}
}

func TestListVMFindingsMultipleHosts(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>100</ID><IP>10.0.0.1</IP>
		    <DETECTION_LIST><DETECTION><QID>1001</QID></DETECTION></DETECTION_LIST></HOST>
		  <HOST><ID>200</ID><IP>10.0.0.2</IP>
		    <DETECTION_LIST><DETECTION><QID>1001</QID></DETECTION></DETECTION_LIST></HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	findings, _, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err != nil {
		t.Fatalf("ListVMFindings: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2 (same QID on two hosts, not aggregated)", len(findings))
	}
	if findings[0].HostID == findings[1].HostID {
		t.Errorf("both findings report the same host: %+v", findings)
	}
}

// Same QID on the same host but different ports must remain two distinct
// findings — this is the fundamental data model the task was built against.
func TestListVMFindingsMultiplePortsSameQID(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>12345</ID><IP>10.0.0.1</IP>
		    <DETECTION_LIST>
		      <DETECTION><QID>105015</QID><PORT>443</PORT><PROTOCOL>tcp</PROTOCOL></DETECTION>
		      <DETECTION><QID>105015</QID><PORT>8443</PORT><PROTOCOL>tcp</PROTOCOL></DETECTION>
		    </DETECTION_LIST>
		  </HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	findings, conflicts, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err != nil {
		t.Fatalf("ListVMFindings: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %+v", conflicts)
	}
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2 distinct port-based detections", len(findings))
	}
	keys := map[string]bool{findings[0].FindingKey(): true, findings[1].FindingKey(): true}
	if !keys["12345:105015:TCP:443"] || !keys["12345:105015:TCP:8443"] {
		t.Errorf("keys = %v, want 12345:105015:TCP:443 and 12345:105015:TCP:8443", keys)
	}
}

func TestListVMFindingsMissingOptionalElements(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>100</ID><IP>10.0.0.1</IP>
		    <DETECTION_LIST>
		      <DETECTION><QID>1001</QID></DETECTION>
		    </DETECTION_LIST>
		  </HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	findings, _, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err != nil {
		t.Fatalf("ListVMFindings: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	f := findings[0]
	if f.HasPort || f.Port != 0 || f.Severity != 0 || f.SSL || f.IsIgnored {
		t.Errorf("finding with missing optional elements should zero-value them: %+v", f)
	}
	if f.FindingKey() != "100:1001" {
		t.Errorf("FindingKey = %q, want host:qid only when port/protocol are absent", f.FindingKey())
	}
}

func TestListVMFindingsEmptyDetectionList(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>100</ID><IP>10.0.0.1</IP></HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	findings, _, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err != nil {
		t.Fatalf("ListVMFindings: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("got %d findings, want 0 for a host with no detections", len(findings))
	}
}

func TestListVMFindingsEmptyHostList(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	findings, conflicts, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err != nil {
		t.Fatalf("ListVMFindings: %v", err)
	}
	if len(findings) != 0 || len(conflicts) != 0 {
		t.Fatalf("empty results must not be an error: findings=%v conflicts=%v", findings, conflicts)
	}
}

func TestListVMFindingsFollowsTruncation(t *testing.T) {
	var calls int
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = r.ParseForm()
		if r.Form.Get("id_min") == "" {
			fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
			  <HOST><ID>100</ID><IP>10.0.0.1</IP>
			    <DETECTION_LIST><DETECTION><QID>1001</QID></DETECTION></DETECTION_LIST></HOST>
			</HOST_LIST><WARNING><CODE>1980</CODE><TEXT>truncated</TEXT>
			  <URL>https://x/api/2.0/fo/asset/host/vm/detection/?action=list&amp;id_min=101</URL>
			</WARNING></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
			return
		}
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>200</ID><IP>10.0.0.2</IP>
		    <DETECTION_LIST><DETECTION><QID>1002</QID></DETECTION></DETECTION_LIST></HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	findings, conflicts, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err != nil {
		t.Fatalf("ListVMFindings: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	if len(findings) != 2 {
		t.Fatalf("got %d findings across both pages, want 2", len(findings))
	}
	if len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %+v", conflicts)
	}
}

// An exact duplicate finding returned on both sides of a truncation boundary
// (Qualys re-including the last record of a page as the first of the next,
// a documented possibility for cursor-based continuation) must collapse to
// one, not appear twice.
func TestListVMFindingsCollapsesExactDuplicateAcrossPages(t *testing.T) {
	var calls int
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = r.ParseForm()
		if r.Form.Get("id_min") == "" {
			fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
			  <HOST><ID>100</ID><IP>10.0.0.1</IP>
			    <DETECTION_LIST><DETECTION><QID>1001</QID><SEVERITY>3</SEVERITY></DETECTION></DETECTION_LIST></HOST>
			</HOST_LIST><WARNING><CODE>1980</CODE><TEXT>truncated</TEXT>
			  <URL>https://x/api/2.0/fo/asset/host/vm/detection/?action=list&amp;id_min=100</URL>
			</WARNING></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
			return
		}
		// Same host/QID/severity/everything repeated verbatim on page 2.
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>100</ID><IP>10.0.0.1</IP>
		    <DETECTION_LIST><DETECTION><QID>1001</QID><SEVERITY>3</SEVERITY></DETECTION></DETECTION_LIST></HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	findings, conflicts, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err != nil {
		t.Fatalf("ListVMFindings: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want the exact duplicate collapsed to 1", len(findings))
	}
	if len(conflicts) != 0 {
		t.Fatalf("an exact duplicate must not be reported as a conflict: %+v", conflicts)
	}
}

// Two records sharing a FindingKey but disagreeing on other fields (e.g. a
// detection updated between pages) must be flagged, not silently merged.
func TestListVMFindingsReportsConflictOnDisagreement(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("id_min") == "" {
			fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
			  <HOST><ID>100</ID><IP>10.0.0.1</IP>
			    <DETECTION_LIST><DETECTION><QID>1001</QID><STATUS>New</STATUS></DETECTION></DETECTION_LIST></HOST>
			</HOST_LIST><WARNING><CODE>1980</CODE><TEXT>truncated</TEXT>
			  <URL>https://x/api/2.0/fo/asset/host/vm/detection/?action=list&amp;id_min=100</URL>
			</WARNING></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
			return
		}
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>100</ID><IP>10.0.0.1</IP>
		    <DETECTION_LIST><DETECTION><QID>1001</QID><STATUS>Fixed</STATUS></DETECTION></DETECTION_LIST></HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	findings, conflicts, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err != nil {
		t.Fatalf("ListVMFindings: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1 (first-seen record kept)", len(findings))
	}
	if findings[0].Status != "New" {
		t.Errorf("kept status = %q, want the first-seen record's status", findings[0].Status)
	}
	if len(conflicts) != 1 {
		t.Fatalf("got %d conflicts, want 1", len(conflicts))
	}
	if conflicts[0].Key != "100:1001" || conflicts[0].First.Status != "New" || conflicts[0].Other.Status != "Fixed" {
		t.Errorf("conflict = %+v", conflicts[0])
	}
}

func TestListVMFindingsSendsHostIPStatusFilters(t *testing.T) {
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	_, _, err := c.ListVMFindings(context.Background(), VMFindingFilter{
		HostIDs: []string{"100", "200"},
		IPs:     []string{"10.0.0.0/24"},
		Status:  []string{"New", "Active"},
	})
	if err != nil {
		t.Fatalf("ListVMFindings: %v", err)
	}
	if form.Get("ids") != "100,200" {
		t.Errorf("ids = %q", form.Get("ids"))
	}
	if form.Get("status") != "New,Active" {
		t.Errorf("status = %q", form.Get("status"))
	}
	if form.Get("ips") == "" {
		t.Errorf("ips should be sent when IPs is set")
	}
}

func TestVMFindingKeyIsDeterministic(t *testing.T) {
	f1 := &VMFinding{HostID: "123456", QID: "105015", Port: 443, Protocol: "tcp", HasPort: true}
	f2 := &VMFinding{HostID: "123456", QID: "105015", Port: 443, Protocol: "tcp", HasPort: true}
	if f1.FindingKey() != f2.FindingKey() {
		t.Errorf("identical findings produced different keys: %q vs %q", f1.FindingKey(), f2.FindingKey())
	}
	if f1.FindingKey() != "123456:105015:TCP:443" {
		t.Errorf("FindingKey = %q", f1.FindingKey())
	}
}

func TestVMFindingKeyWithoutPortOmitsProtocolAndPort(t *testing.T) {
	f := &VMFinding{HostID: "123456", QID: "105015"}
	if f.FindingKey() != "123456:105015" {
		t.Errorf("FindingKey = %q, want host:qid only", f.FindingKey())
	}
}

func TestVMFindingKeyProtocolIsCanonicalUppercase(t *testing.T) {
	f := &VMFinding{HostID: "1", QID: "2", Port: 80, Protocol: "tcp", HasPort: true}
	if f.FindingKey() != "1:2:TCP:80" {
		t.Errorf("FindingKey = %q, want canonical uppercase protocol", f.FindingKey())
	}
}

// --- Error responses ---
// The shared do()/listAll() machinery (already covered by client_test.go and
// pagination_test.go) is reused unmodified by ListVMFindings; these tests
// confirm the error paths actually propagate through this specific call
// rather than being swallowed by findings()'s own decoding.

func TestListVMFindingsAuthenticationFailure(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>Basic authentication failed</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	_, _, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err == nil {
		t.Fatal("expected an error for a 401 response")
	}
}

func TestListVMFindingsPermissionsFailure(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><CODE>2011</CODE><TEXT>User role does not have permission for this API</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	_, _, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err == nil {
		t.Fatal("expected an error for a 403 permissions failure")
	}
}

func TestListVMFindingsMalformedXML(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST><HOST><ID>1</ID>`)
	}))
	defer srv.Close()

	_, _, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err == nil {
		t.Fatal("expected an error for truncated/malformed XML")
	}
}

func TestListVMFindingsQualysAPIError(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><CODE>999</CODE><TEXT>Internal error</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	_, _, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err == nil {
		t.Fatal("expected an error for a 200 response embedding a Qualys error code")
	}
}

func TestListVMFindingsThrottlingIsRetriedThenSucceeds(t *testing.T) {
	var calls int
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("X-RateLimit-ToWait-Sec", "0")
			w.WriteHeader(http.StatusConflict)
			fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>This API cannot be run again for another 1 seconds.</TEXT></RESPONSE></SIMPLE_RETURN>`)
			return
		}
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>1</ID><IP>10.0.0.1</IP>
		    <DETECTION_LIST><DETECTION><QID>1001</QID></DETECTION></DETECTION_LIST></HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	findings, _, err := c.ListVMFindings(context.Background(), VMFindingFilter{})
	if err != nil {
		t.Fatalf("ListVMFindings: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (throttled once, then retried)", calls)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
}

// ListVMFindings reuses listAll's own maxPages bound unmodified (a server
// that always claims truncation must not loop forever): that bound and its
// error are already exercised directly against listAll in
// TestListAllRefusesPartialResultsOnRunaway (pagination_test.go), so it is
// not re-proven with ~1000 real HTTP round trips at every call site — doing
// so here would only make this suite slow without adding coverage.
