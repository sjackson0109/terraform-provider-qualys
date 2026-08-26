package provider

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceWASAuthRecords() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys WAS authentication records, for associating a web " +
			"application (`qualys_was_application`'s `auth_record_ids`) or a scan schedule " +
			"(`qualys_was_scan_schedule`'s `web_app_auth_record_id`) with a record managed " +
			"outside Terraform, by name instead of a hardcoded ID. Credential contents are " +
			"never exposed here — Qualys masks them, the same way the resource's own Read does.",

		ReadContext: dataSourceWASAuthRecordsRead,

		Schema: map[string]*schema.Schema{
			"name_contains": {
				Description: "Filter results to records whose name contains this substring " +
					"(case-insensitive). Applied client-side, after the API call.",
				Type:     schema.TypeString,
				Optional: true,
			},

			"auth_records": {
				Description: "Matching authentication records, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":       {Type: schema.TypeString, Computed: true},
						"name":     {Type: schema.TypeString, Computed: true},
						"comments": {Type: schema.TypeString, Computed: true},
						"tag_ids": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"created": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceWASAuthRecordsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	records, err := c.SearchWASAuthRecords(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if needle := strings.ToLower(d.Get("name_contains").(string)); needle != "" {
		filtered := records[:0]
		for _, r := range records {
			if strings.Contains(strings.ToLower(r.Name), needle) {
				filtered = append(filtered, r)
			}
		}
		records = filtered
	}

	sort.Slice(records, func(i, j int) bool { return numericLess(records[i].ID, records[j].ID) })

	out := make([]interface{}, 0, len(records))
	ids := make([]string, 0, len(records))
	for _, r := range records {
		ids = append(ids, r.ID)
		tagIDs := make([]string, 0, len(r.Tags))
		for _, t := range r.Tags {
			tagIDs = append(tagIDs, t.ID)
		}
		out = append(out, map[string]interface{}{
			"id":       r.ID,
			"name":     r.Name,
			"comments": r.Comments,
			"tag_ids":  tagIDs,
			"created":  r.Created,
		})
	}

	if err := d.Set("auth_records", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
