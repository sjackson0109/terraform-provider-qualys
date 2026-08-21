package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sjackson0109/terraform-provider-qualys/qps"
)

func dataSourceAssetTags() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys asset tags, for referencing tags managed outside " +
			"Terraform when targeting scans, schedules, reports or web applications.",

		ReadContext: dataSourceAssetTagsRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Return tags whose name exactly matches this value.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"name_contains": {
				Description: "Return tags whose name contains this substring.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"parent_tag_id": {
				Description: "Return tags whose parent is this tag.",
				Type:        schema.TypeString,
				Optional:    true,
			},

			"tags": {
				Description: "Matching tags, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":            {Type: schema.TypeString, Computed: true},
						"name":          {Type: schema.TypeString, Computed: true},
						"parent_tag_id": {Type: schema.TypeString, Computed: true},
						"rule_type":     {Type: schema.TypeString, Computed: true},
						"rule_text":     {Type: schema.TypeString, Computed: true},
						"color":         {Type: schema.TypeString, Computed: true},
						"created":       {Type: schema.TypeString, Computed: true},
						"modified":      {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceAssetTagsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
	if v := d.Get("parent_tag_id").(string); v != "" {
		criteria = append(criteria, qps.Criterion{Field: "parent", Operator: "EQUALS", Value: v})
	}

	var filters *qps.Filters
	if len(criteria) > 0 {
		filters = &qps.Filters{Criteria: criteria}
	}

	tags, err := c.SearchTags(ctx, filters)
	if err != nil {
		return diag.FromErr(err)
	}

	// Stable ordering: the API does not guarantee it, and an unstable order shows
	// up as a spurious diff wherever this feeds another resource.
	sort.Slice(tags, func(i, j int) bool { return tags[i].ID < tags[j].ID })

	out := make([]interface{}, 0, len(tags))
	ids := make([]string, 0, len(tags))
	for _, t := range tags {
		ids = append(ids, t.ID)
		out = append(out, map[string]interface{}{
			"id":            t.ID,
			"name":          t.Name,
			"parent_tag_id": t.Parent,
			"rule_type":     t.RuleType,
			"rule_text":     t.RuleText,
			"color":         t.Color,
			"created":       t.Created,
			"modified":      t.Modified,
		})
	}

	if err := d.Set("tags", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
