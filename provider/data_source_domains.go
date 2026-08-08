package provider

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDomains() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys asset domains and their netblocks, for referencing " +
			"domains managed outside Terraform in asset groups. This is read-only: the add/" +
			"update/delete request parameters for this API are not confirmed against official " +
			"documentation, so this provider does not offer a qualys_domain resource.",

		ReadContext: dataSourceDomainsRead,

		Schema: map[string]*schema.Schema{
			"name_contains": {
				Description: "Filter results to domains whose name contains this substring " +
					"(case-insensitive). Applied client-side — the underlying list API takes no " +
					"confirmed filter parameters.",
				Type:     schema.TypeString,
				Optional: true,
			},

			"domains": {
				Description: "Matching domains, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":         {Type: schema.TypeString, Computed: true},
						"name":       {Type: schema.TypeString, Computed: true},
						"network_id": {Type: schema.TypeString, Computed: true},
						"netblocks":  {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}},
					},
				},
			},
		},
	}
}

func dataSourceDomainsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	domains, err := c.ListDomains(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if needle := strings.ToLower(d.Get("name_contains").(string)); needle != "" {
		filtered := domains[:0]
		for _, dom := range domains {
			if strings.Contains(strings.ToLower(dom.Name), needle) {
				filtered = append(filtered, dom)
			}
		}
		domains = filtered
	}

	sort.Slice(domains, func(i, j int) bool { return domains[i].ID < domains[j].ID })

	out := make([]interface{}, 0, len(domains))
	ids := make([]string, 0, len(domains))
	for _, dom := range domains {
		ids = append(ids, dom.ID)
		out = append(out, map[string]interface{}{
			"id":         dom.ID,
			"name":       dom.Name,
			"network_id": dom.NetworkID,
			"netblocks":  dom.Netblocks,
		})
	}

	if err := d.Set("domains", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
