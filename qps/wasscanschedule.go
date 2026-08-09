package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// WAS scan schedule scan types.
const (
	WASScanTypeDiscovery     = "DISCOVERY"
	WASScanTypeVulnerability = "VULNERABILITY"
)

// WAS scan schedule occurrence types.
const (
	WASOccurrenceOnce    = "ONCE"
	WASOccurrenceDaily   = "DAILY"
	WASOccurrenceWeekly  = "WEEKLY"
	WASOccurrenceMonthly = "MONTHLY"
)

// WAS scan schedule scanner appliance types.
const (
	WASScannerExternal = "EXTERNAL"
	WASScannerInternal = "INTERNAL"
)

// WAS scan schedule cancel options.
const (
	WASCancelOptionDefault  = "DEFAULT"
	WASCancelOptionSpecific = "SPECIFIC"
)

// WAS scan schedule notification sender options.
const (
	WASSendMailFromQualysSupport = "QUALYS_SUPPORT"
	WASSendMailFromOwner         = "OWNER"
)

// Week days accepted by a weekly schedule's onDays list.
const (
	WASWeekDaySunday    = "SUNDAY"
	WASWeekDayMonday    = "MONDAY"
	WASWeekDayTuesday   = "TUESDAY"
	WASWeekDayWednesday = "WEDNESDAY"
	WASWeekDayThursday  = "THURSDAY"
	WASWeekDayFriday    = "FRIDAY"
	WASWeekDaySaturday  = "SATURDAY"
)

// WASScheduleRecurrence is the per-occurrence cadence detail for a DAILY,
// WEEKLY or MONTHLY schedule. Shared between qualys_was_scan_schedule and
// qualys_was_report_schedule, which use an identical occurrence structure.
//
// The MONTHLY shape (dayOfMonth, everyNMonths) was seen only in a
// user-supplied WAS *report* schedule example, not a scan schedule one —
// applied here by analogy, since the DAILY and WEEKLY shapes are identical
// across both schedule types in the evidence obtained so far. Only the
// day-of-month form is modelled; a "the Nth weekday of the month" pattern
// (as some scheduling APIs also support) has not been seen in any source.
type WASScheduleRecurrence struct {
	// EveryNDays applies to a DAILY schedule.
	EveryNDays int
	// EveryNWeeks, OnDays and OccurrenceCount apply to a WEEKLY schedule.
	// OnDays holds WASWeekDay* values. OccurrenceCount, if set, ends the
	// schedule after that many occurrences.
	EveryNWeeks     int
	OnDays          []string
	OccurrenceCount int
	// DayOfMonth and EveryNMonths apply to a MONTHLY schedule.
	DayOfMonth   int
	EveryNMonths int
}

// WASScheduleNotification is the pre-scan notification a schedule can send.
type WASScheduleNotification struct {
	Active bool
	// Reschedule enables a separate notification when a scan is
	// rescheduled rather than run as planned.
	Reschedule bool
	// DelayAmount/DelayScale set how long before the scan the notification
	// goes out, e.g. DelayAmount=1, DelayScale="DAY".
	DelayAmount int
	DelayScale  string
	Recipients  []string
	Message     string
}

// WASScanSchedule is a recurring WAS scan.
type WASScanSchedule struct {
	ID       string
	Name     string
	Type     string
	Active   bool
	WebAppID string

	WebAppAuthRecordID        string
	WebAppAuthRecordIsDefault bool
	ScannerType               string
	ScannerFriendlyName       string
	CancelOption              string

	OptionProfileID string
	ProxyID         string
	DNSOverrideID   string

	StartDate        string
	TimeZoneCode     string
	OccurrenceType   string
	CancelAfterHours int
	Recurrence       *WASScheduleRecurrence

	Notification              *WASScheduleNotification
	SendMail                  bool
	SendOneMail               bool
	SendMailFromAddressOption string

	Created string
}

