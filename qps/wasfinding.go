package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// WAS finding severity/type/status enums, confirmed by doc 11 §9 as the
// values the search filters accept.
const (
	WASFindingTypeVulnerability       = "VULNERABILITY"
	WASFindingTypeSensitiveContent    = "SENSITIVE_CONTENT"
	WASFindingTypeInformationGathered = "INFORMATION_GATHERED"

	WASFindingStatusNew       = "NEW"
	WASFindingStatusActive    = "ACTIVE"
	WASFindingStatusReopened  = "REOPENED"
	WASFindingStatusFixed     = "FIXED"
	WASFindingStatusProtected = "PROTECTED"

	WASFindingSourceQualys = "QUALYS"
	WASFindingSourceManual = "MANUAL"
	WASFindingSourceBurp   = "BURP"
)

// WAS finding retest states, Confirmed by a user-supplied "Gap Review"
// document quoting the official Qualys "Retrieve Finding Retest Status"
// reference.
const (
	WASRetestStatusNoRetest    = "NO_RETEST"
	WASRetestStatusUnderRetest = "UNDER_RETEST"
	WASRetestStatusRetested    = "RETESTED"
	WASRetestStatusCanceling   = "CANCELING"
	WASRetestStatusCanceled    = "CANCELED"
)

// WASFinding is a WAS scan finding: a vulnerability, sensitive-content
// exposure, or information-gathered item Qualys (or an imported Burp scan)
// reported against a web application.
//
// This is read-only in this package: the finding lifecycle actions
// (ignore/activate/updateSeverity/restoreSeverity/retest) are not
// implemented, only count/search/get.
type WASFinding struct {
	ID                string
	QID               string
	Name              string
	Type              string
	Severity          int
	Status            string
	FindingType       string
	URL               string
	WebAppID          string
	WebAppName        string
	FirstDetectedDate string
	LastDetectedDate  string
	IsIgnored         bool
	CVSSV3Base        float64
}

type wasFindingWebAppWire struct {
	ID   json.Number `json:"id,omitempty"`
	Name string      `json:"name,omitempty"`
}

type wasFindingCVSSV3Wire struct {
	Base float64 `json:"base,omitempty"`
}

type wasFindingWire struct {
	ID                json.Number           `json:"id,omitempty"`
	QID               json.Number           `json:"qid,omitempty"`
	Name              string                `json:"name,omitempty"`
	Type              string                `json:"type,omitempty"`
	Severity          int                   `json:"severity,omitempty"`
	Status            string                `json:"status,omitempty"`
	FindingType       string                `json:"findingType,omitempty"`
	URL               string                `json:"url,omitempty"`
	WebApp            *wasFindingWebAppWire `json:"webApp,omitempty"`
	FirstDetectedDate string                `json:"firstDetectedDate,omitempty"`
	LastDetectedDate  string                `json:"lastDetectedDate,omitempty"`
	IsIgnored         bool                  `json:"isIgnored,omitempty"`
	CVSSV3            *wasFindingCVSSV3Wire `json:"cvssV3,omitempty"`
}

type wasFindingData struct {
	Finding *wasFindingWire `json:"Finding,omitempty"`
}

func (w *wasFindingWire) toWASFinding() *WASFinding {
	out := &WASFinding{
		ID:                w.ID.String(),
		QID:               w.QID.String(),
		Name:              w.Name,
		Type:              w.Type,
		Severity:          w.Severity,
		Status:            w.Status,
		FindingType:       w.FindingType,
		URL:               w.URL,
		FirstDetectedDate: w.FirstDetectedDate,
		LastDetectedDate:  w.LastDetectedDate,
		IsIgnored:         w.IsIgnored,
	}
	if w.WebApp != nil {
		out.WebAppID = w.WebApp.ID.String()
		out.WebAppName = w.WebApp.Name
	}
	if w.CVSSV3 != nil {
		out.CVSSV3Base = w.CVSSV3.Base
	}
	return out
}

