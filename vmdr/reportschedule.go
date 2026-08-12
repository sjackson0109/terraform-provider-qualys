package vmdr

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// ReportSchedule is a scheduled report definition.
//
// Confirmed absence (doc 03 §10, doc 06 Deliverable 6): there is no
// create/update/delete/enable/disable API for report schedules at all —
// the Qualys documentation tree covers only `list` and `launch_now`, and
// this is GUI-only, not merely undocumented. A qualys_report_schedule
// *resource* is therefore impossible, not deferred; this type backs a
// read-only data source only, so a launch can be targeted by ID or a
// schedule's active state can be checked from Terraform.
//
// Evidence tier: Corroborated. Doc 03 §10 confirms the list action, its
// request filters (id, is_active) and the response DTD name
// (schedule_report_list_output.dtd). Field names below follow this
// codebase's established request/response naming convention; unverified
// against a tenant.
type ReportSchedule struct {
	ID            string
	Title         string
	Active        bool
	TemplateID    string
	TemplateTitle string
	OutputFormat  string
}

type reportScheduleListOutput struct {
	Response struct {
		Schedules []struct {
			ID     string `xml:"ID"`
			Title  string `xml:"TITLE"`
			Active string `xml:"ACTIVE"`
			Report struct {
				TemplateID    string `xml:"TEMPLATE_ID"`
				TemplateTitle string `xml:"TEMPLATE_TITLE"`
				OutputFormat  string `xml:"OUTPUT_FORMAT"`
			} `xml:"REPORT"`
		} `xml:"SCHEDULE_REPORT_LIST>SCHEDULE_REPORT"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *reportScheduleListOutput) warning() *warning { return o.Response.Warning }

func (o *reportScheduleListOutput) schedules() []*ReportSchedule {
	out := make([]*ReportSchedule, 0, len(o.Response.Schedules))
	for _, s := range o.Response.Schedules {
		out = append(out, &ReportSchedule{
			ID:            strings.TrimSpace(s.ID),
			Title:         s.Title,
			Active:        boolFromFlag(s.Active),
			TemplateID:    strings.TrimSpace(s.Report.TemplateID),
			TemplateTitle: s.Report.TemplateTitle,
			OutputFormat:  s.Report.OutputFormat,
		})
	}
	return out
}

// ReportScheduleFilter narrows a report schedule list request.
type ReportScheduleFilter struct {
	ID     string
	Active *bool
}

// ListReportSchedules returns scheduled report definitions matching
// filter, following truncation.
func (c *Client) ListReportSchedules(ctx context.Context, filter ReportScheduleFilter) ([]*ReportSchedule, error) {
	params := url.Values{}
	setIfNotEmpty(params, "id", filter.ID)
	if filter.Active != nil {
		params.Set("is_active", strconv.FormatBool(*filter.Active))
	}

	var all []*ReportSchedule
	err := c.listAll(ctx, request{
		capability: capScheduleReport,
		path:       "schedule/report/",
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(reportScheduleListOutput) },
		func(p paginated) error {
			all = append(all, p.(*reportScheduleListOutput).schedules()...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}

// LaunchReportScheduleNow triggers an immediate run of a scheduled report
// (Confirmed, doc 03 §10: `launch_now` action, `id` param). This is an
// imperative operation, not part of the resource lifecycle — it has no
// stable result to track — so it is exposed here for a caller that wants
// it (e.g. from a null_resource / local-exec trigger) rather than as its
// own resource.
func (c *Client) LaunchReportScheduleNow(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("id", id)
	return c.do(ctx, request{
		capability:    capScheduleReport,
		path:          "schedule/report/",
		action:        "launch_now",
		params:        params,
		nonIdempotent: true,
	}, nil)
}
