package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// WAS report output formats.
const (
	WASReportFormatPDF  = "PDF"
	WASReportFormatHTML = "HTML"
	WASReportFormatXML  = "XML"
	WASReportFormatCSV  = "CSV"
)

// WASReportSchedule is a recurring, automated WAS report generation and
// delivery — distinct from qualys_was_scan_schedule, which runs scans. Both
// share the same startDate/timeZone/occurrenceType/occurrence structure,
// confirmed by a user-supplied "Create and Update Report Schedule"
// walkthrough (transcribed, not a verbatim PDF quote, so treated as strong
// but not absolute evidence). This walkthrough is also the source of the
// MONTHLY occurrence shape (dayOfMonth/everyNMonths) applied by analogy to
// qualys_was_scan_schedule.
type WASReportSchedule struct {
	ID               string
	Name             string
	Active           bool
	ReportTemplateID string
	WebAppID         string
	OutputFormat     string
	Recipients       []string

	StartDate      string
	TimeZoneCode   string
	OccurrenceType string
	Recurrence     *WASScheduleRecurrence

	Created string
}

// WASReportScheduleInput is the desired state of a WAS report schedule.
type WASReportScheduleInput struct {
	Name             string
	Active           bool
	ReportTemplateID string
	WebAppID         string
	OutputFormat     string
	Recipients       []string

	StartDate      string
	TimeZoneCode   string
	OccurrenceType string
	Recurrence     *WASScheduleRecurrence
}

type wasReportScheduleTargetWire struct {
	WebApp *wasIDRefWire `json:"webApp,omitempty"`
}

type wasReportOutputWire struct {
	Format string `json:"format,omitempty"`
}

type wasReportNotificationWire struct {
	Recipients *wasEmailListWire `json:"recipients,omitempty"`
}

type wasReportScheduleScheduleWire struct {
	StartDate      string             `json:"startDate,omitempty"`
	TimeZone       *wasTimeZoneWire   `json:"timeZone,omitempty"`
	OccurrenceType string             `json:"occurrenceType,omitempty"`
	Occurrence     *wasOccurrenceWire `json:"occurrence,omitempty"`
}

type wasReportScheduleWire struct {
	ID             json.Number                    `json:"id,omitempty"`
	Name           string                         `json:"name,omitempty"`
	Active         *bool                          `json:"active,omitempty"`
	ReportTemplate *wasIDRefWire                  `json:"reportTemplate,omitempty"`
	Target         *wasReportScheduleTargetWire   `json:"target,omitempty"`
	Output         *wasReportOutputWire           `json:"output,omitempty"`
	Notification   *wasReportNotificationWire     `json:"notification,omitempty"`
	Schedule       *wasReportScheduleScheduleWire `json:"schedule,omitempty"`
	Created        string                         `json:"createdDate,omitempty"`
}

// wasReportScheduleData is the "data" envelope. The wrapper key
// ReportSchedule is Confirmed — seen verbatim in a user-supplied create
// example, unlike the WasOptionProfile naming-convention guess that turned
// out wrong.
type wasReportScheduleData struct {
	ReportSchedule *wasReportScheduleWire `json:"ReportSchedule,omitempty"`
}

func wasScheduleOccurrenceWireFrom(rec *WASScheduleRecurrence) *wasOccurrenceWire {
	if rec == nil {
		return nil
	}
	occ := &wasOccurrenceWire{}
	if rec.EveryNDays > 0 {
		occ.DailyOccurrence = &wasDailyOccurrenceWire{EveryNDays: rec.EveryNDays}
	}
	if rec.EveryNWeeks > 0 || len(rec.OnDays) > 0 {
		weekly := &wasWeeklyOccurrenceWire{
			EveryNWeeks:     rec.EveryNWeeks,
			OccurrenceCount: rec.OccurrenceCount,
		}
		if len(rec.OnDays) > 0 {
			weekly.OnDays = &wasWeekDayListWire{WeekDay: rec.OnDays}
		}
		occ.WeeklyOccurrence = weekly
	}
	if rec.DayOfMonth > 0 || rec.EveryNMonths > 0 {
		occ.MonthlyOccurrence = &wasMonthlyOccurrenceWire{
			DayOfMonth:   rec.DayOfMonth,
			EveryNMonths: rec.EveryNMonths,
		}
	}
	if occ.DailyOccurrence == nil && occ.WeeklyOccurrence == nil && occ.MonthlyOccurrence == nil {
		return nil
	}
	return occ
}

