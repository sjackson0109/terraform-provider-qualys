package vmdr

import (
	"context"
	"net/url"
	"strings"
)

// Report is one generated VM/PA report, as returned by the report list
// endpoint.
//
// Launching, cancelling, fetching and deleting a report are deliberately
// not modelled by this package: doc 06 classifies them as imperative
// operations (a report is a job, not a resource with a stable lifecycle),
// consistent with how this package already treats VM scans. This type
// supports only a read-only qualys_reports data source, for example to
// poll a launched report's status or find the ID of a report to download
// outside Terraform.
//
// Evidence tier: Corroborated. Doc 03 §10 confirms the list action, its
// request filters (id, state), the response DTD name
// (report_list_output.dtd) and one specific response field name,
// EXPIRATION_DATETIME. The remaining fields follow the same
// request/response naming convention this codebase has confirmed
// elsewhere (ID, TITLE, TYPE, USER_LOGIN, OUTPUT_FORMAT, a nested
// STATUS>STATE matching the shape already confirmed for report
// schedules) — not fabricated, but unverified against a tenant beyond the
// one field doc 03 names directly.
type Report struct {
	ID                 string
	Title              string
	Type               string
	UserLogin          string
	LaunchDatetime     string
	OutputFormat       string
	Size               string
	Status             string
	ExpirationDatetime string
}

type reportListOutput struct {
	Response struct {
		Reports []struct {
			ID             string `xml:"ID"`
			Title          string `xml:"TITLE"`
			Type           string `xml:"TYPE"`
			UserLogin      string `xml:"USER_LOGIN"`
			LaunchDatetime string `xml:"LAUNCH_DATETIME"`
			OutputFormat   string `xml:"OUTPUT_FORMAT"`
			Size           string `xml:"SIZE"`
			Status         struct {
				State string `xml:"STATE"`
			} `xml:"STATUS"`
			ExpirationDatetime string `xml:"EXPIRATION_DATETIME"`
		} `xml:"REPORT_LIST>REPORT"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *reportListOutput) warning() *warning { return o.Response.Warning }

func (o *reportListOutput) reports() []*Report {
	out := make([]*Report, 0, len(o.Response.Reports))
	for _, r := range o.Response.Reports {
		out = append(out, &Report{
			ID:                 strings.TrimSpace(r.ID),
			Title:              r.Title,
			Type:               r.Type,
			UserLogin:          r.UserLogin,
			LaunchDatetime:     r.LaunchDatetime,
			OutputFormat:       r.OutputFormat,
			Size:               r.Size,
			Status:             r.Status.State,
			ExpirationDatetime: r.ExpirationDatetime,
		})
	}
	return out
}

// ReportFilter narrows a report list request.
type ReportFilter struct {
	ID string
	// State is one of Running, Finished, Submitted, Canceled, Errors
	// (Confirmed, doc 03 §10).
	State string
}

// ListReports returns generated reports matching filter, following
// truncation.
func (c *Client) ListReports(ctx context.Context, filter ReportFilter) ([]*Report, error) {
	params := url.Values{}
	setIfNotEmpty(params, "id", filter.ID)
	setIfNotEmpty(params, "state", filter.State)

	var all []*Report
	err := c.listAll(ctx, request{
		capability: capReport,
		path:       "report/",
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(reportListOutput) },
		func(p paginated) error {
			all = append(all, p.(*reportListOutput).reports()...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}
