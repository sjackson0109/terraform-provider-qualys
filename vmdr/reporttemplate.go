package vmdr

import (
	"context"
	"strings"
)

// ReportTemplate is a saved report template, as returned by the legacy V1
// report_template_list.php script — the only listing surface for report
// templates that exists (doc 09 §18). Template CRUD itself uses the
// versioned /api/2.0/fo/report/template/<type>/ family; this call is
// read-only and does not go through that family at all.
type ReportTemplate struct {
	ID           string
	Type         string
	TemplateType string
	Title        string
	OwnerLogin   string
	OwnerName    string
	LastUpdate   string
	Global       bool
}

type reportTemplateListOutput struct {
	Templates []struct {
		ID           string `xml:"ID"`
		Type         string `xml:"TYPE"`
		TemplateType string `xml:"TEMPLATE_TYPE"`
		Title        string `xml:"TITLE"`
		User         struct {
			Login     string `xml:"LOGIN"`
			FirstName string `xml:"FIRSTNAME"`
			LastName  string `xml:"LASTNAME"`
		} `xml:"USER"`
		LastUpdate string `xml:"LAST_UPDATE"`
		Global     string `xml:"GLOBAL"`
	} `xml:"REPORT_TEMPLATE"`
}

func (o *reportTemplateListOutput) templates() []*ReportTemplate {
	out := make([]*ReportTemplate, 0, len(o.Templates))
	for _, t := range o.Templates {
		out = append(out, &ReportTemplate{
			ID:           strings.TrimSpace(t.ID),
			Type:         t.Type,
			TemplateType: t.TemplateType,
			Title:        t.Title,
			OwnerLogin:   t.User.Login,
			OwnerName:    strings.TrimSpace(t.User.FirstName + " " + t.User.LastName),
			LastUpdate:   t.LastUpdate,
			Global:       t.Global == "1",
		})
	}
	return out
}

// ListReportTemplates returns every saved report template. The script takes
// no filter parameters and no truncation is documented for it — templates
// are few enough per subscription that this is not a concern in practice.
func (c *Client) ListReportTemplates(ctx context.Context) ([]*ReportTemplate, error) {
	var out reportTemplateListOutput
	if err := c.do(ctx, request{
		rawEndpoint: c.legacyEndpoint("report_template_list.php"),
		method:      "GET",
	}, &out); err != nil {
		return nil, err
	}
	return out.templates(), nil
}