func wasScheduleRecurrenceFromOccurrenceWire(occ *wasOccurrenceWire) *WASScheduleRecurrence {
	if occ == nil {
		return nil
	}
	rec := &WASScheduleRecurrence{}
	if d := occ.DailyOccurrence; d != nil {
		rec.EveryNDays = d.EveryNDays
	}
	if wk := occ.WeeklyOccurrence; wk != nil {
		rec.EveryNWeeks = wk.EveryNWeeks
		rec.OccurrenceCount = wk.OccurrenceCount
		if wk.OnDays != nil {
			rec.OnDays = wk.OnDays.WeekDay
		}
	}
	if mo := occ.MonthlyOccurrence; mo != nil {
		rec.DayOfMonth = mo.DayOfMonth
		rec.EveryNMonths = mo.EveryNMonths
	}
	return rec
}

func wasReportScheduleInputToWire(in WASReportScheduleInput) *wasReportScheduleWire {
	w := &wasReportScheduleWire{
		Name:   in.Name,
		Active: boolPtr(in.Active),
	}
	if strings.TrimSpace(in.ReportTemplateID) != "" {
		w.ReportTemplate = &wasIDRefWire{ID: json.Number(in.ReportTemplateID)}
	}
	if strings.TrimSpace(in.WebAppID) != "" {
		w.Target = &wasReportScheduleTargetWire{WebApp: &wasIDRefWire{ID: json.Number(in.WebAppID)}}
	}
	if strings.TrimSpace(in.OutputFormat) != "" {
		w.Output = &wasReportOutputWire{Format: in.OutputFormat}
	}
	if len(in.Recipients) > 0 {
		w.Notification = &wasReportNotificationWire{
			Recipients: &wasEmailListWire{Set: &wasEmailSetWire{EmailAddress: in.Recipients}},
		}
	}

	sched := &wasReportScheduleScheduleWire{
		StartDate:      in.StartDate,
		OccurrenceType: in.OccurrenceType,
		Occurrence:     wasScheduleOccurrenceWireFrom(in.Recurrence),
	}
	if strings.TrimSpace(in.TimeZoneCode) != "" {
		sched.TimeZone = &wasTimeZoneWire{Code: in.TimeZoneCode}
	}
	w.Schedule = sched

	return w
}

func (w *wasReportScheduleWire) toWASReportSchedule() *WASReportSchedule {
	out := &WASReportSchedule{
		ID:      w.ID.String(),
		Name:    w.Name,
		Created: w.Created,
	}
	if w.Active != nil {
		out.Active = *w.Active
	}
	if w.ReportTemplate != nil {
		out.ReportTemplateID = w.ReportTemplate.ID.String()
	}
	if w.Target != nil && w.Target.WebApp != nil {
		out.WebAppID = w.Target.WebApp.ID.String()
	}
	if w.Output != nil {
		out.OutputFormat = w.Output.Format
	}
	if w.Notification != nil && w.Notification.Recipients != nil {
		set := w.Notification.Recipients.List
		if set == nil {
			set = w.Notification.Recipients.Set
		}
		if set != nil {
			out.Recipients = set.EmailAddress
		}
	}
	if w.Schedule != nil {
		out.StartDate = w.Schedule.StartDate
		out.OccurrenceType = w.Schedule.OccurrenceType
		if w.Schedule.TimeZone != nil {
			out.TimeZoneCode = w.Schedule.TimeZone.Code
		}
		out.Recurrence = wasScheduleRecurrenceFromOccurrenceWire(w.Schedule.Occurrence)
	}
	return out
}

