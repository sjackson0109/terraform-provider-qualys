package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
