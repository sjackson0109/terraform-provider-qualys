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

func TestCreateWASScanScheduleSendsSchedulingSubObject(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WasScanSchedule":{"id":9,"name":"weekly-storefront"}}]}}`)
	}))
	defer srv.Close()

	sched, err := c.CreateWASScanSchedule(context.Background(), WASScanScheduleInput{
		Name: "weekly-storefront", Type: WASScanTypeVulnerability, Active: true, WebAppID: "1296335669",
		OptionProfileID: "712265669", WebAppAuthRecordID: "175535669", ScannerType: WASScannerExternal,
		StartDate: "2026-08-16T02:00:00Z", TimeZoneCode: "Europe/London",
		OccurrenceType: WASOccurrenceWeekly,
		Recurrence: &WASScheduleRecurrence{
			EveryNWeeks: 1, OnDays: []string{WASWeekDaySunday},
		},
	})
	if err != nil {
		t.Fatalf("CreateWASScanSchedule: %v", err)
	}
	if sched.ID != "9" {
		t.Errorf("sched = %+v", sched)
	}
	if gotPath != "/qps/rest/3.0/create/was/wasscanschedule" {
		t.Errorf("path = %q", gotPath)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	schedule, _ := data["WasScanSchedule"].(map[string]interface{})
	if schedule == nil {
		t.Fatalf("no WasScanSchedule in payload: %v", data)
	}

	target, _ := schedule["target"].(map[string]interface{})
	webApp, _ := target["webApp"].(map[string]interface{})
	if webApp["id"] != float64(1296335669) {
		t.Errorf("target.webApp.id = %v", webApp["id"])
	}
	authRecord, _ := target["webAppAuthRecord"].(map[string]interface{})
	if authRecord["id"] != float64(175535669) {
		t.Errorf("target.webAppAuthRecord.id = %v", authRecord["id"])
	}
	scanner, _ := target["scannerAppliance"].(map[string]interface{})
	if scanner["type"] != "EXTERNAL" {
		t.Errorf("target.scannerAppliance.type = %v", scanner["type"])
	}

	scheduling, _ := schedule["scheduling"].(map[string]interface{})
	if scheduling == nil {
		t.Fatalf("no scheduling sub-object in payload: %v", schedule)
	}
	timeZone, _ := scheduling["timeZone"].(map[string]interface{})
	if timeZone["code"] != "Europe/London" {
		t.Errorf("scheduling.timeZone.code = %v", timeZone["code"])
	}
	if scheduling["occurrenceType"] != "WEEKLY" {
		t.Errorf("scheduling.occurrenceType = %v", scheduling["occurrenceType"])
	}
	occurrence, _ := scheduling["occurrence"].(map[string]interface{})
	weekly, _ := occurrence["weeklyOccurrence"].(map[string]interface{})
	if weekly["everyNWeeks"] != float64(1) {
		t.Errorf("occurrence.weeklyOccurrence.everyNWeeks = %v", weekly["everyNWeeks"])
	}
	onDays, _ := weekly["onDays"].(map[string]interface{})
	days, _ := onDays["WeekDay"].([]interface{})
	if len(days) != 1 || days[0] != "SUNDAY" {
		t.Errorf("onDays.WeekDay = %v", onDays)
	}

	// Flat top-level fields from the earlier (incorrect) schema must not
	// reappear at the top level now that scheduling is a sub-object.
	if _, present := schedule["startDate"]; present {
		t.Errorf("startDate must live under scheduling, not top-level: %v", schedule)
	}
	if _, present := schedule["cancelOption"]; present {
		t.Errorf("cancelOption must live under target, not top-level: %v", schedule)
	}
}

