package provider

import (
	"context"
	"sort"

	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceWASReportTemplates() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys WAS report templates, for referencing a template ID from " +
			"`qualys_was_report_schedule`. Read-only: a user-supplied walkthrough confirms " +
			"the API supports count/search/get but not create/update/delete — templates are " +
			"managed in the Qualys UI. Distinct from `data.qualys_report_templates`, which " +
			"covers the separate, legacy VM report template API.",

		ReadContext: dataSourceWASReportTemplatesRead,

		Schema: map[string]*schema.Schema{
			"name_contains": {
				Description: "Return templates whose name contains this substring.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"type": {
				Description: "Restrict to this template type: one of `WAS_WEBAPP_REPORT`, " +
					"`WAS_SCAN_REPORT`, `WAS_SCORECARD_REPORT`, `WAS_CATALOG_REPORT`.",
				Type:     schema.TypeString,
				Optional: true,
			},

			"templates": {
				Description: "Matching templates, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":   {Type: schema.TypeString, Computed: true},
						"name": {Type: schema.TypeString, Computed: true},
						"type": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceWASReportTemplatesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	var criteria []qps.Criterion
	if v := d.Get("name_contains").(string); v != "" {
		criteria = append(criteria, qps.Criterion{Field: "name", Operator: "CONTAINS", Value: v})
	}
	if v := d.Get("type").(string); v != "" {
		criteria = append(criteria, qps.Criterion{Field: "type", Operator: "EQUALS", Value: v})
	}

	var filters *qps.Filters
	if len(criteria) > 0 {
		filters = &qps.Filters{Criteria: criteria}
	}

	templates, err := c.SearchWASReportTemplates(ctx, filters)
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(templates, func(i, j int) bool { return templates[i].ID < templates[j].ID })

	out := make([]interface{}, 0, len(templates))
	ids := make([]string, 0, len(templates))
	for _, t := range templates {
		ids = append(ids, t.ID)
		out = append(out, map[string]interface{}{
			"id":   t.ID,
			"name": t.Name,
			"type": t.Type,
		})
	}

	if err := d.Set("templates", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
