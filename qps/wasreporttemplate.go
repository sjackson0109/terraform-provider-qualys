package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// WAS report template types, confirmed by a user-supplied "Report
// Templates" walkthrough as the `type` filter's accepted values.
const (
	WASReportTemplateTypeWebApp    = "WAS_WEBAPP_REPORT"
	WASReportTemplateTypeScan      = "WAS_SCAN_REPORT"
	WASReportTemplateTypeScorecard = "WAS_SCORECARD_REPORT"
	WASReportTemplateTypeCatalog   = "WAS_CATALOG_REPORT"
)

// WASReportTemplate is a WAS report template.
//
// Read-only: the same walkthrough states the API supports count/search/get
// but not create/update/delete — templates are managed in the Qualys UI
// and referenced by ID when generating a report or a report schedule. Only
// id/name/type are modelled; the walkthrough describes richer template
// configuration (display settings, graph/grouping configuration, filters,
// search lists, included statuses) in prose only, with no field names, so
// this package does not attempt to decode it.
type WASReportTemplate struct {
	ID   string
	Name string
	Type string
}

type wasReportTemplateWire struct {
	ID   json.Number `json:"id,omitempty"`
	Name string      `json:"name,omitempty"`
	Type string      `json:"type,omitempty"`
}

// wasReportTemplateData is the "data" envelope. Unlike ReportSchedule
// (seen verbatim in a create example) or OptionProfile (a naming-convention
// guess that turned out wrong for that object), no worked example showed
// this wrapper key — "Get Report Template Details" is documented in prose
// only. ReportTemplate is chosen because three of the four wrapper keys
// confirmed so far in this codebase (WebApp, OptionProfile, ReportSchedule)
// have no "Was" prefix, against one that does (WasScanSchedule) — the more
// common pattern, not a confirmed fact. If wrong, Qualys rejects the
// request cleanly rather than silently misbehaving.
type wasReportTemplateData struct {
	ReportTemplate *wasReportTemplateWire `json:"ReportTemplate,omitempty"`
}

func (w *wasReportTemplateWire) toWASReportTemplate() *WASReportTemplate {
	return &WASReportTemplate{ID: w.ID.String(), Name: w.Name, Type: w.Type}
}

// GetWASReportTemplate returns one WAS report template, or ErrNotFound.
func (c *Client) GetWASReportTemplate(ctx context.Context, id string) (*WASReportTemplate, error) {
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, "/qps/rest/3.0/get/was/reporttemplate/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	templates, err := decodeWASReportTemplates(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS report template %s: %w", id, ErrNotFound)
	}
	return templates[0], nil
}

// SearchWASReportTemplates returns WAS report templates matching filters, following pagination.
func (c *Client) SearchWASReportTemplates(ctx context.Context, filters *Filters) ([]*WASReportTemplate, error) {
	var all []*WASReportTemplate
	err := c.SearchAll(ctx, "/qps/rest/3.0/search/was/reporttemplate", filters, 100, 0,
		func(raw json.RawMessage) error {
			templates, err := decodeWASReportTemplates(raw)
			if err != nil {
				return err
			}
			all = append(all, templates...)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return all, nil
}

func decodeWASReportTemplates(raw json.RawMessage) ([]*WASReportTemplate, error) {
	items := decodeListOrSingle(raw)
	out := make([]*WASReportTemplate, 0, len(items))
	for _, item := range items {
		var d wasReportTemplateData
		if err := json.Unmarshal(item, &d); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding WAS report template data: %w", err)
		}
		if d.ReportTemplate != nil {
			out = append(out, d.ReportTemplate.toWASReportTemplate())
		}
	}
	return out, nil
}
