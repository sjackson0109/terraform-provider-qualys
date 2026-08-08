package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func validSchedule() map[string]interface{} {
	return map[string]interface{}{
		"title":               "nightly",
		"option_profile_id":   "51451401",
		"asset_group_ids":     []interface{}{"4021975"},
		"use_default_scanner": true,
		"occurrence":          "daily",
		"frequency_days":      1,
		"start_hour":          2,
		"start_minute":        30,
		"time_zone_code":      "US-NY",
	}
}

func TestScanScheduleAcceptsAValidDailySchedule(t *testing.T) {
	if err := diffFor(t, resourceScanSchedule(), validSchedule()); err != nil {
		t.Fatalf("valid schedule rejected: %v", err)
	}
}

func TestScanScheduleRequiresFrequencyForItsOccurrence(t *testing.T) {
	cfg := validSchedule()
	delete(cfg, "frequency_days")
	if err := diffFor(t, resourceScanSchedule(), cfg); err == nil {
		t.Error("expected an error: a daily schedule needs frequency_days")
	}

	cfg = validSchedule()
	cfg["occurrence"] = "weekly"
	delete(cfg, "frequency_days")
	cfg["frequency_weeks"] = 1
	if err := diffFor(t, resourceScanSchedule(), cfg); err == nil {
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

	err := diffFor(t, resourceScanSchedule(), cfg)
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
	if err := diffFor(t, resourceScanSchedule(), cfg); err == nil {
		t.Error("expected an error: a schedule with no target scans nothing")
	}

	cfg = validSchedule()
	cfg["use_default_scanner"] = false
	if err := diffFor(t, resourceScanSchedule(), cfg); err == nil {
		t.Error("expected an error: a schedule with no scanner cannot run")
	}
}

func TestScanScheduleWeekdayNamesAreValidated(t *testing.T) {
	elem := resourceScanSchedule().Schema["weekdays"].Elem.(*schema.Schema)
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
