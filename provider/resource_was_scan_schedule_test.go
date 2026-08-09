package provider

import "testing"

func TestWASScanScheduleRequiresCoreFields(t *testing.T) {
	r := resourceWASScanSchedule()
	for _, attr := range []string{"name", "web_app_id", "type", "start_date", "time_zone", "occurrence_type"} {
		if s := r.Schema[attr]; s == nil || !s.Required {
			t.Errorf("%s should be required", attr)
		}
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
