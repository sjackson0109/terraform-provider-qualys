package qps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestCreateWASReportScheduleSendsConfirmedShape(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"ReportSchedule":{"id":7,"name":"weekly-exec"}}]}}`)
	}))
	defer srv.Close()

	sched, err := c.CreateWASReportSchedule(context.Background(), WASReportScheduleInput{
		Name: "weekly-exec", Active: true, ReportTemplateID: "55442", WebAppID: "1296335669",
		OutputFormat: WASReportFormatPDF, Recipients: []string{"security@example.com"},
		StartDate: "2026-08-16T06:00:00Z", TimeZoneCode: "Europe/London",
		OccurrenceType: WASOccurrenceWeekly,
		Recurrence:     &WASScheduleRecurrence{EveryNWeeks: 1, OnDays: []string{WASWeekDayMonday}},
	})
	if err != nil {
		t.Fatalf("CreateWASReportSchedule: %v", err)
	}
	if sched.ID != "7" {
		t.Errorf("sched = %+v", sched)
	}
	if gotPath != "/qps/rest/3.0/create/was/reportschedule" {
		t.Errorf("path = %q", gotPath)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	schedule, _ := data["ReportSchedule"].(map[string]interface{})
	if schedule == nil {
		t.Fatalf("no ReportSchedule in payload: %v", data)
	}

	reportTemplate, _ := schedule["reportTemplate"].(map[string]interface{})
	if reportTemplate["id"] != float64(55442) {
		t.Errorf("reportTemplate.id = %v", reportTemplate["id"])
	}
	target, _ := schedule["target"].(map[string]interface{})
	webApp, _ := target["webApp"].(map[string]interface{})
	if webApp["id"] != float64(1296335669) {
		t.Errorf("target.webApp.id = %v", webApp["id"])
	}
	output, _ := schedule["output"].(map[string]interface{})
	if output["format"] != "PDF" {
		t.Errorf("output.format = %v", output["format"])
	}
	notification, _ := schedule["notification"].(map[string]interface{})
	recipients, _ := notification["recipients"].(map[string]interface{})
	set, _ := recipients["set"].(map[string]interface{})
	emails, _ := set["EmailAddress"].([]interface{})
	if len(emails) != 1 || emails[0] != "security@example.com" {
		t.Errorf("notification.recipients.set.EmailAddress = %v", emails)
	}

	schedWire, _ := schedule["schedule"].(map[string]interface{})
	occurrence, _ := schedWire["occurrence"].(map[string]interface{})
	weekly, _ := occurrence["weeklyOccurrence"].(map[string]interface{})
	if weekly["everyNWeeks"] != float64(1) {
		t.Errorf("schedule.occurrence.weeklyOccurrence.everyNWeeks = %v", weekly["everyNWeeks"])
	}
}

func TestCreateWASReportScheduleSendsMonthlyRecurrence(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"ReportSchedule":{"id":8,"name":"monthly"}}]}}`)
	}))
	defer srv.Close()

	_, err := c.CreateWASReportSchedule(context.Background(), WASReportScheduleInput{
		Name: "monthly", ReportTemplateID: "1", WebAppID: "1",
		StartDate: "2026-08-16T06:00:00Z", TimeZoneCode: "Europe/London",
		OccurrenceType: WASOccurrenceMonthly,
		Recurrence:     &WASScheduleRecurrence{DayOfMonth: 1, EveryNMonths: 1},
	})
	if err != nil {
		t.Fatalf("CreateWASReportSchedule: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	schedule, _ := data["ReportSchedule"].(map[string]interface{})
	schedWire, _ := schedule["schedule"].(map[string]interface{})
	occurrence, _ := schedWire["occurrence"].(map[string]interface{})
	monthly, _ := occurrence["monthlyOccurrence"].(map[string]interface{})
	if monthly["dayOfMonth"] != float64(1) || monthly["everyNMonths"] != float64(1) {
		t.Errorf("occurrence.monthlyOccurrence = %v", monthly)
	}
}

func TestCreateWASReportScheduleRequiresATemplateAndTarget(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent without a template and target")
	}))
	defer srv.Close()

	if _, err := c.CreateWASReportSchedule(context.Background(), WASReportScheduleInput{
		Name: "x", StartDate: "2026-09-01", TimeZoneCode: "UTC", OccurrenceType: WASOccurrenceOnce,
	}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestWASReportScheduleNotFoundIsRecognised(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
	}))
	defer srv.Close()

	_, err := c.GetWASReportSchedule(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateWASReportScheduleIsNotRetriedOnTransportError(t *testing.T) {
	var calls int
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("test server does not support hijacking")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		conn.Close()
	}))
	defer srv.Close()

	in := WASReportScheduleInput{
		Name: "x", ReportTemplateID: "1", WebAppID: "1", StartDate: "2026-09-01", TimeZoneCode: "UTC",
		OccurrenceType: WASOccurrenceOnce,
	}
	if _, err := c.CreateWASReportSchedule(context.Background(), in); err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("create was sent %d times; a lost response must not cause a re-send", calls)
	}
}

func TestActivateAndDeactivateWASReportScheduleUseDedicatedEndpoints(t *testing.T) {
	var gotPaths []string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	if err := c.ActivateWASReportSchedule(context.Background(), "12345"); err != nil {
		t.Fatalf("ActivateWASReportSchedule: %v", err)
	}
	if err := c.DeactivateWASReportSchedule(context.Background(), "12345"); err != nil {
		t.Fatalf("DeactivateWASReportSchedule: %v", err)
	}
	want := []string{
		"/qps/rest/3.0/activate/was/reportschedule/12345",
		"/qps/rest/3.0/deactivate/was/reportschedule/12345",
	}
	if len(gotPaths) != 2 || gotPaths[0] != want[0] || gotPaths[1] != want[1] {
		t.Errorf("paths = %v, want %v", gotPaths, want)
	}
}

func TestSearchWASReportSchedulesDecodesListShape(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"false",
		  "data":[{"ReportSchedule":{"id":1,"name":"a"}},{"ReportSchedule":{"id":2,"name":"b"}}]}}`)
	}))
	defer srv.Close()

	schedules, err := c.SearchWASReportSchedules(context.Background(), nil)
	if err != nil {
		t.Fatalf("SearchWASReportSchedules: %v", err)
	}
	if len(schedules) != 2 {
		t.Fatalf("got %d schedules, want 2", len(schedules))
	}
}
