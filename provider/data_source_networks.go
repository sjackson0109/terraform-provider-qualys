package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNetworks() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys networks, so configuration can reference a network by " +
			"name rather than hardcoding its ID.",

		ReadContext: dataSourceNetworksRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Return only the network with this exact name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"networks": {
				Description: "Matching networks, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":            {Type: schema.TypeString, Computed: true},
						"name":          {Type: schema.TypeString, Computed: true},
						"appliance_ids": {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}},
					},
				},
			},
		},
	}
}

func dataSourceNetworksRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	nets, err := c.ListNetworks(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if want := d.Get("name").(string); want != "" {
		filtered := nets[:0]
		for _, n := range nets {
			if n.Name == want {
				filtered = append(filtered, n)
			}
		}
		nets = filtered
	}

	sort.Slice(nets, func(i, j int) bool { return nets[i].ID < nets[j].ID })

	out := make([]interface{}, 0, len(nets))
	ids := make([]string, 0, len(nets))
	for _, n := range nets {
		ids = append(ids, n.ID)
		out = append(out, map[string]interface{}{
			"id":            n.ID,
			"name":          n.Name,
			"appliance_ids": n.ApplianceIDs,
		})
	}

	if err := d.Set("networks", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