// CreateWASReportSchedule creates a WAS report schedule and returns it.
func (c *Client) CreateWASReportSchedule(ctx context.Context, in WASReportScheduleInput) (*WASReportSchedule, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("qualys qps: WAS report schedule name is required")
	}
	if strings.TrimSpace(in.ReportTemplateID) == "" {
		return nil, fmt.Errorf("qualys qps: WAS report schedule requires a report template")
	}
	if strings.TrimSpace(in.WebAppID) == "" {
		return nil, fmt.Errorf("qualys qps: WAS report schedule requires a web application target")
	}
	if strings.TrimSpace(in.StartDate) == "" || strings.TrimSpace(in.TimeZoneCode) == "" || strings.TrimSpace(in.OccurrenceType) == "" {
		return nil, fmt.Errorf("qualys qps: WAS report schedule requires start_date, time_zone_code and occurrence_type")
	}

	var resp ServiceResponse
	err := c.call(ctx, http.MethodPost, "/qps/rest/3.0/create/was/reportschedule",
		&ServiceRequest{Data: wasReportScheduleData{ReportSchedule: wasReportScheduleInputToWire(in)}}, &resp, true)
	if err != nil {
		return nil, err
	}
	schedules, err := decodeWASReportSchedules(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(schedules) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS report schedule was created but the API returned no data; " +
			"import it by ID to bring it under management")
	}
	return schedules[0], nil
}

// UpdateWASReportSchedule applies desired state to an existing WAS report schedule.
func (c *Client) UpdateWASReportSchedule(ctx context.Context, id string, in WASReportScheduleInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS report schedule id is required for update")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/update/was/reportschedule/"+id,
		&ServiceRequest{Data: wasReportScheduleData{ReportSchedule: wasReportScheduleInputToWire(in)}}, nil, false)
}

// ActivateWASReportSchedule activates a report schedule via its dedicated endpoint.
func (c *Client) ActivateWASReportSchedule(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS report schedule id is required")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/activate/was/reportschedule/"+id, nil, nil, false)
}

// DeactivateWASReportSchedule deactivates a report schedule via its dedicated endpoint.
func (c *Client) DeactivateWASReportSchedule(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS report schedule id is required")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/deactivate/was/reportschedule/"+id, nil, nil, false)
}

// DeleteWASReportSchedule removes a WAS report schedule.
func (c *Client) DeleteWASReportSchedule(ctx context.Context, id string) error {
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/delete/was/reportschedule/"+id, nil, nil, true)
}

// GetWASReportSchedule returns one WAS report schedule, or ErrNotFound.
func (c *Client) GetWASReportSchedule(ctx context.Context, id string) (*WASReportSchedule, error) {
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, "/qps/rest/3.0/get/was/reportschedule/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	schedules, err := decodeWASReportSchedules(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(schedules) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS report schedule %s: %w", id, ErrNotFound)
	}
	return schedules[0], nil
}

// SearchWASReportSchedules returns WAS report schedules matching filters, following pagination.
func (c *Client) SearchWASReportSchedules(ctx context.Context, filters *Filters) ([]*WASReportSchedule, error) {
	var all []*WASReportSchedule
	err := c.SearchAll(ctx, "/qps/rest/3.0/search/was/reportschedule", filters, 100, 0,
		func(raw json.RawMessage) error {
			schedules, err := decodeWASReportSchedules(raw)
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

func decodeWASReportSchedules(raw json.RawMessage) ([]*WASReportSchedule, error) {
	items := decodeListOrSingle(raw)
	out := make([]*WASReportSchedule, 0, len(items))
	for _, item := range items {
		var d wasReportScheduleData
		if err := json.Unmarshal(item, &d); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding WAS report schedule data: %w", err)
		}
		if d.ReportSchedule != nil {
			out = append(out, d.ReportSchedule.toWASReportSchedule())
		}
	}
	return out, nil
}
