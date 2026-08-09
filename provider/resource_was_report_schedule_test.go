package provider

import "testing"

func TestWASReportScheduleRequiresCoreFields(t *testing.T) {
	r := resourceWASReportSchedule()
	for _, attr := range []string{
		"name", "report_template_id", "web_app_id", "output_format", "recipients",
		"start_date", "time_zone_code", "occurrence_type",
	} {
		if s := r.Schema[attr]; s == nil || !s.Required {
			t.Errorf("%s should be required", attr)
		}
	}
}

func TestWASReportScheduleActiveDefaultsTrue(t *testing.T) {
	s := resourceWASReportSchedule().Schema["active"]
	if s == nil || s.Default != true {
		t.Error("active should default to true")
	}
}
