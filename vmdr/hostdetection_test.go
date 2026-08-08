package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestListHostDetectionsSupportsStaleFilter(t *testing.T) {
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>1</ID><IP>10.0.0.1</IP><TRACKING_METHOD>IP</TRACKING_METHOD>
		    <OS>Linux</OS><LAST_SCAN_DATETIME>2025-01-02T00:00:00Z</LAST_SCAN_DATETIME></HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	hosts, err := c.ListHostDetections(context.Background(), HostDetectionFilter{VMScanDateBefore: "2025-06-01"})
	if err != nil {
		t.Fatalf("ListHostDetections: %v", err)
	}
	if form.Get("vm_scan_date_before") != "2025-06-01" {
		t.Errorf("vm_scan_date_before = %q", form.Get("vm_scan_date_before"))
	}
	if len(hosts) != 1 || hosts[0].LastScanDatetime != "2025-01-02T00:00:00Z" {
		t.Errorf("hosts = %+v", hosts)
	}
}

func TestListHostDetectionsFollowsTruncation(t *testing.T) {
	var calls int
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = r.ParseForm()
		if r.Form.Get("id_min") == "" {
			fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
			  <HOST><ID>1</ID><IP>10.0.0.1</IP></HOST>
			</HOST_LIST><WARNING><CODE>1980</CODE><TEXT>truncated</TEXT>
			  <URL>https://x/api/2.0/fo/asset/host/vm/detection/?action=list&amp;id_min=2</URL>
			</WARNING></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
			return
		}
		fmt.Fprint(w, `<HOST_LIST_VM_DETECTION_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>2</ID><IP>10.0.0.2</IP></HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_VM_DETECTION_OUTPUT>`)
	}))
	defer srv.Close()

	hosts, err := c.ListHostDetections(context.Background(), HostDetectionFilter{})
	if err != nil {
		t.Fatalf("ListHostDetections: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("got %d hosts, want 2 across both pages", len(hosts))
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}
