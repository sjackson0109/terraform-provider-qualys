package provider

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceWASDNSOverrides() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys WAS DNS override records, for referencing a record " +
			"managed outside Terraform from `qualys_was_scan_schedule`'s `dns_override_id` " +
			"or `qualys_was_application`'s `dns_override_ids`/`default_dns_override_id`, by " +
			"name instead of a hardcoded ID.",

		ReadContext: dataSourceWASDNSOverridesRead,

		Schema: map[string]*schema.Schema{
			"name_contains": {
				Description: "Filter results to records whose name contains this substring " +
					"(case-insensitive). Applied client-side, after the API call.",
				Type:     schema.TypeString,
				Optional: true,
			},

			"dns_overrides": {
				Description: "Matching DNS override records, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":   {Type: schema.TypeString, Computed: true},
						"name": {Type: schema.TypeString, Computed: true},
						"mapping": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host_name":  {Type: schema.TypeString, Computed: true},
									"ip_address": {Type: schema.TypeString, Computed: true},
								},
							},
						},
						"comments": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"tag_ids": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"created": {Type: schema.TypeString, Computed: true},
						"updated": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceWASDNSOverridesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	overrides, err := c.SearchWASDNSOverrides(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if needle := strings.ToLower(d.Get("name_contains").(string)); needle != "" {
		filtered := overrides[:0]
		for _, o := range overrides {
			if strings.Contains(strings.ToLower(o.Name), needle) {
				filtered = append(filtered, o)
			}
		}
		overrides = filtered
	}

	sort.Slice(overrides, func(i, j int) bool { return numericLess(overrides[i].ID, overrides[j].ID) })

	out := make([]interface{}, 0, len(overrides))
	ids := make([]string, 0, len(overrides))
	for _, o := range overrides {
		ids = append(ids, o.ID)

		mappings := make([]interface{}, 0, len(o.Mappings))
		for _, m := range o.Mappings {
			mappings = append(mappings, map[string]interface{}{
				"host_name":  m.HostName,
				"ip_address": m.IPAddress,
			})
		}
		tagIDs := make([]string, 0, len(o.Tags))
		for _, t := range o.Tags {
			tagIDs = append(tagIDs, t.ID)
		}

		out = append(out, map[string]interface{}{
			"id":       o.ID,
			"name":     o.Name,
			"mapping":  mappings,
			"comments": o.Comments,
			"tag_ids":  tagIDs,
			"created":  o.Created,
			"updated":  o.Updated,
		})
	}

	if err := d.Set("dns_overrides", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
