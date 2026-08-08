package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

// The update action distinguishes selectors from setters. Sending a bare `owner`
// filters the selection instead of setting anything, so writes must use new_*.
func TestUpdateIPsUsesSetterSpelling(t *testing.T) {
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	err := c.UpdateIPs(context.Background(), []string{"10.0.0.1"}, "", HostAssetUpdate{
		Owner:          "platform",
		TrackingMethod: "DNS",
		Comment:        "managed by terraform",
		UD1:            "one",
	})
	if err != nil {
		t.Fatalf("UpdateIPs: %v", err)
	}

	for _, setter := range []string{"new_owner", "new_tracking_method", "new_comment", "new_ud1"} {
		if form.Get(setter) == "" {
			t.Errorf("%s was not sent; without it the value is not written", setter)
		}
	}
	// Bare names would silently filter rather than set.
	for _, selector := range []string{"owner", "comment", "ud1"} {
		if form.Get(selector) != "" {
			t.Errorf("%s was sent as a setter, but it is a selector; the update would "+
				"filter instead of writing", selector)
		}
	}
	// tracking_method is a legitimate selector, but must not carry the new value.
	if form.Get("tracking_method") == "DNS" {
		t.Error("the new tracking method was sent as a selector rather than new_tracking_method")
	}
}

func TestAddIPsRequiresAModule(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	err := c.AddIPs(context.Background(), AddIPsInput{IPs: []string{"10.0.0.1"}})
	if err == nil {
		t.Fatal("expected an error: an IP registered for no module cannot be scanned")
	}
}

func TestAddIPsExpandsCIDR(t *testing.T) {
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	if err := c.AddIPs(context.Background(), AddIPsInput{
		IPs: []string{"10.0.0.0/30"}, EnableVM: true,
	}); err != nil {
		t.Fatalf("AddIPs: %v", err)
	}
	if got, want := form.Get("ips"), "10.0.0.0-10.0.0.3"; got != want {
		t.Errorf("ips = %q, want %q", got, want)
	}
	if form.Get("enable_vm") != "1" {
		t.Errorf("enable_vm = %q", form.Get("enable_vm"))
	}
}

// An unselected purge would target the whole subscription.
func TestPurgeRequiresASelector(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("purge was dispatched with no selector")
	}))
	defer srv.Close()

	if err := c.PurgeHosts(context.Background(), PurgeInput{}); err == nil {
		t.Fatal("expected an error for an unselected purge")
	}
}

func TestPurgeIsNotRetried(t *testing.T) {
	var calls int
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-RateLimit-ToWait-Sec", "1")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>This API cannot be run again for another 1 seconds.</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	_ = c.PurgeHosts(context.Background(), PurgeInput{IPs: []string{"10.0.0.1"}})
	if calls != 1 {
		t.Errorf("purge was issued %d times; a destructive call must not be repeated", calls)
	}
}

func TestListHostsSupportsStaleFilter(t *testing.T) {
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<HOST_LIST_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST><ID>1</ID><IP>10.0.0.1</IP><TRACKING_METHOD>IP</TRACKING_METHOD>
		    <LAST_VM_SCANNED_DATE>2025-01-02T00:00:00Z</LAST_VM_SCANNED_DATE></HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	hosts, err := c.ListHosts(context.Background(), HostFilter{NoVMScanSince: "2025-06-01"})
	if err != nil {
		t.Fatalf("ListHosts: %v", err)
	}
	if form.Get("no_vm_scan_since") != "2025-06-01" {
		t.Errorf("no_vm_scan_since = %q", form.Get("no_vm_scan_since"))
	}
	if len(hosts) != 1 || hosts[0].IP != "10.0.0.1" {
		t.Errorf("hosts = %+v", hosts)
	}
}

func TestNetworkCreateAndList(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch r.Form.Get("action") {
		case "create":
			fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>Network created</TEXT>
			  <ITEM_LIST><ITEM><KEY>ID</KEY><VALUE>7343</VALUE></ITEM></ITEM_LIST>
			</RESPONSE></SIMPLE_RETURN>`)
		default:
			fmt.Fprint(w, `<NETWORK_LIST_OUTPUT><RESPONSE><NETWORK_LIST>
			  <NETWORK><ID>7343</ID><NAME>dmz</NAME>
			    <SCANNER_APPLIANCE_LIST><SCANNER_APPLIANCE><ID>11</ID></SCANNER_APPLIANCE></SCANNER_APPLIANCE_LIST>
			  </NETWORK>
			</NETWORK_LIST></RESPONSE></NETWORK_LIST_OUTPUT>`)
		}
	}))
	defer srv.Close()

	id, err := c.CreateNetwork(context.Background(), "dmz")
	if err != nil {
		t.Fatalf("CreateNetwork: %v", err)
	}
	if id != "7343" {
		t.Errorf("id = %q", id)
	}

	n, err := c.GetNetwork(context.Background(), "7343")
	if err != nil {
		t.Fatalf("GetNetwork: %v", err)
	}
	if n.Name != "dmz" || len(n.ApplianceIDs) != 1 || n.ApplianceIDs[0] != "11" {
		t.Errorf("network = %+v", n)
	}
}

// data_scope and friends narrow what is purged, not which hosts. Treating them
// as selectors would let a filter-only input through as a subscription-wide
// purge.
func TestPurgeGuardCountsOnlyHostSelectors(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("purge was dispatched without a host selector")
	}))
	defer srv.Close()

	for _, in := range []PurgeInput{
		{DataScope: "vm"},
		{ComplianceEnabled: true},
		{OSPattern: ".*"},
		{NoVMScanSince: "2020-01-01"},
		{NetworkIDs: []string{"7343"}},
	} {
		if err := c.PurgeHosts(context.Background(), in); err == nil {
			t.Errorf("expected an error for filter-only purge input %+v", in)
		}
	}
}

func TestPurgeAcceptsAHostSelector(t *testing.T) {
	var dispatched bool
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dispatched = true
		fmt.Fprint(w, `<BATCH_RETURN><RESPONSE><BATCH_LIST><BATCH>
		  <TEXT>Hosts Queued for Purging</TEXT></BATCH></BATCH_LIST></RESPONSE></BATCH_RETURN>`)
	}))
	defer srv.Close()

	if err := c.PurgeHosts(context.Background(), PurgeInput{
		AssetGroupIDs: []string{"4021975"}, DataScope: "vm",
	}); err != nil {
		t.Fatalf("PurgeHosts: %v", err)
	}
	if !dispatched {
		t.Error("a purge with a host selector should be dispatched")
	}
}
