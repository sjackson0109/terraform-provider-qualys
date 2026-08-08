package vmdr

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Scan schedules are the declarative alternative to launching scans. A scan is a
// job — it has no stable identity to converge on — whereas a schedule is
// configuration, which is why this provider models schedules and not scans.

// ScanScheduleActive encodes the four states the API reports.
//
// This is not a boolean. Values 2 and 3 apply to continuous schedules and were
// added after the 10.15 DTD, so a client that reads ACTIVE as a bool will
// mis-read a paused continuous schedule as active.
type ScanScheduleActive int

const (
	ScheduleDeactivated ScanScheduleActive = 0
	ScheduleActive      ScanScheduleActive = 1
	// ScheduleActiveNotPaused and SchedulePaused apply to continuous schedules.
	ScheduleActiveNotPaused ScanScheduleActive = 2
	SchedulePaused          ScanScheduleActive = 3
)

// Enabled reports whether the schedule will run. Paused continuous schedules
// count as disabled, which loses the distinction between 0 and 3 — callers that
// need it should read Active directly.
func (a ScanScheduleActive) Enabled() bool {
	return a == ScheduleActive || a == ScheduleActiveNotPaused
}

// ScanSchedule is a recurring VM scan.
type ScanSchedule struct {
	ID     string
	Title  string
	Active ScanScheduleActive

	OptionProfileID    string
	OptionProfileTitle string

	// Targets
	IPs                []string
	AssetGroupIDs      []string
	TargetFromTags     bool
	TagSetInclude      []string
	TagSetExclude      []string
	TagIncludeSelector string
	ExcludeIPs         []string
	NetworkID          string

	// Scanner selection
	ScannerNames     []string
	DefaultScanner   bool
	ScannersInAG     bool
	ScannersInTagset bool

	// Recurrence
	Occurrence      string
	FrequencyDays   int
	FrequencyWeeks  int
	Weekdays        []string
	FrequencyMonths int
	DayOfMonth      int
	DayOfWeek       int
	WeekOfMonth     string
	Recurrence      int

	// Time
	StartDate    string
	StartHour    int
	StartMinute  int
	TimeZoneCode string
	ObserveDST   bool

	// Duration
	EndAfterHours   int
	EndAfterMinutes int

	// Notifications
	BeforeNotify        bool
	BeforeNotifyUnit    string
	BeforeNotifyTime    int
	BeforeNotifyMessage string
	AfterNotify         bool
	AfterNotifyMessage  string
	RecipientGroupIDs   []string
}

// ScanScheduleInput is the desired state of a schedule.
type ScanScheduleInput struct {
	Title  string
	Active bool

	OptionProfileID    string
	OptionProfileTitle string

	IPs                []string
	AssetGroupIDs      []string
	TargetFromTags     bool
	TagSetInclude      []string
	TagSetExclude      []string
	TagIncludeSelector string
	ExcludeIPs         []string
	NetworkID          string

	ScannerNames     []string
	DefaultScanner   bool
	ScannersInAG     bool
	ScannersInTagset bool

	Occurrence      string
	FrequencyDays   int
	FrequencyWeeks  int
	Weekdays        []string
	FrequencyMonths int
	DayOfMonth      int
	DayOfWeek       int
	WeekOfMonth     string
	Recurrence      int

	StartDate    string
	StartHour    int
	StartMinute  int
	TimeZoneCode string
	ObserveDST   bool

	EndAfterHours   int
	EndAfterMinutes int

	BeforeNotify        bool
	BeforeNotifyUnit    string
	BeforeNotifyTime    int
	BeforeNotifyMessage string
	AfterNotify         bool
	AfterNotifyMessage  string
	RecipientGroupIDs   []string
}

