package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
)

func dataSourceScans() *schema.Resource {
	return &schema.Resource{
		Description: "Look up VM scan jobs — for example to check a running scan's status, " +
			"or find recent scans of a target. Read-only: launching, cancelling, pausing, " +
			"resuming and deleting a scan are imperative operations this provider " +
			"deliberately does not model as resources (see the README).\n\n" +
			"Evidence tier: Corroborated, not Confirmed — see `vmdr.Scan`'s doc comment. " +
			"Verify field values against your tenant.",

		ReadContext: dataSourceScansRead,

		Schema: map[string]*schema.Schema{
			"state":           {Type: schema.TypeString, Optional: true},
			"type":            {Type: schema.TypeString, Optional: true},
			"target":          {Type: schema.TypeString, Optional: true},
			"user_login":      {Type: schema.TypeString, Optional: true},
			"launched_after":  {Type: schema.TypeString, Optional: true, Description: "RFC3339 datetime."},
			"launched_before": {Type: schema.TypeString, Optional: true, Description: "RFC3339 datetime."},

			"scans": {
				Description: "Matching scans, ordered by reference for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ref":             {Type: schema.TypeString, Computed: true},
						"title":           {Type: schema.TypeString, Computed: true},
						"type":            {Type: schema.TypeString, Computed: true},
						"user_login":      {Type: schema.TypeString, Computed: true},
						"launch_datetime": {Type: schema.TypeString, Computed: true},
						"duration":        {Type: schema.TypeString, Computed: true},
						"target":          {Type: schema.TypeString, Computed: true},
						"processed":       {Type: schema.TypeBool, Computed: true},
						"status":          {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceScansRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	scans, err := c.ListScans(ctx, vmdr.ScanFilter{
		State:          d.Get("state").(string),
		Type:           d.Get("type").(string),
		Target:         d.Get("target").(string),
		UserLogin:      d.Get("user_login").(string),
		LaunchedAfter:  d.Get("launched_after").(string),
		LaunchedBefore: d.Get("launched_before").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(scans, func(i, j int) bool { return scans[i].Ref < scans[j].Ref })

	out := make([]interface{}, 0, len(scans))
	refs := make([]string, 0, len(scans))
	for _, s := range scans {
		refs = append(refs, s.Ref)
		out = append(out, map[string]interface{}{
			"ref":             s.Ref,
			"title":           s.Title,
			"type":            s.Type,
			"user_login":      s.UserLogin,
			"launch_datetime": s.LaunchDatetime,
			"duration":        s.Duration,
			"target":          s.Target,
			"processed":       s.Processed,
			"status":          s.Status,
		})
	}

	if err := d.Set("scans", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(refs))
	return nil
}