// WASScanScheduleInput is the desired state of a WAS scan schedule.
//
// The element model is now Confirmed against a user-supplied "Create and
// Update Scan Schedule" walkthrough (transcribed, not a verbatim PDF quote,
// so treated as strong but not absolute evidence) covering ONCE/DAILY/WEEKLY
// occurrences, the full target sub-object (web app, auth record,
// scanner appliance, cancel option), scheduling (startDate, timeZone.code,
// cancelAfterNHours), notifications (active, reschedule, delay, email
// recipients, message) and completion mail (sendMail, sendOneMail,
// sendMailFromAddressOption). This replaced an earlier version built only
// from doc 11's official Quick Reference, which named these top-level
// elements but not their exact nesting (scheduling and notification turned
// out to be sub-objects, not flat fields; cancelOption turned out to live
// under target, not top-level) or the recurrence sub-elements at all.
//
// MONTHLY recurrence detail remains unconfirmed — see the WASScheduleRecurrence
// doc comment. Tag-based (multi-web-app) targeting is still not modelled.
type WASScanScheduleInput struct {
	Name     string
	Type     string
	Active   bool
	WebAppID string

	WebAppAuthRecordID        string
	WebAppAuthRecordIsDefault bool
	ScannerType               string
	ScannerFriendlyName       string
	CancelOption              string

	OptionProfileID string
	ProxyID         string
	DNSOverrideID   string

	StartDate        string
	TimeZoneCode     string
	OccurrenceType   string
	CancelAfterHours int
	Recurrence       *WASScheduleRecurrence

	Notification              *WASScheduleNotification
	SendMail                  bool
	SendOneMail               bool
	SendMailFromAddressOption string
}

type wasIDRefWire struct {
	ID json.Number `json:"id,omitempty"`
}

type wasAuthRecordRefWire struct {
	ID        json.Number `json:"id,omitempty"`
	IsDefault *bool       `json:"isDefault,omitempty"`
}

type wasScannerApplianceWire struct {
	Type         string `json:"type,omitempty"`
	FriendlyName string `json:"friendlyName,omitempty"`
}

type wasScheduleTargetWire struct {
	WebApp           *wasIDRefWire            `json:"webApp,omitempty"`
	WebAppAuthRecord *wasAuthRecordRefWire    `json:"webAppAuthRecord,omitempty"`
	ScannerAppliance *wasScannerApplianceWire `json:"scannerAppliance,omitempty"`
	CancelOption     string                   `json:"cancelOption,omitempty"`
}

type wasTimeZoneWire struct {
	Code string `json:"code,omitempty"`
}

type wasWeekDayListWire struct {
	WeekDay []string `json:"WeekDay,omitempty"`
}

type wasDailyOccurrenceWire struct {
	EveryNDays int `json:"everyNDays,omitempty"`
}

type wasWeeklyOccurrenceWire struct {
	EveryNWeeks     int                 `json:"everyNWeeks,omitempty"`
	OccurrenceCount int                 `json:"occurrenceCount,omitempty"`
	OnDays          *wasWeekDayListWire `json:"onDays,omitempty"`
}

type wasMonthlyOccurrenceWire struct {
	DayOfMonth   int `json:"dayOfMonth,omitempty"`
	EveryNMonths int `json:"everyNMonths,omitempty"`
}

type wasOccurrenceWire struct {
	DailyOccurrence   *wasDailyOccurrenceWire   `json:"dailyOccurrence,omitempty"`
	WeeklyOccurrence  *wasWeeklyOccurrenceWire  `json:"weeklyOccurrence,omitempty"`
	MonthlyOccurrence *wasMonthlyOccurrenceWire `json:"monthlyOccurrence,omitempty"`
}

type wasSchedulingWire struct {
	CancelAfterNHours int                `json:"cancelAfterNHours,omitempty"`
	StartDate         string             `json:"startDate,omitempty"`
	TimeZone          *wasTimeZoneWire   `json:"timeZone,omitempty"`
	OccurrenceType    string             `json:"occurrenceType,omitempty"`
	Occurrence        *wasOccurrenceWire `json:"occurrence,omitempty"`
}

type wasDelayWire struct {
	NB    int    `json:"nb,omitempty"`
	Scale string `json:"scale,omitempty"`
}

