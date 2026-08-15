package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAssetGroups() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys asset groups, for referencing groups managed outside " +
			"Terraform in scans, schedules and reports.",

		ReadContext: dataSourceAssetGroupsRead,

		Schema: map[string]*schema.Schema{
			"ids": {
				Description: "Restrict the lookup to these asset group IDs.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"network_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"title_contains": {
				Description: "Filter results to groups whose title contains this substring " +
					"(case-insensitive). Applied client-side, after the API call.",
				Type:     schema.TypeString,
				Optional: true,
			},

			"asset_groups": {
				Description: "Matching asset groups, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":              {Type: schema.TypeString, Computed: true},
						"title":           {Type: schema.TypeString, Computed: true},
						"network_id":      {Type: schema.TypeString, Computed: true},
						"comments":        {Type: schema.TypeString, Computed: true},
						"ips":             {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}},
						"domains":         {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}},
						"dns_names":       {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}},
						"netbios_names":   {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}},
						"appliance_ids":   {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}},
						"business_impact": {Type: schema.TypeString, Computed: true},
						"owner_id":        {Type: schema.TypeString, Computed: true},
						"unit_id":         {Type: schema.TypeString, Computed: true},
						"last_update":     {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceAssetGroupsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	groups, err := c.ListAssetGroups(ctx, vmdr.AssetGroupFilter{
		IDs:        stringSet(d, "ids"),
		NetworkIDs: stringSet(d, "network_ids"),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	if needle := strings.ToLower(d.Get("title_contains").(string)); needle != "" {
		filtered := groups[:0]
		for _, g := range groups {
			if strings.Contains(strings.ToLower(g.Title), needle) {
				filtered = append(filtered, g)
			}
		}
		groups = filtered
	}

	// Sort by ID so the output is stable across reads; Qualys does not guarantee
	// list ordering, and an unstable order would show as a spurious diff wherever
	// the data source feeds another resource.
	sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })

	out := make([]interface{}, 0, len(groups))
	ids := make([]string, 0, len(groups))
	for _, g := range groups {
		ids = append(ids, g.ID)
		out = append(out, map[string]interface{}{
			"id":              g.ID,
			"title":           g.Title,
			"network_id":      g.NetworkID,
			"comments":        g.Comments,
			"ips":             g.IPs,
			"domains":         g.Domains,
			"dns_names":       g.DNSNames,
			"netbios_names":   g.NetBIOSNames,
			"appliance_ids":   g.ApplianceIDs,
			"business_impact": g.BusinessImpact,
			"owner_id":        g.OwnerID,
			"unit_id":         g.UnitID,
			"last_update":     g.LastUpdate,
		})
	}

	if err := d.Set("asset_groups", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}

// digestID derives a stable synthetic ID from the result set, so that the data
// source's ID changes only when the results actually change.
func digestID(parts []string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, ",")))
	return hex.EncodeToString(h[:16])
}
