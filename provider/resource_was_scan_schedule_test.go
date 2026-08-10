package provider

import "testing"

func TestWASScanScheduleRequiresCoreFields(t *testing.T) {
	r := resourceWASScanSchedule()
	for _, attr := range []string{"name", "web_app_id", "type", "start_date", "time_zone_code", "occurrence_type"} {
		if s := r.Schema[attr]; s == nil || !s.Required {
			t.Errorf("%s should be required", attr)
		}
	}
}

func TestWASScanScheduleAuthRecordIDAndUseDefaultConflict(t *testing.T) {
	s := resourceWASScanSchedule().Schema
	if len(s["web_app_auth_record_id"].ConflictsWith) == 0 {
		t.Error("web_app_auth_record_id must conflict with web_app_auth_record_use_default")
	}
	if len(s["web_app_auth_record_use_default"].ConflictsWith) == 0 {
		t.Error("web_app_auth_record_use_default must conflict with web_app_auth_record_id")
	}
}

func TestWASScanScheduleActiveDefaultsTrue(t *testing.T) {
	s := resourceWASScanSchedule().Schema["active"]
	if s == nil || s.Default != true {
		t.Error("active should default to true")
	}
}

func TestWASScanScheduleOptionProfileIsOptional(t *testing.T) {
	// A web application with a default option profile does not need one set
	// explicitly, so this cannot be Required at the schema level.
	s := resourceWASScanSchedule().Schema["option_profile_id"]
	if s == nil || s.Required {
		t.Error("option_profile_id must be optional")
	}
}
