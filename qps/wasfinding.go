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
// (ignore/activate/updateSeverity/restoreSeverity/retest) are implemented
// as their own dedicated methods (Ignore/Reopen/Fix/Retest WASFinding), not
// through this type.
//
// UniqueID is Confirmed via a different Finding-family call
// (GetWASFindingRetestStatus's response already decodes a Finding.uniqueId
// — the same object family this file's get/search calls decode) rather
// than a source specific to this file. CVSSV3Base is the only CVSS score
// this session found confirmed on the Finding object itself (doc 11 §9
// lists `cvssV3.base` as a Finding filter field, implying it is also a
// response field); a CVSS v2 score, if the API exposes one here at all, is
// not confirmed, so it is deliberately not modelled — CVSS v2 is still
// available indirectly via KnowledgeBase enrichment (see the
// qualys_was_findings data source), which is where Qualys's actual v2
// scoring data lives for many objects in this API family. Richer fields a
// user-supplied requirements document asked for — category, description,
// impact, solution, payload/request/response, parameter, authentication,
// ssl, access_path, external_reference, owasp/wasc category, CWE — have no
// confirmed representation on this object in this session's evidence and
// are deliberately not implemented rather than guessed; see doc 08 for the
// open gap.
type WASFinding struct {
	ID                string
	UniqueID          string
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
	UniqueID          string                `json:"uniqueId,omitempty"`
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
		UniqueID:          w.UniqueID,
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
//
// ID/QID/WebAppID/Severity/Status/Type/FindingType/IsIgnored are all
// EQUALS criteria on fields doc 11 §9 lists as Confirmed Finding filters.
// The four date bounds use GREATER/LESSER, which doc 11 §9 documents as
// generally valid qps operators (the same file's SearchAll already relies
// on GREATER working for its own pagination cursor) applied to
// firstDetectedDate/lastDetectedDate — themselves Confirmed filterable
// fields — but the operator/field combination itself is Corroborated, not
// independently Confirmed, since "not every operator is valid on every
// field" per the same doc. Verify against a tenant before relying on the
// date bounds specifically.
type WASFindingFilter struct {
	ID          string
	WebAppID    string
	QID         string
	Severity    string
	Status      string
	Type        string
	FindingType string
	IsIgnored   *bool

	FirstDetectedAfter  string
	FirstDetectedBefore string
	LastDetectedAfter   string
	LastDetectedBefore  string
}

func (f WASFindingFilter) toCriteria() []Criterion {
	var out []Criterion
	if f.ID != "" {
		out = append(out, Criterion{Field: "id", Operator: "EQUALS", Value: f.ID})
	}
	if f.WebAppID != "" {
		out = append(out, Criterion{Field: "webApp.id", Operator: "EQUALS", Value: f.WebAppID})
	}
	if f.QID != "" {
		out = append(out, Criterion{Field: "qid", Operator: "EQUALS", Value: f.QID})
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
	if f.FirstDetectedAfter != "" {
		out = append(out, Criterion{Field: "firstDetectedDate", Operator: "GREATER", Value: f.FirstDetectedAfter})
	}
	if f.FirstDetectedBefore != "" {
		out = append(out, Criterion{Field: "firstDetectedDate", Operator: "LESSER", Value: f.FirstDetectedBefore})
	}
	if f.LastDetectedAfter != "" {
		out = append(out, Criterion{Field: "lastDetectedDate", Operator: "GREATER", Value: f.LastDetectedAfter})
	}
	if f.LastDetectedBefore != "" {
		out = append(out, Criterion{Field: "lastDetectedDate", Operator: "LESSER", Value: f.LastDetectedBefore})
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

// WASFindingConflict records two findings decoded with the same ID that
// disagree on their other fields — a signal that a finding changed between
// pages of one search, or that pagination returned inconsistent data,
// rather than a true duplicate. Exact duplicates (identical in every
// field) are collapsed silently; this is only raised for a genuine
// conflict, so a caller can surface it rather than one side silently
// winning.
type WASFindingConflict struct {
	ID    string
	First *WASFinding
	Other *WASFinding
}

// SearchWASFindings returns WAS findings matching filter, following
// pagination. The Qualys finding ID is the deduplication key: an exact
// duplicate returned across a pagination boundary collapses to one record;
// a duplicate ID whose other fields disagree is reported in the returned
// conflicts slice (the first-seen record is kept) rather than silently
// resolved.
func (c *Client) SearchWASFindings(ctx context.Context, filter WASFindingFilter) ([]*WASFinding, []WASFindingConflict, error) {
	var filters *Filters
	if criteria := filter.toCriteria(); len(criteria) > 0 {
		filters = &Filters{Criteria: criteria}
	}

	seen := make(map[string]*WASFinding)
	var ordered []*WASFinding
	var conflicts []WASFindingConflict

	err := c.SearchAll(ctx, "/qps/rest/3.0/search/was/finding", filters, 100, 0,
		func(raw json.RawMessage) error {
			findings, err := decodeWASFindings(raw)
			if err != nil {
				return err
			}
			for _, f := range findings {
				existing, dup := seen[f.ID]
				if !dup {
					seen[f.ID] = f
					ordered = append(ordered, f)
					continue
				}
				if *existing != *f {
					conflicts = append(conflicts, WASFindingConflict{ID: f.ID, First: existing, Other: f})
				}
			}
			return nil
		})
	if err != nil {
		return nil, nil, err
	}
	return ordered, conflicts, nil
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
