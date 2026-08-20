package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
)

func validSchedule() map[string]interface{} {
	return map[string]interface{}{
		"title":               "nightly",
		"option_profile_id":   "51451401",
		"asset_group_ids":     []interface{}{"4021975"},
		"use_default_scanner": true,
		"occurrence":          "daily",
		"frequency_days":      1,
		"start_date":          "01/15/2026",
		"start_hour":          2,
		"start_minute":        30,
		"time_zone_code":      "US-NY",
	}
}

func TestScanScheduleAcceptsAValidDailySchedule(t *testing.T) {
	if err := diffFor(t, resourceVMScanSchedule(), validSchedule()); err != nil {
		t.Fatalf("valid schedule rejected: %v", err)
	}
}

func TestScanScheduleRequiresFrequencyForItsOccurrence(t *testing.T) {
	cfg := validSchedule()
	delete(cfg, "frequency_days")
	if err := diffFor(t, resourceVMScanSchedule(), cfg); err == nil {
		t.Error("expected an error: a daily schedule needs frequency_days")
	}

	cfg = validSchedule()
	cfg["occurrence"] = "weekly"
	delete(cfg, "frequency_days")
	cfg["frequency_weeks"] = 1
	if err := diffFor(t, resourceVMScanSchedule(), cfg); err == nil {
		t.Error("expected an error: a weekly schedule needs weekdays")
	}
}

// day_of_month and week_of_month are alternatives. Accepting both would leave
// which one wins up to the API.
func TestScanScheduleRejectsConflictingMonthlySelectors(t *testing.T) {
	cfg := validSchedule()
	cfg["occurrence"] = "monthly"
	delete(cfg, "frequency_days")
	cfg["frequency_months"] = 1
	cfg["day_of_month"] = 15
	cfg["week_of_month"] = "first"

	err := diffFor(t, resourceVMScanSchedule(), cfg)
	if err == nil {
		t.Fatal("expected an error when both monthly selectors are set")
	}
	if !strings.Contains(err.Error(), "alternatives") {
		t.Errorf("unexpected error: %v", err)
	}
}

// A schedule with no target scans nothing, and one with no scanner cannot run.
// Both are accepted by the API and silently useless, so they are caught here.
func TestScanScheduleRequiresATargetAndAScanner(t *testing.T) {
	cfg := validSchedule()
	delete(cfg, "asset_group_ids")
	if err := diffFor(t, resourceVMScanSchedule(), cfg); err == nil {
		t.Error("expected an error: a schedule with no target scans nothing")
	}

	cfg = validSchedule()
	cfg["use_default_scanner"] = false
	if err := diffFor(t, resourceVMScanSchedule(), cfg); err == nil {
		t.Error("expected an error: a schedule with no scanner cannot run")
	}
}

func TestScanScheduleWeekdayNamesAreValidated(t *testing.T) {
	elem := resourceVMScanSchedule().Schema["weekdays"].Elem.(*schema.Schema)
	v := elem.ValidateFunc
	if _, errs := v("monday", "weekdays"); len(errs) != 0 {
		t.Errorf("valid weekday rejected: %v", errs)
	}
	// Numeric days belong to day_of_week, not weekdays. Rejecting them here
	// catches an easy confusion between the two encodings.
	if _, errs := v("1", "weekdays"); len(errs) == 0 {
		t.Error("expected a numeric day to be rejected; weekdays takes names")
	}
}

// TestScanScheduleReadRefreshesRecurrenceAndTargets is a regression test:
// Read's values map used to omit weekdays, day_of_week, start_date and ips
// even though the list/get API genuinely returns all four (unlike
// asset_group_ids, which the API only ever reports by title — see the
// comment in resourceVMScanScheduleRead). A schedule imported or refreshed
// after drifting outside Terraform used to silently show the wrong (zero)
// value for these fields with no diff ever appearing.
func TestScanScheduleReadRefreshesRecurrenceAndTargets(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<SCHEDULE_SCAN_LIST_OUTPUT><RESPONSE><SCHEDULE_SCAN_LIST>
		  <SCAN>
		    <ID>160642</ID><ACTIVE>1</ACTIVE><TITLE>nightly</TITLE>
		    <OPTION_PROFILE><ID>51451401</ID><TITLE>web</TITLE></OPTION_PROFILE>
		    <SCHEDULE>
		      <START_DATE_UTC>08/20/2026</START_DATE_UTC>
		      <START_HOUR>2</START_HOUR><START_MINUTE>30</START_MINUTE>
		      <TIME_ZONE><TIME_ZONE_CODE>US-NY</TIME_ZONE_CODE></TIME_ZONE>
		      <WEEKLY frequency_weeks="2" weekdays="monday,friday"/>
		    </SCHEDULE>
		    <TARGET>
		      <IP_SET><IP>10.0.0.5</IP><IP>10.0.0.9</IP></IP_SET>
		    </TARGET>
		  </SCAN>
		</SCHEDULE_SCAN_LIST></RESPONSE></SCHEDULE_SCAN_LIST_OUTPUT>`)
	}))
	defer srv.Close()
	c, err := vmdr.NewClient(vmdr.Config{
		BaseURL: srv.URL, Username: "u", Password: "p", HTTPClient: srv.Client(), MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("vmdr.NewClient: %v", err)
	}

	r := resourceVMScanSchedule()
	d := r.Data(&terraform.InstanceState{ID: "160642"})
	diags := r.ReadContext(context.Background(), d, &clients{vmdr: c})
	if diags.HasError() {
		t.Fatalf("Read: %v", diags)
	}

	weekdays := stringSetFromInterface(d.Get("weekdays"))
	if len(weekdays) != 2 {
		t.Errorf("weekdays = %v, want 2 entries", weekdays)
	}
	if got := d.Get("start_date").(string); got != "08/20/2026" {
		t.Errorf("start_date = %q", got)
	}
	ips := stringSetFromInterface(d.Get("ips"))
	if len(ips) != 2 {
		t.Errorf("ips = %v, want 2 entries", ips)
	}
}
