package provider

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVMScanSchedules() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys scan schedules, for auditing schedules managed outside " +
			"Terraform or referencing them from other configuration.",

		ReadContext: dataSourceVMScanSchedulesRead,

		Schema: map[string]*schema.Schema{
			"id_filter": {
				Description: "Restrict the lookup to a single schedule ID.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"title_contains": {
				Description: "Filter results to schedules whose title contains this substring " +
					"(case-insensitive). Applied client-side, after the API call.",
				Type:     schema.TypeString,
				Optional: true,
			},

			"scan_schedules": {
				Description: "Matching scan schedules, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":                   {Type: schema.TypeString, Computed: true},
						"title":                {Type: schema.TypeString, Computed: true},
						"active":               {Type: schema.TypeBool, Computed: true},
						"active_state":         {Type: schema.TypeInt, Computed: true},
						"option_profile_id":    {Type: schema.TypeString, Computed: true},
						"option_profile_title": {Type: schema.TypeString, Computed: true},
						"occurrence":           {Type: schema.TypeString, Computed: true},
						"start_date":           {Type: schema.TypeString, Computed: true},
						"time_zone_code":       {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

// dataSourceVMScanSchedulesRead uses "id_filter" rather than "id" for the input
// attribute: a computed "id" is also present (the data source's own synthetic
// ID), and TypeString "id" is reserved by the SDK for that purpose.
func dataSourceVMScanSchedulesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	schedules, err := c.ListScanSchedules(ctx, d.Get("id_filter").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	if needle := strings.ToLower(d.Get("title_contains").(string)); needle != "" {
		filtered := schedules[:0]
		for _, s := range schedules {
			if strings.Contains(strings.ToLower(s.Title), needle) {
				filtered = append(filtered, s)
			}
		}
		schedules = filtered
	}

	sort.Slice(schedules, func(i, j int) bool { return schedules[i].ID < schedules[j].ID })

	out := make([]interface{}, 0, len(schedules))
	ids := make([]string, 0, len(schedules))
	for _, s := range schedules {
		ids = append(ids, s.ID)
		out = append(out, map[string]interface{}{
			"id":                   s.ID,
			"title":                s.Title,
			"active":               s.Active.Enabled(),
			"active_state":         int(s.Active),
			"option_profile_id":    s.OptionProfileID,
			"option_profile_title": s.OptionProfileTitle,
			"occurrence":           s.Occurrence,
			"start_date":           s.StartDate,
			"time_zone_code":       s.TimeZoneCode,
		})
	}

	if err := d.Set("scan_schedules", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