type scheduleScanListOutput struct {
	Response struct {
		Schedules []struct {
			ID     string `xml:"ID"`
			Active string `xml:"ACTIVE"`
			Title  string `xml:"TITLE"`

			OptionProfile struct {
				Title string `xml:"TITLE"`
				ID    string `xml:"ID"`
			} `xml:"OPTION_PROFILE"`

			Processing struct {
				Occurrence string `xml:"OCCURRENCE"`
			} `xml:"PROCESSING_PRIORITY"`

			Schedule struct {
				StartDateUTC string `xml:"START_DATE_UTC"`
				StartHour    string `xml:"START_HOUR"`
				StartMinute  string `xml:"START_MINUTE"`
				TimeZone     struct {
					Code string `xml:"TIME_ZONE_CODE"`
				} `xml:"TIME_ZONE"`
				DST string `xml:"DST_SELECTED"`

				Daily struct {
					FrequencyDays string `xml:"frequency_days,attr"`
				} `xml:"DAILY"`
				Weekly struct {
					FrequencyWeeks string `xml:"frequency_weeks,attr"`
					Weekdays       string `xml:"weekdays,attr"`
				} `xml:"WEEKLY"`
				Monthly struct {
					FrequencyMonths string `xml:"frequency_months,attr"`
					DayOfMonth      string `xml:"day_of_month,attr"`
					DayOfWeek       string `xml:"day_of_week,attr"`
					WeekOfMonth     string `xml:"week_of_month,attr"`
				} `xml:"MONTHLY"`
			} `xml:"SCHEDULE"`

			Target struct {
				IPs           string `xml:"IP_SET>IP"`
				AssetGroupIDs string `xml:"ASSET_GROUP_TITLE_LIST>ASSET_GROUP_TITLE"`
			} `xml:"TARGET"`
		} `xml:"SCHEDULE_SCAN_LIST>SCAN"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *scheduleScanListOutput) warning() *warning { return o.Response.Warning }

func (o *scheduleScanListOutput) schedules() []*ScanSchedule {
	out := make([]*ScanSchedule, 0, len(o.Response.Schedules))
	for i := range o.Response.Schedules {
		s := &o.Response.Schedules[i]
		active, _ := strconv.Atoi(strings.TrimSpace(s.Active))
		sched := &ScanSchedule{
			ID:                 strings.TrimSpace(s.ID),
			Title:              s.Title,
			Active:             ScanScheduleActive(active),
			OptionProfileID:    strings.TrimSpace(s.OptionProfile.ID),
			OptionProfileTitle: s.OptionProfile.Title,
			StartDate:          s.Schedule.StartDateUTC,
			TimeZoneCode:       s.Schedule.TimeZone.Code,
			ObserveDST:         isTrue(s.Schedule.DST),
		}
		sched.StartHour, _ = strconv.Atoi(strings.TrimSpace(s.Schedule.StartHour))
		sched.StartMinute, _ = strconv.Atoi(strings.TrimSpace(s.Schedule.StartMinute))
		sched.FrequencyDays, _ = strconv.Atoi(strings.TrimSpace(s.Schedule.Daily.FrequencyDays))
		sched.FrequencyWeeks, _ = strconv.Atoi(strings.TrimSpace(s.Schedule.Weekly.FrequencyWeeks))
		sched.FrequencyMonths, _ = strconv.Atoi(strings.TrimSpace(s.Schedule.Monthly.FrequencyMonths))
		sched.DayOfMonth, _ = strconv.Atoi(strings.TrimSpace(s.Schedule.Monthly.DayOfMonth))

		switch {
		case sched.FrequencyDays > 0:
			sched.Occurrence = "daily"
		case sched.FrequencyWeeks > 0:
			sched.Occurrence = "weekly"
			sched.Weekdays = splitCSV(s.Schedule.Weekly.Weekdays)
		case sched.FrequencyMonths > 0:
			sched.Occurrence = "monthly"
			sched.WeekOfMonth = s.Schedule.Monthly.WeekOfMonth
		}

		out = append(out, sched)
	}
	return out
}

// scheduleParams encodes the fields shared by create and update.
func scheduleParams(in ScanScheduleInput) (url.Values, error) {
	p := url.Values{}
	p.Set("scan_title", in.Title)
	p.Set("active", boolParam(in.Active))

	if in.OptionProfileID == "" && in.OptionProfileTitle == "" {
		return nil, fmt.Errorf("qualys vmdr: an option profile is required for a scan schedule")
	}
	setIfNotEmpty(p, "option_id", in.OptionProfileID)
	setIfNotEmpty(p, "option_title", in.OptionProfileTitle)

	// Targets
	if in.TargetFromTags {
		p.Set("target_from", "tags")
		setListIfNotEmpty(p, "tag_set_include", in.TagSetInclude)
		setListIfNotEmpty(p, "tag_set_exclude", in.TagSetExclude)
		p.Set("tag_set_by", "id")
		setIfNotEmpty(p, "tag_include_selector", in.TagIncludeSelector)
	} else {
		p.Set("target_from", "assets")
		if len(in.IPs) > 0 {
			ips, err := FormatIPSet(in.IPs)
			if err != nil {
				return nil, err
			}
			p.Set("ip", ips)
		}
		setListIfNotEmpty(p, "asset_group_ids", in.AssetGroupIDs)
	}
	if len(in.ExcludeIPs) > 0 {
		ex, err := FormatIPSet(in.ExcludeIPs)
		if err != nil {
			return nil, err
		}
		p.Set("exclude_ip_per_scan", ex)
	}
	setIfNotEmpty(p, "ip_network_id", in.NetworkID)

	// Scanner selection
	setListIfNotEmpty(p, "iscanner_name", in.ScannerNames)
	if in.DefaultScanner {
		p.Set("default_scanner", "1")
	}
	if in.ScannersInAG {
		p.Set("scanners_in_ag", "1")
	}
	if in.ScannersInTagset {
		p.Set("scanners_in_tagset", "1")
	}

	// Recurrence. The API requires each occurrence's fields as a group.
	switch strings.ToLower(in.Occurrence) {
	case "daily":
		if in.FrequencyDays <= 0 {
			return nil, fmt.Errorf("qualys vmdr: a daily schedule requires frequency_days")
		}
		p.Set("occurrence", "daily")
		p.Set("frequency_days", strconv.Itoa(in.FrequencyDays))
	case "weekly":
		if in.FrequencyWeeks <= 0 || len(in.Weekdays) == 0 {
			return nil, fmt.Errorf("qualys vmdr: a weekly schedule requires frequency_weeks and weekdays")
		}
		p.Set("occurrence", "weekly")
		p.Set("frequency_weeks", strconv.Itoa(in.FrequencyWeeks))
		// weekdays takes day names, unlike day_of_week which is numeric.
		p.Set("weekdays", strings.Join(trimAll(in.Weekdays), ","))
	case "monthly":
		if in.FrequencyMonths <= 0 {
			return nil, fmt.Errorf("qualys vmdr: a monthly schedule requires frequency_months")
		}
		p.Set("occurrence", "monthly")
		p.Set("frequency_months", strconv.Itoa(in.FrequencyMonths))
		switch {
		case in.DayOfMonth > 0:
			p.Set("day_of_month", strconv.Itoa(in.DayOfMonth))
		case in.WeekOfMonth != "":
			p.Set("day_of_week", strconv.Itoa(in.DayOfWeek))
			p.Set("week_of_month", in.WeekOfMonth)
		default:
			return nil, fmt.Errorf("qualys vmdr: a monthly schedule requires either day_of_month " +
				"or day_of_week with week_of_month")
		}
	case "":
		return nil, fmt.Errorf("qualys vmdr: an occurrence (daily, weekly or monthly) is required")
	default:
		return nil, fmt.Errorf("qualys vmdr: unknown occurrence %q", in.Occurrence)
	}
	if in.Recurrence > 0 {
		p.Set("recurrence", strconv.Itoa(in.Recurrence))
	}

	// Time
	setIfNotEmpty(p, "start_date", in.StartDate)
	p.Set("start_hour", strconv.Itoa(in.StartHour))
	p.Set("start_minute", strconv.Itoa(in.StartMinute))
	setIfNotEmpty(p, "time_zone_code", in.TimeZoneCode)
	if in.ObserveDST {
		p.Set("observe_dst", "yes")
	} else {
		p.Set("observe_dst", "no")
	}

	// Duration cap. These bound how long a scan may run; they are not an end date,
	// and the API has no end-date parameter.
	if in.EndAfterHours > 0 || in.EndAfterMinutes > 0 {
		p.Set("end_after", strconv.Itoa(in.EndAfterHours))
		p.Set("end_after_mins", strconv.Itoa(in.EndAfterMinutes))
	}

	// Notifications. recipient_group_ids is only valid alongside a notify flag.
	if in.BeforeNotify {
		p.Set("before_notify", "1")
		setIfNotEmpty(p, "before_notify_unit", in.BeforeNotifyUnit)
		if in.BeforeNotifyTime > 0 {
			p.Set("before_notify_time", strconv.Itoa(in.BeforeNotifyTime))
		}
		setIfNotEmpty(p, "before_notify_message", in.BeforeNotifyMessage)
	}
	if in.AfterNotify {
		p.Set("after_notify", "1")
		setIfNotEmpty(p, "after_notify_message", in.AfterNotifyMessage)
	}
	if in.BeforeNotify || in.AfterNotify {
		setListIfNotEmpty(p, "recipient_group_ids", in.RecipientGroupIDs)
	}

	return p, nil
}

// CreateScanSchedule creates a schedule and returns its ID.
func (c *Client) CreateScanSchedule(ctx context.Context, in ScanScheduleInput) (string, error) {
	params, err := scheduleParams(in)
	if err != nil {
		return "", err
	}

	var out simpleReturn
	if err := c.do(ctx, request{
		capability: capScheduleScan,
		path:       "schedule/scan/",
		action:     "create",
		params:     params,
	}, &out); err != nil {
		return "", err
	}

	id, ok := out.itemValue("ID")
	if !ok || strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("qualys vmdr: scan schedule was created but the API returned no ID "+
			"(response: %q); import it to bring it under management", out.Response.Text)
	}
	return strings.TrimSpace(id), nil
}

// UpdateScanSchedule applies desired state.
//
// Changing the start time requires set_start_time=1 together with all five time
// fields; the API does not accept them individually. This client always sends
// the complete group, so any time change takes effect.
func (c *Client) UpdateScanSchedule(ctx context.Context, id string, in ScanScheduleInput) error {
	params, err := scheduleParams(in)
	if err != nil {
		return err
	}
	params.Set("id", id)
	params.Set("set_start_time", "1")

	return c.do(ctx, request{
		capability: capScheduleScan,
		path:       "schedule/scan/",
		action:     "update",
		params:     params,
	}, nil)
}

// DeleteScanSchedule removes a schedule. Scans already running are unaffected.
func (c *Client) DeleteScanSchedule(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("id", id)
	return c.do(ctx, request{
		capability:  capScheduleScan,
		path:        "schedule/scan/",
		action:      "delete",
		params:      params,
		destructive: true,
	}, nil)
}

// GetScanSchedule returns one schedule, or ErrNotFound.
func (c *Client) GetScanSchedule(ctx context.Context, id string) (*ScanSchedule, error) {
	schedules, err := c.ListScanSchedules(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, s := range schedules {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, fmt.Errorf("qualys vmdr: scan schedule %s: %w", id, ErrNotFound)
}

// ListScanSchedules returns schedules, optionally filtered to one ID.
func (c *Client) ListScanSchedules(ctx context.Context, id string) ([]*ScanSchedule, error) {
	params := url.Values{}
	setIfNotEmpty(params, "id", id)
	params.Set("show_notifications", "1")

	var all []*ScanSchedule
	err := c.listAll(ctx, request{
		capability: capScheduleScan,
		path:       "schedule/scan/",
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(scheduleScanListOutput) },
		func(p paginated) error {
			all = append(all, p.(*scheduleScanListOutput).schedules()...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}