// wasEmailSetWire is the recipients list wrapper. A user-supplied example
// shows only the "set" (authoritative replace) idiom on write; "list" is
// this package's own addition for decoding a GET response, inferred by
// analogy with how tags round-trip rather than confirmed against a sample
// response.
type wasEmailSetWire struct {
	EmailAddress []string `json:"EmailAddress,omitempty"`
}

type wasEmailListWire struct {
	Set  *wasEmailSetWire `json:"set,omitempty"`
	List *wasEmailSetWire `json:"list,omitempty"`
}

type wasNotificationWire struct {
	Active     *bool             `json:"active,omitempty"`
	Reschedule *bool             `json:"reschedule,omitempty"`
	Delay      *wasDelayWire     `json:"delay,omitempty"`
	Recipients *wasEmailListWire `json:"recipients,omitempty"`
	Message    string            `json:"message,omitempty"`
}

type wasScanScheduleWire struct {
	ID                        json.Number            `json:"id,omitempty"`
	Name                      string                 `json:"name,omitempty"`
	Type                      string                 `json:"type,omitempty"`
	Active                    *bool                  `json:"active,omitempty"`
	Target                    *wasScheduleTargetWire `json:"target,omitempty"`
	Profile                   *wasIDRefWire          `json:"profile,omitempty"`
	Proxy                     *wasIDRefWire          `json:"proxy,omitempty"`
	DNSOverride               *wasIDRefWire          `json:"dnsOverride,omitempty"`
	Scheduling                *wasSchedulingWire     `json:"scheduling,omitempty"`
	Notification              *wasNotificationWire   `json:"notification,omitempty"`
	SendMail                  *bool                  `json:"sendMail,omitempty"`
	SendOneMail               *bool                  `json:"sendOneMail,omitempty"`
	SendMailFromAddressOption string                 `json:"sendMailFromAddressOption,omitempty"`
	Created                   string                 `json:"createdDate,omitempty"`
}

// wasScanScheduleData is the "data" envelope. The wrapper key WasScanSchedule
// is Confirmed: it matches this package's naming-convention guess and
// appears verbatim in a user-supplied create example.
type wasScanScheduleData struct {
	WasScanSchedule *wasScanScheduleWire `json:"WasScanSchedule,omitempty"`
}

func boolPtr(b bool) *bool { return &b }

func wasScanScheduleInputToWire(in WASScanScheduleInput) *wasScanScheduleWire {
	w := &wasScanScheduleWire{
		Name:                      in.Name,
		Type:                      in.Type,
		Active:                    boolPtr(in.Active),
		SendMail:                  boolPtr(in.SendMail),
		SendOneMail:               boolPtr(in.SendOneMail),
		SendMailFromAddressOption: in.SendMailFromAddressOption,
	}

	if strings.TrimSpace(in.WebAppID) != "" {
		target := &wasScheduleTargetWire{
			WebApp:       &wasIDRefWire{ID: json.Number(in.WebAppID)},
			CancelOption: in.CancelOption,
		}
		switch {
		case strings.TrimSpace(in.WebAppAuthRecordID) != "":
			target.WebAppAuthRecord = &wasAuthRecordRefWire{ID: json.Number(in.WebAppAuthRecordID)}
		case in.WebAppAuthRecordIsDefault:
			target.WebAppAuthRecord = &wasAuthRecordRefWire{IsDefault: boolPtr(true)}
		}
		if strings.TrimSpace(in.ScannerType) != "" {
			target.ScannerAppliance = &wasScannerApplianceWire{
				Type:         in.ScannerType,
				FriendlyName: in.ScannerFriendlyName,
			}
		}
		w.Target = target
	}
	if strings.TrimSpace(in.OptionProfileID) != "" {
		w.Profile = &wasIDRefWire{ID: json.Number(in.OptionProfileID)}
	}
	if strings.TrimSpace(in.ProxyID) != "" {
		w.Proxy = &wasIDRefWire{ID: json.Number(in.ProxyID)}
	}
	if strings.TrimSpace(in.DNSOverrideID) != "" {
		w.DNSOverride = &wasIDRefWire{ID: json.Number(in.DNSOverrideID)}
	}

	scheduling := &wasSchedulingWire{
		CancelAfterNHours: in.CancelAfterHours,
		StartDate:         in.StartDate,
		OccurrenceType:    in.OccurrenceType,
		Occurrence:        wasScheduleOccurrenceWireFrom(in.Recurrence),
	}
	if strings.TrimSpace(in.TimeZoneCode) != "" {
		scheduling.TimeZone = &wasTimeZoneWire{Code: in.TimeZoneCode}
	}
	w.Scheduling = scheduling

	if in.Notification != nil {
		n := &wasNotificationWire{
			Active:     boolPtr(in.Notification.Active),
			Reschedule: boolPtr(in.Notification.Reschedule),
			Message:    in.Notification.Message,
		}
		if in.Notification.DelayAmount > 0 {
			n.Delay = &wasDelayWire{NB: in.Notification.DelayAmount, Scale: in.Notification.DelayScale}
		}
		if len(in.Notification.Recipients) > 0 {
			n.Recipients = &wasEmailListWire{Set: &wasEmailSetWire{EmailAddress: in.Notification.Recipients}}
		}
		w.Notification = n
	}

	return w
}