// WASFindingFilter narrows a finding search. Empty fields are omitted.
type WASFindingFilter struct {
	WebAppID    string
	Severity    string
	Status      string
	Type        string
	FindingType string
	IsIgnored   *bool
}

func (f WASFindingFilter) toCriteria() []Criterion {
	var out []Criterion
	if f.WebAppID != "" {
		out = append(out, Criterion{Field: "webApp.id", Operator: "EQUALS", Value: f.WebAppID})
	}
	if f.Severity != "" {
		out = append(out, Criterion{Field: "severity", Operator: "EQUALS", Value: f.Severity})
	}
	if f.Status != "" {
		out = append(out, Criterion{Field: "status", Operator: "EQUALS", Value: f.Status})
	}
	if f.Type != "" {
		out = append(out, Criterion{Field: "type", Operator: "EQUALS", Value: f.Type})
	}
	if f.FindingType != "" {
		out = append(out, Criterion{Field: "findingType", Operator: "EQUALS", Value: f.FindingType})
	}
	if f.IsIgnored != nil {
		v := "false"
		if *f.IsIgnored {
			v = "true"
		}
		out = append(out, Criterion{Field: "isIgnored", Operator: "EQUALS", Value: v})
	}
	return out
}

type wasFindingActionWire struct {
	Comment string `json:"comment,omitempty"`
}

type wasFindingActionData struct {
	Finding *wasFindingActionWire `json:"Finding,omitempty"`
}

// IgnoreWASFinding marks a finding as ignored (accepted risk, false
// positive, third-party issue, or compensating control), with comment as
// the audit-trail justification. Confirmed by a user-supplied "Findings
// Lifecycle Actions" walkthrough: POST ignore/was/finding/<id> with a
// data.Finding.comment body — the same Finding wrapper key already used for
// get/search.
func (c *Client) IgnoreWASFinding(ctx context.Context, id, comment string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS finding id is required")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/ignore/was/finding/"+id,
		&ServiceRequest{Data: wasFindingActionData{Finding: &wasFindingActionWire{Comment: comment}}}, nil, false)
}

// ReopenWASFinding reverses IgnoreWASFinding, returning a finding to active status.
func (c *Client) ReopenWASFinding(ctx context.Context, id, comment string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS finding id is required")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/reopen/was/finding/"+id,
		&ServiceRequest{Data: wasFindingActionData{Finding: &wasFindingActionWire{Comment: comment}}}, nil, false)
}

// FixWASFinding marks a finding as fixed. Deliberately not wrapped in a
// Terraform resource: the source walkthrough itself recommends against
// using lifecycle actions as a substitute for rescanning — "fixed" is
// meant to be scan-verified (a later scan can revert it if the
// vulnerability is still detected), not administratively declared the way
// "ignored" legitimately is. Exposed here for scripted use outside the
// declarative resource model.
func (c *Client) FixWASFinding(ctx context.Context, id, comment string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS finding id is required")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/fix/was/finding/"+id,
		&ServiceRequest{Data: wasFindingActionData{Finding: &wasFindingActionWire{Comment: comment}}}, nil, false)
}

type wasFindingRetestRequestWire struct {
	ID json.Number `json:"id,omitempty"`
}

type wasFindingRetestRequestData struct {
	Finding *wasFindingRetestRequestWire `json:"Finding,omitempty"`
}

// RetestWASFinding triggers an asynchronous retest of a finding (a
// potential vulnerability, confirmed vulnerability, or sensitive-content
// finding). Confirmed by a user-supplied "Gap Review" document quoting the
// official Qualys "Retest Finding" reference: POST retest/was/finding/<id>
// with a data.Finding.id body. Requires the WAS.VULN.RETEST permission.
// Poll GetWASFindingRetestStatus for progress — this call only starts the
// retest, it does not wait for it.
func (c *Client) RetestWASFinding(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS finding id is required")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/retest/was/finding/"+id,
		&ServiceRequest{Data: wasFindingRetestRequestData{Finding: &wasFindingRetestRequestWire{ID: json.Number(id)}}},
		nil, false)
}

