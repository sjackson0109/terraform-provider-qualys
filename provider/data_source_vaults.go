package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVaults() *schema.Resource {
	return &schema.Resource{
		Description: "Look up credential vaults configured for the subscription.\n\n" +
			"Referencing a vault from an authentication record keeps the secret out of " +
			"Terraform configuration and state entirely, which is preferable to supplying " +
			"a password.",

		ReadContext: dataSourceVaultsRead,

		Schema: map[string]*schema.Schema{
			"title": {
				Description: "Return only the vault with this exact title.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"vaults": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":    {Type: schema.TypeString, Computed: true},
						"title": {Type: schema.TypeString, Computed: true},
						"type":  {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceVaultsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	vaults, err := c.ListVaults(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if want := d.Get("title").(string); want != "" {
		filtered := vaults[:0]
		for _, v := range vaults {
			if v.Title == want {
				filtered = append(filtered, v)
			}
		}
		vaults = filtered
	}

	sort.Slice(vaults, func(i, j int) bool { return vaults[i].ID < vaults[j].ID })

	out := make([]interface{}, 0, len(vaults))
	ids := make([]string, 0, len(vaults))
	for _, v := range vaults {
		ids = append(ids, v.ID)
		out = append(out, map[string]interface{}{
			"id": v.ID, "title": v.Title, "type": v.Type,
		})
	}

	if err := d.Set("vaults", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