func (w *wasScanScheduleWire) toWASScanSchedule() *WASScanSchedule {
	out := &WASScanSchedule{
		ID:                        w.ID.String(),
		Name:                      w.Name,
		Type:                      w.Type,
		SendMailFromAddressOption: w.SendMailFromAddressOption,
		Created:                   w.Created,
	}
	if w.Active != nil {
		out.Active = *w.Active
	}
	if w.Profile != nil {
		out.OptionProfileID = w.Profile.ID.String()
	}
	if w.Proxy != nil {
		out.ProxyID = w.Proxy.ID.String()
	}
	if w.DNSOverride != nil {
		out.DNSOverrideID = w.DNSOverride.ID.String()
	}
	if w.SendMail != nil {
		out.SendMail = *w.SendMail
	}
	if w.SendOneMail != nil {
		out.SendOneMail = *w.SendOneMail
	}

	if w.Target != nil {
		if w.Target.WebApp != nil {
			out.WebAppID = w.Target.WebApp.ID.String()
		}
		if w.Target.WebAppAuthRecord != nil {
			out.WebAppAuthRecordID = w.Target.WebAppAuthRecord.ID.String()
			if w.Target.WebAppAuthRecord.IsDefault != nil {
				out.WebAppAuthRecordIsDefault = *w.Target.WebAppAuthRecord.IsDefault
			}
		}
		if w.Target.ScannerAppliance != nil {
			out.ScannerType = w.Target.ScannerAppliance.Type
			out.ScannerFriendlyName = w.Target.ScannerAppliance.FriendlyName
		}
		out.CancelOption = w.Target.CancelOption
	}

	if w.Scheduling != nil {
		out.StartDate = w.Scheduling.StartDate
		out.OccurrenceType = w.Scheduling.OccurrenceType
		out.CancelAfterHours = w.Scheduling.CancelAfterNHours
		if w.Scheduling.TimeZone != nil {
			out.TimeZoneCode = w.Scheduling.TimeZone.Code
		}
		out.Recurrence = wasScheduleRecurrenceFromOccurrenceWire(w.Scheduling.Occurrence)
	}

	if w.Notification != nil {
		n := &WASScheduleNotification{Message: w.Notification.Message}
		if w.Notification.Active != nil {
			n.Active = *w.Notification.Active
		}
		if w.Notification.Reschedule != nil {
			n.Reschedule = *w.Notification.Reschedule
		}
		if w.Notification.Delay != nil {
			n.DelayAmount = w.Notification.Delay.NB
			n.DelayScale = w.Notification.Delay.Scale
		}
		if w.Notification.Recipients != nil {
			set := w.Notification.Recipients.List
			if set == nil {
				set = w.Notification.Recipients.Set
			}
			if set != nil {
				n.Recipients = set.EmailAddress
			}
		}
		out.Notification = n
	}

	return out
}

