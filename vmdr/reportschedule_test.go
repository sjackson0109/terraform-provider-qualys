package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestListReportSchedulesSendsConfirmedFiltersAndParsesFields(t *testing.T) {
	var gotForm url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.Form
		fmt.Fprint(w, `<SCHEDULE_REPORT_LIST_OUTPUT><RESPONSE><SCHEDULE_REPORT_LIST>
			<SCHEDULE_REPORT><ID>501</ID><TITLE>Weekly Exec Summary</TITLE><ACTIVE>1</ACTIVE>
			<REPORT><TEMPLATE_ID>10</TEMPLATE_ID><TEMPLATE_TITLE>Executive</TEMPLATE_TITLE>
			<OUTPUT_FORMAT>pdf</OUTPUT_FORMAT></REPORT></SCHEDULE_REPORT>
			</SCHEDULE_REPORT_LIST></RESPONSE></SCHEDULE_REPORT_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	active := true
	schedules, err := c.ListReportSchedules(context.Background(), ReportScheduleFilter{ID: "501", Active: &active})
	if err != nil {
		t.Fatalf("ListReportSchedules: %v", err)
	}
	if gotForm.Get("id") != "501" || gotForm.Get("is_active") != "true" {
		t.Errorf("form = %v", gotForm)
	}
	if len(schedules) != 1 {
		t.Fatalf("got %d schedules, want 1", len(schedules))
	}
	s := schedules[0]
	if s.ID != "501" || s.Title != "Weekly Exec Summary" || !s.Active ||
		s.TemplateID != "10" || s.TemplateTitle != "Executive" || s.OutputFormat != "pdf" {
		t.Errorf("schedule = %+v", s)
	}
}

func TestLaunchReportScheduleNowSendsID(t *testing.T) {
	var gotAction string
	var gotForm url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotAction = r.Form.Get("action")
		gotForm = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>OK</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	if err := c.LaunchReportScheduleNow(context.Background(), "501"); err != nil {
		t.Fatalf("LaunchReportScheduleNow: %v", err)
	}
	if gotAction != "launch_now" || gotForm.Get("id") != "501" {
		t.Errorf("action = %q form = %v", gotAction, gotForm)
	}
}