// WASFindingRetestStatus is the current state of a finding's asynchronous
// retest, as returned by GetWASFindingRetestStatus. RetestStatus is one of
// the WASRetestStatus* constants; not validated client-side.
type WASFindingRetestStatus struct {
	ID            string
	UniqueID      string
	RetestStatus  string
	RetestedDate  string
	FindingStatus string
	Reason        string
}

type wasFindingRetestDetailWire struct {
	RetestStatus  string `json:"retestStatus,omitempty"`
	RetestedDate  string `json:"retestedDate,omitempty"`
	FindingStatus string `json:"findingStatus,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

type wasFindingRetestStatusWire struct {
	ID       json.Number                 `json:"id,omitempty"`
	UniqueID string                      `json:"uniqueId,omitempty"`
	Retest   *wasFindingRetestDetailWire `json:"retest,omitempty"`
}

type wasFindingRetestStatusData struct {
	Finding *wasFindingRetestStatusWire `json:"Finding,omitempty"`
}

// GetWASFindingRetestStatus returns a finding's current retest status.
// Confirmed by the same "Gap Review" document quoting the official Qualys
// "Retrieve Finding Retest Status" reference: POST
// retestStatus/was/finding/<id> — a POST despite reading state, as the
// source documents it.
func (c *Client) GetWASFindingRetestStatus(ctx context.Context, id string) (*WASFindingRetestStatus, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("qualys qps: WAS finding id is required")
	}
	var resp ServiceResponse
	err := c.call(ctx, http.MethodPost, "/qps/rest/3.0/retestStatus/was/finding/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	items := decodeListOrSingle(resp.Data)
	if len(items) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS finding %s retest status: %w", id, ErrNotFound)
	}
	var d wasFindingRetestStatusData
	if err := json.Unmarshal(items[0], &d); err != nil {
		return nil, fmt.Errorf("qualys qps: decoding WAS finding retest status: %w", err)
	}
	if d.Finding == nil {
		return nil, fmt.Errorf("qualys qps: WAS finding %s retest status: %w", id, ErrNotFound)
	}
	out := &WASFindingRetestStatus{ID: d.Finding.ID.String(), UniqueID: d.Finding.UniqueID}
	if d.Finding.Retest != nil {
		out.RetestStatus = d.Finding.Retest.RetestStatus
		out.RetestedDate = d.Finding.Retest.RetestedDate
		out.FindingStatus = d.Finding.Retest.FindingStatus
		out.Reason = d.Finding.Retest.Reason
	}
	return out, nil
}

// GetWASFinding returns one WAS finding, or ErrNotFound.
func (c *Client) GetWASFinding(ctx context.Context, id string) (*WASFinding, error) {
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, "/qps/rest/3.0/get/was/finding/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	findings, err := decodeWASFindings(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(findings) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS finding %s: %w", id, ErrNotFound)
	}
	return findings[0], nil
}

// SearchWASFindings returns WAS findings matching filter, following pagination.
func (c *Client) SearchWASFindings(ctx context.Context, filter WASFindingFilter) ([]*WASFinding, error) {
	var filters *Filters
	if criteria := filter.toCriteria(); len(criteria) > 0 {
		filters = &Filters{Criteria: criteria}
	}

	var all []*WASFinding
	err := c.SearchAll(ctx, "/qps/rest/3.0/search/was/finding", filters, 100, 0,
		func(raw json.RawMessage) error {
			findings, err := decodeWASFindings(raw)
			if err != nil {
				return err
			}
			all = append(all, findings...)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return all, nil
}

func decodeWASFindings(raw json.RawMessage) ([]*WASFinding, error) {
	items := decodeListOrSingle(raw)
	out := make([]*WASFinding, 0, len(items))
	for _, item := range items {
		var d wasFindingData
		if err := json.Unmarshal(item, &d); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding WAS finding data: %w", err)
		}
		if d.Finding != nil {
			out = append(out, d.Finding.toWASFinding())
		}
	}
	return out, nil
}
