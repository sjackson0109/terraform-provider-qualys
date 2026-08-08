package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func baseSchedule() ScanScheduleInput {
	return ScanScheduleInput{
		Title:           "nightly",
		Active:          true,
		OptionProfileID: "51451401",
		AssetGroupIDs:   []string{"4021975"},
		Occurrence:      "daily",
		FrequencyDays:   1,
		StartDate:       "01/15/2026",
		StartHour:       2,
		StartMinute:     30,
		TimeZoneCode:    "US-NY",
	}
}

func captureSchedule(t *testing.T, in ScanScheduleInput, update bool) url.Values {
	t.Helper()
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT>
		  <ITEM_LIST><ITEM><KEY>ID</KEY><VALUE>160642</VALUE></ITEM></ITEM_LIST>
		</RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	var err error
	if update {
		err = c.UpdateScanSchedule(context.Background(), "160642", in)
	} else {
		_, err = c.CreateScanSchedule(context.Background(), in)
	}
	if err != nil {
		t.Fatalf("schedule call: %v", err)
	}
	return form
}

// The API only accepts a start-time change when set_start_time=1 accompanies all
// five time fields. Omitting any of them silently leaves the time unchanged.
func TestUpdateSendsCompleteStartTimeGroup(t *testing.T) {
	form := captureSchedule(t, baseSchedule(), true)

	if form.Get("set_start_time") != "1" {
		t.Error("set_start_time was not sent; the start time would not change")
	}
	for _, k := range []string{"start_date", "start_hour", "start_minute", "time_zone_code", "observe_dst"} {
		if _, ok := form[k]; !ok {
			t.Errorf("%s was not sent; the API requires the whole time group together", k)
		}
	}
}

func TestWeeklyScheduleUsesDayNames(t *testing.T) {
	in := baseSchedule()
	in.Occurrence = "weekly"
	in.FrequencyDays = 0
	in.FrequencyWeeks = 1
	in.Weekdays = []string{"monday", "thursday"}

	form := captureSchedule(t, in, false)

	// weekdays takes day names; day_of_week (monthly) is numeric. Mixing them up
	// is easy and the API would reject or misinterpret the value.
	if got, want := form.Get("weekdays"), "monday,thursday"; got != want {
		t.Errorf("weekdays = %q, want %q", got, want)
	}
	if form.Get("frequency_weeks") != "1" {
		t.Errorf("frequency_weeks = %q", form.Get("frequency_weeks"))
	}
}

func TestMonthlyScheduleRequiresADaySelector(t *testing.T) {
	in := baseSchedule()
	in.Occurrence = "monthly"
	in.FrequencyDays = 0
	in.FrequencyMonths = 1

	if _, err := scheduleParams(in); err == nil {
		t.Fatal("expected an error: a monthly schedule needs day_of_month or week_of_month")
	}

	in.DayOfMonth = 15
	p, err := scheduleParams(in)
	if err != nil {
		t.Fatalf("scheduleParams: %v", err)
	}
	if p.Get("day_of_month") != "15" {
		t.Errorf("day_of_month = %q", p.Get("day_of_month"))
	}
}

func TestRecurrenceRequiresItsFrequency(t *testing.T) {
	in := baseSchedule()
	in.FrequencyDays = 0
	if _, err := scheduleParams(in); err == nil {
		t.Error("expected an error: a daily schedule needs frequency_days")
	}

	in = baseSchedule()
	in.Occurrence = "weekly"
	in.FrequencyDays = 0
	in.FrequencyWeeks = 1
	if _, err := scheduleParams(in); err == nil {
		t.Error("expected an error: a weekly schedule needs weekdays")
	}
}

func TestScheduleRequiresOptionProfile(t *testing.T) {
	in := baseSchedule()
	in.OptionProfileID = ""
	if _, err := scheduleParams(in); err == nil {
		t.Error("expected an error: a scan schedule cannot run without an option profile")
	}
}

// recipient_group_ids is only valid alongside a notify flag.
func TestRecipientsOnlySentWithNotification(t *testing.T) {
	in := baseSchedule()
	in.RecipientGroupIDs = []string{"4228"}

	p, err := scheduleParams(in)
	if err != nil {
		t.Fatalf("scheduleParams: %v", err)
	}
	if p.Get("recipient_group_ids") != "" {
		t.Error("recipients were sent without a notification flag, which the API rejects")
	}

	in.BeforeNotify = true
	in.BeforeNotifyTime = 20
	in.BeforeNotifyUnit = "hours"
	p, err = scheduleParams(in)
	if err != nil {
		t.Fatalf("scheduleParams: %v", err)
	}
	if p.Get("recipient_group_ids") != "4228" {
		t.Errorf("recipient_group_ids = %q", p.Get("recipient_group_ids"))
	}
}

