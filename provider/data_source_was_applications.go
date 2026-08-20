package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sjackson0109/terraform-provider-qualys/qps"
)

func dataSourceWASApplications() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys WAS web applications, for referencing applications " +
			"managed outside Terraform in scan schedules.",

		ReadContext: dataSourceWASApplicationsRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Return web applications whose name exactly matches this value.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"name_contains": {
				Description: "Return web applications whose name contains this substring.",
				Type:        schema.TypeString,
				Optional:    true,
			},

			"web_applications": {
				Description: "Matching web applications, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":      {Type: schema.TypeString, Computed: true},
						"name":    {Type: schema.TypeString, Computed: true},
						"url":     {Type: schema.TypeString, Computed: true},
						"created": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceWASApplicationsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	var criteria []qps.Criterion
	if v := d.Get("name").(string); v != "" {
		criteria = append(criteria, qps.Criterion{Field: "name", Operator: "EQUALS", Value: v})
	}
	if v := d.Get("name_contains").(string); v != "" {
		criteria = append(criteria, qps.Criterion{Field: "name", Operator: "CONTAINS", Value: v})
	}

	var filters *qps.Filters
	if len(criteria) > 0 {
		filters = &qps.Filters{Criteria: criteria}
	}

	apps, err := c.SearchWebApps(ctx, filters)
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(apps, func(i, j int) bool { return apps[i].ID < apps[j].ID })

	out := make([]interface{}, 0, len(apps))
	ids := make([]string, 0, len(apps))
	for _, a := range apps {
		ids = append(ids, a.ID)
		out = append(out, map[string]interface{}{
			"id":      a.ID,
			"name":    a.Name,
			"url":     a.URL,
			"created": a.Created,
		})
	}

	if err := d.Set("web_applications", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
