package provider

import (
	"context"
	"sort"

	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHostDetections() *schema.Resource {
	return &schema.Resource{
		Description: "Look up registered hosts together with their last VM scan time, the " +
			"input to a stale-asset review. This is read-only: there is no create/update/" +
			"delete API for host detections, and purging stale hosts is a separate, " +
			"deliberately imperative operation this provider does not perform.",

		ReadContext: dataSourceHostDetectionsRead,

		Schema: map[string]*schema.Schema{
			"ips": {
				Description: "Restrict the lookup to these IPs, hyphenated ranges or CIDR blocks.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"ids": {
				Description: "Restrict the lookup to these host IDs.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"vm_scan_date_before": {
				Description: "Return hosts last scanned before this date " +
					"(`YYYY-MM-DD` or `YYYY-MM-DDTHH:MM:SSZ`) — hosts stale as of that date.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"vm_scan_date_after": {
				Description: "Return hosts last scanned after this date " +
					"(`YYYY-MM-DD` or `YYYY-MM-DDTHH:MM:SSZ`).",
				Type:     schema.TypeString,
				Optional: true,
			},

			"hosts": {
				Description: "Matching hosts, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":                 {Type: schema.TypeString, Computed: true},
						"ip":                 {Type: schema.TypeString, Computed: true},
						"tracking_method":    {Type: schema.TypeString, Computed: true},
						"dns":                {Type: schema.TypeString, Computed: true},
						"netbios":            {Type: schema.TypeString, Computed: true},
						"os":                 {Type: schema.TypeString, Computed: true},
						"last_scan_datetime": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceHostDetectionsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	hosts, err := c.ListHostDetections(ctx, vmdr.HostDetectionFilter{
		IPs:              stringSet(d, "ips"),
		IDs:              stringSet(d, "ids"),
		VMScanDateBefore: d.Get("vm_scan_date_before").(string),
		VMScanDateAfter:  d.Get("vm_scan_date_after").(string),
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
			"id":                 h.ID,
			"ip":                 h.IP,
			"tracking_method":    h.TrackingMethod,
			"dns":                h.DNS,
			"netbios":            h.NetBIOS,
			"os":                 h.OS,
			"last_scan_datetime": h.LastScanDatetime,
		})
	}

	if err := d.Set("hosts", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