func TestCreateWASScanScheduleSendsDailyRecurrence(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WasScanSchedule":{"id":10,"name":"daily"}}]}}`)
	}))
	defer srv.Close()

	_, err := c.CreateWASScanSchedule(context.Background(), WASScanScheduleInput{
		Name: "daily", WebAppID: "1", StartDate: "2026-08-16T02:00:00Z", TimeZoneCode: "UTC",
		OccurrenceType: WASOccurrenceDaily,
		Recurrence:     &WASScheduleRecurrence{EveryNDays: 3},
	})
	if err != nil {
		t.Fatalf("CreateWASScanSchedule: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	schedule, _ := data["WasScanSchedule"].(map[string]interface{})
	scheduling, _ := schedule["scheduling"].(map[string]interface{})
	occurrence, _ := scheduling["occurrence"].(map[string]interface{})
	daily, _ := occurrence["dailyOccurrence"].(map[string]interface{})
	if daily["everyNDays"] != float64(3) {
		t.Errorf("occurrence.dailyOccurrence.everyNDays = %v", daily["everyNDays"])
	}
}

func TestCreateWASScanScheduleSendsMonthlyRecurrence(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WasScanSchedule":{"id":12,"name":"monthly"}}]}}`)
	}))
	defer srv.Close()

	_, err := c.CreateWASScanSchedule(context.Background(), WASScanScheduleInput{
		Name: "monthly", WebAppID: "1", StartDate: "2026-08-16T02:00:00Z", TimeZoneCode: "UTC",
		OccurrenceType: WASOccurrenceMonthly,
		Recurrence:     &WASScheduleRecurrence{DayOfMonth: 1, EveryNMonths: 1},
	})
	if err != nil {
		t.Fatalf("CreateWASScanSchedule: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	schedule, _ := data["WasScanSchedule"].(map[string]interface{})
	scheduling, _ := schedule["scheduling"].(map[string]interface{})
	occurrence, _ := scheduling["occurrence"].(map[string]interface{})
	monthly, _ := occurrence["monthlyOccurrence"].(map[string]interface{})
	if monthly["dayOfMonth"] != float64(1) || monthly["everyNMonths"] != float64(1) {
		t.Errorf("occurrence.monthlyOccurrence = %v", monthly)
	}
}

func TestCreateWASScanScheduleSendsNotification(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WasScanSchedule":{"id":11,"name":"notified"}}]}}`)
	}))
	defer srv.Close()

	_, err := c.CreateWASScanSchedule(context.Background(), WASScanScheduleInput{
		Name: "notified", WebAppID: "1", StartDate: "2026-08-16T02:00:00Z", TimeZoneCode: "UTC",
		OccurrenceType: WASOccurrenceOnce,
		Notification: &WASScheduleNotification{
			Active: true, DelayAmount: 1, DelayScale: "DAY",
			Recipients: []string{"security@example.com"}, Message: "A scan is starting soon.",
		},
	})
	if err != nil {
		t.Fatalf("CreateWASScanSchedule: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	schedule, _ := data["WasScanSchedule"].(map[string]interface{})
	notification, _ := schedule["notification"].(map[string]interface{})
	if notification == nil {
		t.Fatalf("no notification in payload: %v", schedule)
	}
	recipients, _ := notification["recipients"].(map[string]interface{})
	set, _ := recipients["set"].(map[string]interface{})
	emails, _ := set["EmailAddress"].([]interface{})
	if len(emails) != 1 || emails[0] != "security@example.com" {
		t.Errorf("recipients.set.EmailAddress = %v", emails)
	}
}

func TestCreateWASScanScheduleRequiresATarget(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent without a web app target")
	}))
	defer srv.Close()

	_, err := c.CreateWASScanSchedule(context.Background(), WASScanScheduleInput{
		Name: "x", StartDate: "2026-09-01", TimeZoneCode: "UTC", OccurrenceType: WASOccurrenceOnce,
	})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestWASScanScheduleNotFoundIsRecognised(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
	}))
	defer srv.Close()

	_, err := c.GetWASScanSchedule(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateWASScanScheduleIsNotRetriedOnTransportError(t *testing.T) {
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

	in := WASScanScheduleInput{
		Name: "x", WebAppID: "1", StartDate: "2026-09-01", TimeZoneCode: "UTC",
		OccurrenceType: WASOccurrenceOnce,
	}
	if _, err := c.CreateWASScanSchedule(context.Background(), in); err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("create was sent %d times; a lost response must not cause a re-send", calls)
	}
}

func TestActivateAndDeactivateWASScanScheduleUseDedicatedEndpoints(t *testing.T) {
	var gotPaths []string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	if err := c.ActivateWASScanSchedule(context.Background(), "1688"); err != nil {
		t.Fatalf("ActivateWASScanSchedule: %v", err)
	}
	if err := c.DeactivateWASScanSchedule(context.Background(), "1688"); err != nil {
		t.Fatalf("DeactivateWASScanSchedule: %v", err)
	}
	want := []string{
		"/qps/rest/3.0/activate/was/wasscanschedule/1688",
		"/qps/rest/3.0/deactivate/was/wasscanschedule/1688",
	}
	if len(gotPaths) != 2 || gotPaths[0] != want[0] || gotPaths[1] != want[1] {
		t.Errorf("paths = %v, want %v", gotPaths, want)
	}
}

func TestSearchWASScanSchedulesDecodesListShape(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"false",
		  "data":[{"WasScanSchedule":{"id":1,"name":"a"}},{"WasScanSchedule":{"id":2,"name":"b"}}]}}`)
	}))
	defer srv.Close()

	schedules, err := c.SearchWASScanSchedules(context.Background(), nil)
	if err != nil {
		t.Fatalf("SearchWASScanSchedules: %v", err)
	}
	if len(schedules) != 2 {
		t.Fatalf("got %d schedules, want 2", len(schedules))
	}
}
