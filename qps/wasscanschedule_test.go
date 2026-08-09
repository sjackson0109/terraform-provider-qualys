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

func TestCreateWASScanScheduleSendsConfirmedElements(t *testing.T) {
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
		Name: "weekly-storefront", Type: WASScanTypeVulnerability, WebAppID: "555",
		OptionProfileID: "42", StartDate: "2026-09-01T09:00:00Z", TimeZone: "UTC",
		OccurrenceType: WASOccurrenceWeekly,
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
	if webApp["id"] != float64(555) {
		t.Errorf("target.webApp.id = %v, want 555", webApp["id"])
	}
	profile, _ := schedule["profile"].(map[string]interface{})
	if profile["id"] != float64(42) {
		t.Errorf("profile.id = %v, want 42", profile["id"])
	}
	if schedule["occurrenceType"] != "WEEKLY" {
		t.Errorf("occurrenceType = %v", schedule["occurrenceType"])
	}
}

func TestCreateWASScanScheduleRequiresATarget(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent without a web app target")
	}))
	defer srv.Close()

	_, err := c.CreateWASScanSchedule(context.Background(), WASScanScheduleInput{
		Name: "x", StartDate: "2026-09-01", TimeZone: "UTC", OccurrenceType: WASOccurrenceOnce,
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
		Name: "x", WebAppID: "1", StartDate: "2026-09-01", TimeZone: "UTC",
		OccurrenceType: WASOccurrenceOnce,
	}
	if _, err := c.CreateWASScanSchedule(context.Background(), in); err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("create was sent %d times; a lost response must not cause a re-send", calls)
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