// CreateWASScanSchedule creates a WAS scan schedule and returns it.
func (c *Client) CreateWASScanSchedule(ctx context.Context, in WASScanScheduleInput) (*WASScanSchedule, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("qualys qps: WAS scan schedule name is required")
	}
	if strings.TrimSpace(in.WebAppID) == "" {
		return nil, fmt.Errorf("qualys qps: WAS scan schedule requires a web application target")
	}
	if strings.TrimSpace(in.StartDate) == "" || strings.TrimSpace(in.TimeZoneCode) == "" || strings.TrimSpace(in.OccurrenceType) == "" {
		return nil, fmt.Errorf("qualys qps: WAS scan schedule requires start_date, time_zone_code and occurrence_type")
	}

	var resp ServiceResponse
	err := c.call(ctx, http.MethodPost, "/qps/rest/3.0/create/was/wasscanschedule",
		&ServiceRequest{Data: wasScanScheduleData{WasScanSchedule: wasScanScheduleInputToWire(in)}}, &resp, true)
	if err != nil {
		return nil, err
	}
	schedules, err := decodeWASScanSchedules(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(schedules) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS scan schedule was created but the API returned no data; " +
			"import it by ID to bring it under management")
	}
	return schedules[0], nil
}

// UpdateWASScanSchedule applies desired state to an existing WAS scan
// schedule. A user-supplied example notes that update only requires the
// fields being changed, but this sends the full desired state on every
// call, consistent with every other update encoder in this package — the
// server accepts a superset of the minimal patch it documents.
func (c *Client) UpdateWASScanSchedule(ctx context.Context, id string, in WASScanScheduleInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS scan schedule id is required for update")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/update/was/wasscanschedule/"+id,
		&ServiceRequest{Data: wasScanScheduleData{WasScanSchedule: wasScanScheduleInputToWire(in)}}, nil, false)
}

// ActivateWASScanSchedule activates a schedule, confirmed as its own
// dedicated endpoint (not "via update", an earlier open question this
// package's own discovery notes recorded).
func (c *Client) ActivateWASScanSchedule(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS scan schedule id is required")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/activate/was/wasscanschedule/"+id, nil, nil, false)
}

// DeactivateWASScanSchedule deactivates a schedule (retained, but excluded
// from future scheduled launches), confirmed as its own dedicated endpoint.
func (c *Client) DeactivateWASScanSchedule(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS scan schedule id is required")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/deactivate/was/wasscanschedule/"+id, nil, nil, false)
}

// DeleteWASScanSchedule removes a WAS scan schedule.
func (c *Client) DeleteWASScanSchedule(ctx context.Context, id string) error {
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/delete/was/wasscanschedule/"+id, nil, nil, true)
}

// GetWASScanSchedule returns one WAS scan schedule, or ErrNotFound.
func (c *Client) GetWASScanSchedule(ctx context.Context, id string) (*WASScanSchedule, error) {
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, "/qps/rest/3.0/get/was/wasscanschedule/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	schedules, err := decodeWASScanSchedules(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(schedules) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS scan schedule %s: %w", id, ErrNotFound)
	}
	return schedules[0], nil
}

// SearchWASScanSchedules returns WAS scan schedules matching filters, following pagination.
func (c *Client) SearchWASScanSchedules(ctx context.Context, filters *Filters) ([]*WASScanSchedule, error) {
	var all []*WASScanSchedule
	err := c.SearchAll(ctx, "/qps/rest/3.0/search/was/wasscanschedule", filters, 100, 0,
		func(raw json.RawMessage) error {
			schedules, err := decodeWASScanSchedules(raw)
			if err != nil {
				return err
			}
			all = append(all, schedules...)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return all, nil
}

func decodeWASScanSchedules(raw json.RawMessage) ([]*WASScanSchedule, error) {
	items := decodeListOrSingle(raw)
	out := make([]*WASScanSchedule, 0, len(items))
	for _, item := range items {
		var d wasScanScheduleData
		if err := json.Unmarshal(item, &d); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding WAS scan schedule data: %w", err)
		}
		if d.WasScanSchedule != nil {
			out = append(out, d.WasScanSchedule.toWASScanSchedule())
		}
	}
	return out, nil
}
