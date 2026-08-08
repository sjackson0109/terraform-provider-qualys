package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestCreateVirtualScannerReturnsActivationCode(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<APPLIANCE_CREATE_OUTPUT><RESPONSE>
		  <TEXT>New virtual scanner created successfully</TEXT>
		  <ID>11223</ID><NAME>dmz-scanner</NAME>
		  <ACTIVATION_CODE>12345678901234</ACTIVATION_CODE>
		  <REMAINING_QVSA_LICENSES>4</REMAINING_QVSA_LICENSES>
		</RESPONSE></APPLIANCE_CREATE_OUTPUT>`)
	}))
	defer srv.Close()

	got, err := c.CreateVirtualScanner(context.Background(), VirtualScannerInput{Name: "dmz-scanner"})
	if err != nil {
		t.Fatalf("CreateVirtualScanner: %v", err)
	}
	if got.ID != "11223" {
		t.Errorf("ID = %q", got.ID)
	}
	// The activation code is the whole point: without it the appliance VM cannot
	// be personalised.
	if got.ActivationCode != "12345678901234" {
		t.Errorf("ActivationCode = %q", got.ActivationCode)
	}
	if got.RemainingLicenses != "4" {
		t.Errorf("RemainingLicenses = %q", got.RemainingLicenses)
	}
}

// Creating a scanner consumes a licence, so it must not be repeated on a
// blocked response.
func TestCreateVirtualScannerIsNotRetried(t *testing.T) {
	var calls int
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-RateLimit-ToWait-Sec", "1")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>This API cannot be run again for another 1 seconds.</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	_, _ = c.CreateVirtualScanner(context.Background(), VirtualScannerInput{Name: "x"})
	if calls != 1 {
		t.Errorf("create was issued %d times; it consumes a licence and must not repeat", calls)
	}
}

func TestPollingIntervalIsRangeChecked(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("an out-of-range polling interval should not reach the API")
	}))
	defer srv.Close()

	if _, err := c.CreateVirtualScanner(context.Background(), VirtualScannerInput{
		Name: "x", PollingInterval: 30,
	}); err == nil {
		t.Error("expected an error below the documented minimum of 60")
	}
	if err := c.UpdateAppliance(context.Background(), "1", ApplianceUpdate{PollingInterval: 3600}); err == nil {
		t.Error("expected an error above the documented maximum of 360")
	}
}

// set_vlans replaces the whole list, so it must only be sent when the caller
// actually manages VLANs — otherwise an unrelated update would wipe them.
func TestVLANsOnlySentWhenManaged(t *testing.T) {
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	if err := c.UpdateAppliance(context.Background(), "1", ApplianceUpdate{Name: "renamed"}); err != nil {
		t.Fatalf("UpdateAppliance: %v", err)
	}
	if _, sent := form["set_vlans"]; sent {
		t.Error("set_vlans was sent for an update that does not manage VLANs; it would clear them")
	}

	err := c.UpdateAppliance(context.Background(), "1", ApplianceUpdate{
		SetVLANs: true,
		VLANs:    []VLAN{{ID: "10", IP: "10.0.0.5", Netmask: "255.255.255.0", Name: "dmz"}},
	})
	if err != nil {
		t.Fatalf("UpdateAppliance: %v", err)
	}
	if got, want := form.Get("set_vlans"), "10|10.0.0.5|255.255.255.0|dmz"; got != want {
		t.Errorf("set_vlans = %q, want %q", got, want)
	}

	// An empty managed list clears VLANs, which the API expresses as "".
	if err := c.UpdateAppliance(context.Background(), "1", ApplianceUpdate{SetVLANs: true}); err != nil {
		t.Fatalf("UpdateAppliance: %v", err)
	}
	if v, sent := form["set_vlans"]; !sent || v[0] != "" {
		t.Errorf("clearing VLANs should send an empty set_vlans, got %v", v)
	}
}

func TestApplianceReadinessSignals(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<APPLIANCE_LIST_OUTPUT><RESPONSE><APPLIANCE_LIST><APPLIANCE>
		  <ID>11223</ID><NAME>dmz-scanner</NAME><STATUS>Online</STATUS>
		  <HEARTBEATS_MISSED>0</HEARTBEATS_MISSED>
		  <VULNSIGS_VERSION>2.5.730-2</VULNSIGS_VERSION><VULNSIGS_LATEST>2.5.730-2</VULNSIGS_LATEST>
		  <ML_VERSION>12.13.38-1</ML_VERSION><ML_LATEST>12.13.39-1</ML_LATEST>
		</APPLIANCE></APPLIANCE_LIST></RESPONSE></APPLIANCE_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	a, err := c.GetAppliance(context.Background(), "11223")
	if err != nil {
		t.Fatalf("GetAppliance: %v", err)
	}
	if !a.Online() {
		t.Error("appliance reported Online but Online() said otherwise")
	}
	// Signatures match but the module version lags, so the appliance is not fully
	// up to date. Comparing only one pair would miss this.
	if a.UpToDate() {
		t.Error("UpToDate() ignored a lagging ML version")
	}
}

func TestAssignApplianceToNetworkIsASeparateAction(t *testing.T) {
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	if err := c.AssignApplianceToNetwork(context.Background(), "11223", "7343"); err != nil {
		t.Fatalf("AssignApplianceToNetwork: %v", err)
	}
	if form.Get("action") != "assign_network_id" {
		t.Errorf("action = %q; network assignment is its own action, not an update parameter",
			form.Get("action"))
	}
	if form.Get("appliance_id") != "11223" || form.Get("network_id") != "7343" {
		t.Errorf("form = %v", form)
	}
}
