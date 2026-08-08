package provider

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceReportTemplates() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys report templates, for referencing a template ID when " +
			"launching a report. This is read-only: report templates have no create/update/" +
			"delete API of their own for most types (only scan, PCI scan and patch templates " +
			"do, each with its own dedicated endpoint), and this list is the only surface " +
			"that enumerates templates of every type together.",

		ReadContext: dataSourceReportTemplatesRead,

		Schema: map[string]*schema.Schema{
			"title_contains": {
				Description: "Filter results to templates whose title contains this substring " +
					"(case-insensitive). Applied client-side — the underlying list API takes no " +
					"filter parameters at all.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"template_type": {
				Description: "Filter results to this template type, for example `Scan` or " +
					"`Policy`. Applied client-side.",
				Type:     schema.TypeString,
				Optional: true,
			},

			"report_templates": {
				Description: "Matching report templates, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":            {Type: schema.TypeString, Computed: true},
						"type":          {Type: schema.TypeString, Computed: true},
						"template_type": {Type: schema.TypeString, Computed: true},
						"title":         {Type: schema.TypeString, Computed: true},
						"owner_login":   {Type: schema.TypeString, Computed: true},
						"owner_name":    {Type: schema.TypeString, Computed: true},
						"last_update":   {Type: schema.TypeString, Computed: true},
						"global":        {Type: schema.TypeBool, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceReportTemplatesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	templates, err := c.ListReportTemplates(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if needle := strings.ToLower(d.Get("title_contains").(string)); needle != "" {
		filtered := templates[:0]
		for _, t := range templates {
			if strings.Contains(strings.ToLower(t.Title), needle) {
				filtered = append(filtered, t)
			}
		}
		templates = filtered
	}
	if want := d.Get("template_type").(string); want != "" {
		filtered := templates[:0]
		for _, t := range templates {
			if strings.EqualFold(t.TemplateType, want) {
				filtered = append(filtered, t)
			}
		}
		templates = filtered
	}

	sort.Slice(templates, func(i, j int) bool { return templates[i].ID < templates[j].ID })

	out := make([]interface{}, 0, len(templates))
	ids := make([]string, 0, len(templates))
	for _, t := range templates {
		ids = append(ids, t.ID)
		out = append(out, map[string]interface{}{
			"id":            t.ID,
			"type":          t.Type,
			"template_type": t.TemplateType,
			"title":         t.Title,
			"owner_login":   t.OwnerLogin,
			"owner_name":    t.OwnerName,
			"last_update":   t.LastUpdate,
			"global":        t.Global,
		})
	}

	if err := d.Set("report_templates", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
