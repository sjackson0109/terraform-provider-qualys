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

// WASScanSchedule is a recurring WAS scan.
type WASScanSchedule struct {
	ID                 string
	Name               string
	Type               string
	WebAppID           string
	OptionProfileID    string
	StartDate          string
	TimeZone           string
	OccurrenceType     string
	Notification       bool
	Reschedule         bool
	RandomizeScan      bool
	WebAppAuthRecordID string
	ProxyID            string
	DNSOverrideID      string
	CancelOption       string
	SendMail           bool
	Active             bool
	Created            string
}

// WASScanScheduleInput is the desired state of a WAS scan schedule.
//
// This models only the elements doc 11 (this provider's discovery notes,
// §8) confirms by name from the official Quick Reference: name,
// target.webApp.id, type, profile.id, startDate, timeZone, occurrenceType,
// notification, reschedule, and the simple optional id/boolean references.
// Tag-based (multi-web-app) targeting and the scannerAppliance sub-object
// are not modelled — their exact wire nesting was only described in prose,
// never seen in a sample payload, and stacking an unverified nesting on top
// of an already-partial resource was judged too much unverified surface for
// one pass.
//
// The per-occurrence recurrence detail (every N days/weeks/months, which
// weekdays, which day of month) is NOT modelled: doc 11 confirms
// occurrenceType itself but explicitly could not find the frequency
// sub-element names anywhere reachable during discovery, including a fourth
// pass that reached several mirrors of the WAS API User Guide, all blocked
// by this environment's network egress policy. A DAILY/WEEKLY/MONTHLY
// schedule created here uses whatever default cadence Qualys applies when
// those fields are omitted from the request — which has not been observed
// against a live tenant. Verify the resulting cadence in the Qualys UI after
// the first apply.
type WASScanScheduleInput struct {
	Name               string
	Type               string
	WebAppID           string
	OptionProfileID    string
	StartDate          string
	TimeZone           string
	OccurrenceType     string
	Notification       bool
	Reschedule         bool
	RandomizeScan      bool
	WebAppAuthRecordID string
	ProxyID            string
	DNSOverrideID      string
	CancelOption       string
	SendMail           bool
}

type wasIDRefWire struct {
	ID json.Number `json:"id,omitempty"`
}

type wasScanScheduleTargetWire struct {
	WebApp           *wasIDRefWire `json:"webApp,omitempty"`
	AuthRecordOption string        `json:"authRecordOption,omitempty"`
	RandomizeScan    *bool         `json:"randomizeScan,omitempty"`
	WebAppAuthRecord *wasIDRefWire `json:"webAppAuthRecord,omitempty"`
}

type wasScanScheduleWire struct {
	ID             json.Number                `json:"id,omitempty"`
	Name           string                     `json:"name,omitempty"`
	Type           string                     `json:"type,omitempty"`
	Target         *wasScanScheduleTargetWire `json:"target,omitempty"`
	Profile        *wasIDRefWire              `json:"profile,omitempty"`
	StartDate      string                     `json:"startDate,omitempty"`
	TimeZone       string                     `json:"timeZone,omitempty"`
	OccurrenceType string                     `json:"occurrenceType,omitempty"`
	Notification   *bool                      `json:"notification,omitempty"`
	Reschedule     *bool                      `json:"reschedule,omitempty"`
	Proxy          *wasIDRefWire              `json:"proxy,omitempty"`
	DNSOverride    *wasIDRefWire              `json:"dnsOverride,omitempty"`
	CancelOption   string                     `json:"cancelOption,omitempty"`
	SendMail       *bool                      `json:"sendMail,omitempty"`
	Active         *bool                      `json:"active,omitempty"`
	Created        string                     `json:"createdDate,omitempty"`
}

// wasScanScheduleData is the "data" envelope. The wrapper key WasScanSchedule
// follows this API's naming convention (WasOptionProfile for
// was/optionprofile) rather than a confirmed sample payload — no source
// reached during discovery showed one. If Qualys rejects it, the fix is a
// one-line rename here, not a schema redesign.
type wasScanScheduleData struct {
	WasScanSchedule *wasScanScheduleWire `json:"WasScanSchedule,omitempty"`
}

func boolPtr(b bool) *bool { return &b }

func wasScanScheduleInputToWire(in WASScanScheduleInput) *wasScanScheduleWire {
	w := &wasScanScheduleWire{
		Name:           in.Name,
		Type:           in.Type,
		StartDate:      in.StartDate,
		TimeZone:       in.TimeZone,
		OccurrenceType: in.OccurrenceType,
		Notification:   boolPtr(in.Notification),
		Reschedule:     boolPtr(in.Reschedule),
		CancelOption:   in.CancelOption,
		SendMail:       boolPtr(in.SendMail),
	}

	if strings.TrimSpace(in.WebAppID) != "" {
		target := &wasScanScheduleTargetWire{
			WebApp:        &wasIDRefWire{ID: json.Number(in.WebAppID)},
			RandomizeScan: boolPtr(in.RandomizeScan),
		}
		if strings.TrimSpace(in.WebAppAuthRecordID) != "" {
			target.WebAppAuthRecord = &wasIDRefWire{ID: json.Number(in.WebAppAuthRecordID)}
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

	return w
}

func (w *wasScanScheduleWire) toWASScanSchedule() *WASScanSchedule {
	out := &WASScanSchedule{
		ID:             w.ID.String(),
		Name:           w.Name,
		Type:           w.Type,
		StartDate:      w.StartDate,
		TimeZone:       w.TimeZone,
		OccurrenceType: w.OccurrenceType,
		CancelOption:   w.CancelOption,
		Created:        w.Created,
	}
	if w.Profile != nil {
		out.OptionProfileID = w.Profile.ID.String()
	}
	if w.Target != nil {
		if w.Target.WebApp != nil {
			out.WebAppID = w.Target.WebApp.ID.String()
		}
		if w.Target.WebAppAuthRecord != nil {
			out.WebAppAuthRecordID = w.Target.WebAppAuthRecord.ID.String()
		}
		if w.Target.RandomizeScan != nil {
			out.RandomizeScan = *w.Target.RandomizeScan
		}
	}
	if w.Proxy != nil {
		out.ProxyID = w.Proxy.ID.String()
	}
	if w.DNSOverride != nil {
		out.DNSOverrideID = w.DNSOverride.ID.String()
	}
	if w.Notification != nil {
		out.Notification = *w.Notification
	}
	if w.Reschedule != nil {
		out.Reschedule = *w.Reschedule
	}
	if w.SendMail != nil {
		out.SendMail = *w.SendMail
	}
	if w.Active != nil {
		out.Active = *w.Active
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
	if strings.TrimSpace(in.StartDate) == "" || strings.TrimSpace(in.TimeZone) == "" || strings.TrimSpace(in.OccurrenceType) == "" {
		return nil, fmt.Errorf("qualys qps: WAS scan schedule requires start_date, time_zone and occurrence_type")
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

// UpdateWASScanSchedule applies desired state to an existing WAS scan schedule.
func (c *Client) UpdateWASScanSchedule(ctx context.Context, id string, in WASScanScheduleInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS scan schedule id is required for update")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/update/was/wasscanschedule/"+id,
		&ServiceRequest{Data: wasScanScheduleData{WasScanSchedule: wasScanScheduleInputToWire(in)}}, nil, false)
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