// ACTIVE is a four-valued enum; reading it as a boolean mis-reports a paused
// continuous schedule.
func TestActiveIsFourValued(t *testing.T) {
	cases := []struct {
		v       ScanScheduleActive
		enabled bool
	}{
		{ScheduleDeactivated, false},
		{ScheduleActive, true},
		{ScheduleActiveNotPaused, true},
		{SchedulePaused, false},
	}
	for _, c := range cases {
		if got := c.v.Enabled(); got != c.enabled {
			t.Errorf("Active(%d).Enabled() = %v, want %v", c.v, got, c.enabled)
		}
	}
}

func TestListParsesActiveAndRecurrence(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<SCHEDULE_SCAN_LIST_OUTPUT><RESPONSE><SCHEDULE_SCAN_LIST>
		  <SCAN>
		    <ID>160642</ID><ACTIVE>3</ACTIVE><TITLE>nightly</TITLE>
		    <OPTION_PROFILE><ID>51451401</ID><TITLE>web</TITLE></OPTION_PROFILE>
		    <SCHEDULE>
		      <START_HOUR>2</START_HOUR><START_MINUTE>30</START_MINUTE>
		      <TIME_ZONE><TIME_ZONE_CODE>US-NY</TIME_ZONE_CODE></TIME_ZONE>
		      <WEEKLY frequency_weeks="2" weekdays="monday,friday"/>
		    </SCHEDULE>
		  </SCAN>
		</SCHEDULE_SCAN_LIST></RESPONSE></SCHEDULE_SCAN_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	s, err := c.GetScanSchedule(context.Background(), "160642")
	if err != nil {
		t.Fatalf("GetScanSchedule: %v", err)
	}
	if s.Active != SchedulePaused {
		t.Errorf("Active = %d, want %d (paused)", s.Active, SchedulePaused)
	}
	if s.Active.Enabled() {
		t.Error("a paused schedule reported itself as enabled")
	}
	if s.Occurrence != "weekly" || s.FrequencyWeeks != 2 {
		t.Errorf("recurrence = %s/%d", s.Occurrence, s.FrequencyWeeks)
	}
	if len(s.Weekdays) != 2 {
		t.Errorf("weekdays = %v", s.Weekdays)
	}
}

// The five time fields are a required-together group; a schedule without a
// start date cannot be encoded at all, matching the schema's Required flag.
func TestScheduleParamsRequireStartDate(t *testing.T) {
	in := baseSchedule()
	in.StartDate = ""
	if _, err := scheduleParams(in); err == nil {
		t.Fatal("expected an error: the API only accepts the time as a complete group")
	}
}

// The list output reports targets; a plain-string mapping would keep only the
// last IP element and drop ranges entirely.
func TestListParsesTargets(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<SCHEDULE_SCAN_LIST_OUTPUT><RESPONSE><SCHEDULE_SCAN_LIST>
		  <SCAN>
		    <ID>160642</ID><ACTIVE>1</ACTIVE><TITLE>nightly</TITLE>
		    <TARGET>
		      <IP_SET><IP>10.0.0.5</IP><IP>10.0.0.9</IP><IP_RANGE>10.1.0.0-10.1.0.255</IP_RANGE></IP_SET>
		      <ASSET_GROUP_TITLE_LIST><ASSET_GROUP_TITLE>prod-web</ASSET_GROUP_TITLE></ASSET_GROUP_TITLE_LIST>
		    </TARGET>
		    <SCHEDULE>
		      <START_HOUR>2</START_HOUR><START_MINUTE>0</START_MINUTE>
		      <MONTHLY frequency_months="1" day_of_week="3" week_of_month="second"/>
		    </SCHEDULE>
		  </SCAN>
		</SCHEDULE_SCAN_LIST></RESPONSE></SCHEDULE_SCAN_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	s, err := c.GetScanSchedule(context.Background(), "160642")
	if err != nil {
		t.Fatalf("GetScanSchedule: %v", err)
	}
	if len(s.IPs) != 3 {
		t.Errorf("IPs = %v; all IP and IP_RANGE elements must survive decoding", s.IPs)
	}
	if len(s.AssetGroupTitles) != 1 || s.AssetGroupTitles[0] != "prod-web" {
		t.Errorf("AssetGroupTitles = %v", s.AssetGroupTitles)
	}
	if s.DayOfWeek != 3 || s.WeekOfMonth != "second" {
		t.Errorf("day-in-week recurrence not decoded: %+v", s)
	}
}
