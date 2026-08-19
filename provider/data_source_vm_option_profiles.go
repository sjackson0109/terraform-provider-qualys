package provider

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVMOptionProfiles() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys VM scan option profiles, for referencing profiles " +
			"managed outside Terraform from scan schedules.",

		ReadContext: dataSourceVMOptionProfilesRead,

		Schema: map[string]*schema.Schema{
			"title_contains": {
				Description: "Filter results to profiles whose title contains this substring " +
					"(case-insensitive). Applied client-side, after the API call — the list API " +
					"takes no server-side title filter.",
				Type:     schema.TypeString,
				Optional: true,
			},

			"option_profiles": {
				Description: "Matching option profiles, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":      {Type: schema.TypeString, Computed: true},
						"title":   {Type: schema.TypeString, Computed: true},
						"owner":   {Type: schema.TypeString, Computed: true},
						"global":  {Type: schema.TypeBool, Computed: true},
						"default": {Type: schema.TypeBool, Computed: true},
						"offline": {Type: schema.TypeBool, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceVMOptionProfilesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	profiles, err := c.ListVMOptionProfiles(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if needle := strings.ToLower(d.Get("title_contains").(string)); needle != "" {
		filtered := profiles[:0]
		for _, p := range profiles {
			if strings.Contains(strings.ToLower(p.Title), needle) {
				filtered = append(filtered, p)
			}
		}
		profiles = filtered
	}

	sort.Slice(profiles, func(i, j int) bool { return profiles[i].ID < profiles[j].ID })

	out := make([]interface{}, 0, len(profiles))
	ids := make([]string, 0, len(profiles))
	for _, p := range profiles {
		ids = append(ids, p.ID)
		out = append(out, map[string]interface{}{
			"id":      p.ID,
			"title":   p.Title,
			"owner":   p.Owner,
			"global":  p.Global,
			"default": p.Default,
			"offline": p.Offline,
		})
	}

	if err := d.Set("option_profiles", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
