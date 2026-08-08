package provider

import (
	"context"
	"sort"

	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHostAssets() *schema.Resource {
	return &schema.Resource{
		Description: "Look up registered host assets.\n\n" +
			"`no_vm_scan_since` is how stale assets are identified: it returns hosts that " +
			"have not been scanned since the given date, which is the input to a review " +
			"before any purge.",

		ReadContext: dataSourceHostAssetsRead,

		Schema: map[string]*schema.Schema{
			"ips": {
				Description: "Restrict the lookup to these addresses, ranges or CIDR blocks.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"network_id": {Type: schema.TypeString, Optional: true},
			"no_vm_scan_since": {
				Description: "Return hosts not scanned since this date (`YYYY-MM-DD`).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"vm_scan_since": {
				Description: "Return hosts scanned since this date (`YYYY-MM-DD`).",
				Type:        schema.TypeString,
				Optional:    true,
			},

			"hosts": {
				Description: "Matching hosts, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":              {Type: schema.TypeString, Computed: true},
						"ip":              {Type: schema.TypeString, Computed: true},
						"tracking_method": {Type: schema.TypeString, Computed: true},
						"dns":             {Type: schema.TypeString, Computed: true},
						"netbios":         {Type: schema.TypeString, Computed: true},
						"os":              {Type: schema.TypeString, Computed: true},
						"owner":           {Type: schema.TypeString, Computed: true},
						"comments":        {Type: schema.TypeString, Computed: true},
						"last_vm_scan":    {Type: schema.TypeString, Computed: true},
						"network_id":      {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceHostAssetsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	hosts, err := c.ListHosts(ctx, vmdr.HostFilter{
		IPs:           stringSet(d, "ips"),
		NetworkID:     d.Get("network_id").(string),
		NoVMScanSince: d.Get("no_vm_scan_since").(string),
		VMScanSince:   d.Get("vm_scan_since").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(hosts, func(i, j int) bool { return hosts[i].ID < hosts[j].ID })

	out := make([]interface{}, 0, len(hosts))
	ids := make([]string, 0, len(hosts))
	for _, h := range hosts {
		ids = append(ids, h.ID)
		out = append(out, map[string]interface{}{
			"id":              h.ID,
			"ip":              h.IP,
			"tracking_method": h.TrackingMethod,
			"dns":             h.DNS,
			"netbios":         h.NetBIOS,
			"os":              h.OS,
			"owner":           h.Owner,
			"comments":        h.Comments,
			"last_vm_scan":    h.LastVMScan,
			"network_id":      h.NetworkID,
		})
	}

	if err := d.Set("hosts", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
