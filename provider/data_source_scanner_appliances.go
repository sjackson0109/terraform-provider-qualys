package provider

import (
	"context"
	"sort"

	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceScannerAppliances() *schema.Resource {
	return &schema.Resource{
		Description: "Look up scanner appliances, including physical ones that cannot be " +
			"managed through the API.\n\n" +
			"`up_to_date` compares the appliance's signature and module versions against " +
			"the latest Qualys has published, which is a better readiness signal than " +
			"`status` alone.",

		ReadContext: dataSourceScannerAppliancesRead,

		Schema: map[string]*schema.Schema{
			"name":       {Type: schema.TypeString, Optional: true},
			"network_id": {Type: schema.TypeString, Optional: true},
			"type": {
				Description: "Restrict to `physical`, `virtual` or `offline` appliances.",
				Type:        schema.TypeString,
				Optional:    true,
			},

			"appliances": {
				Description: "Matching appliances, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":                 {Type: schema.TypeString, Computed: true},
						"uuid":               {Type: schema.TypeString, Computed: true},
						"name":               {Type: schema.TypeString, Computed: true},
						"status":             {Type: schema.TypeString, Computed: true},
						"type":               {Type: schema.TypeString, Computed: true},
						"network_id":         {Type: schema.TypeString, Computed: true},
						"software_version":   {Type: schema.TypeString, Computed: true},
						"vulnsigs_version":   {Type: schema.TypeString, Computed: true},
						"heartbeats_missed":  {Type: schema.TypeString, Computed: true},
						"up_to_date":         {Type: schema.TypeBool, Computed: true},
						"running_scan_count": {Type: schema.TypeInt, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceScannerAppliancesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	appliances, err := c.ListAppliances(ctx, vmdr.ApplianceFilter{
		Name:      d.Get("name").(string),
		NetworkID: d.Get("network_id").(string),
		Type:      d.Get("type").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(appliances, func(i, j int) bool { return appliances[i].ID < appliances[j].ID })

	out := make([]interface{}, 0, len(appliances))
	ids := make([]string, 0, len(appliances))
	for _, a := range appliances {
		ids = append(ids, a.ID)
		out = append(out, map[string]interface{}{
			"id":                 a.ID,
			"uuid":               a.UUID,
			"name":               a.Name,
			"status":             a.Status,
			"type":               a.Type,
			"network_id":         a.NetworkID,
			"software_version":   a.Version,
			"vulnsigs_version":   a.VulnSigsVersion,
			"heartbeats_missed":  a.HeartbeatsMissed,
			"up_to_date":         a.UpToDate(),
			"running_scan_count": a.RunningScanCount,
		})
	}

	if err := d.Set("appliances", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
