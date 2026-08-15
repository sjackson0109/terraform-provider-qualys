package provider

import (
	"context"
	"sort"

	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceReports() *schema.Resource {
	return &schema.Resource{
		Description: "Look up generated VM/PA reports — for example to poll a launched " +
			"report's status. Read-only: launching, cancelling, fetching and deleting a " +
			"report are imperative operations this provider deliberately does not model as " +
			"resources (see the README).\n\n" +
			"Evidence tier: Corroborated, not Confirmed, except `expiration_datetime` which " +
			"is Confirmed — see `vmdr.Report`'s doc comment. Verify other fields against " +
			"your tenant.",

		ReadContext: dataSourceReportsRead,

		Schema: map[string]*schema.Schema{
			"id": {Type: schema.TypeString, Optional: true},
			"state": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "One of Running, Finished, Submitted, Canceled, Errors.",
			},

			"reports": {
				Description: "Matching reports, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":                  {Type: schema.TypeString, Computed: true},
						"title":               {Type: schema.TypeString, Computed: true},
						"type":                {Type: schema.TypeString, Computed: true},
						"user_login":          {Type: schema.TypeString, Computed: true},
						"launch_datetime":     {Type: schema.TypeString, Computed: true},
						"output_format":       {Type: schema.TypeString, Computed: true},
						"size":                {Type: schema.TypeString, Computed: true},
						"status":              {Type: schema.TypeString, Computed: true},
						"expiration_datetime": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceReportsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	reports, err := c.ListReports(ctx, vmdr.ReportFilter{
		ID:    d.Get("id").(string),
		State: d.Get("state").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(reports, func(i, j int) bool { return reports[i].ID < reports[j].ID })

	out := make([]interface{}, 0, len(reports))
	ids := make([]string, 0, len(reports))
	for _, r := range reports {
		ids = append(ids, r.ID)
		out = append(out, map[string]interface{}{
			"id":                  r.ID,
			"title":               r.Title,
			"type":                r.Type,
			"user_login":          r.UserLogin,
			"launch_datetime":     r.LaunchDatetime,
			"output_format":       r.OutputFormat,
			"size":                r.Size,
			"status":              r.Status,
			"expiration_datetime": r.ExpirationDatetime,
		})
	}

	if err := d.Set("reports", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
