package provider

import (
	"context"
	"sort"

	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceReportSchedules() *schema.Resource {
	return &schema.Resource{
		Description: "Look up scheduled report definitions. Read-only, and deliberately " +
			"has no counterpart `qualys_report_schedule` resource: doc 03 §10 confirms " +
			"there is no create/update/delete/enable/disable API for report schedules at " +
			"all — Qualys's own documentation tree covers only `list` and `launch_now`, and " +
			"schedule management is GUI-only. This is a Confirmed absence, not an " +
			"unresearched gap.\n\n" +
			"Evidence tier for the fields below: Corroborated, not Confirmed — see " +
			"`vmdr.ReportSchedule`'s doc comment.",

		ReadContext: dataSourceReportSchedulesRead,

		Schema: map[string]*schema.Schema{
			"id":     {Type: schema.TypeString, Optional: true},
			"active": {Type: schema.TypeBool, Optional: true},

			"report_schedules": {
				Description: "Matching report schedules, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":             {Type: schema.TypeString, Computed: true},
						"title":          {Type: schema.TypeString, Computed: true},
						"active":         {Type: schema.TypeBool, Computed: true},
						"template_id":    {Type: schema.TypeString, Computed: true},
						"template_title": {Type: schema.TypeString, Computed: true},
						"output_format":  {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceReportSchedulesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	filter := vmdr.ReportScheduleFilter{ID: d.Get("id").(string)}
	if v, ok := d.GetOkExists("active"); ok {
		b := v.(bool)
		filter.Active = &b
	}

	schedules, err := c.ListReportSchedules(ctx, filter)
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(schedules, func(i, j int) bool { return schedules[i].ID < schedules[j].ID })

	out := make([]interface{}, 0, len(schedules))
	ids := make([]string, 0, len(schedules))
	for _, s := range schedules {
		ids = append(ids, s.ID)
		out = append(out, map[string]interface{}{
			"id":             s.ID,
			"title":          s.Title,
			"active":         s.Active,
			"template_id":    s.TemplateID,
			"template_title": s.TemplateTitle,
			"output_format":  s.OutputFormat,
		})
	}

	if err := d.Set("report_schedules", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
