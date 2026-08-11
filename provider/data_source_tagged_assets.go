package provider

import (
	"context"
	"sort"

	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceTaggedAssets() *schema.Resource {
	return &schema.Resource{
		Description: "Look up AssetView/CSAM host assets by tag (doc 03 §14, Confirmed: " +
			"`search/am/hostasset` with Criteria `tagName`/`tagId` EQUALS). Useful for " +
			"referencing the current membership of a dynamic tag from Terraform, or " +
			"confirming a `qualys_host_tag_assignment` took effect.",

		ReadContext: dataSourceTaggedAssetsRead,

		Schema: map[string]*schema.Schema{
			"tag_id": {
				Description:  "Return only hosts tagged with this tag ID.",
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"tag_id", "tag_name"},
			},
			"tag_name": {
				Description: "Return only hosts tagged with this tag name.",
				Type:        schema.TypeString,
				Optional:    true,
			},

			"host_assets": {
				Description: "Matching host assets, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":              {Type: schema.TypeString, Computed: true},
						"name":            {Type: schema.TypeString, Computed: true},
						"dns_host_name":   {Type: schema.TypeString, Computed: true},
						"netbios_name":    {Type: schema.TypeString, Computed: true},
						"tracking_method": {Type: schema.TypeString, Computed: true},
						"os":              {Type: schema.TypeString, Computed: true},
						"created":         {Type: schema.TypeString, Computed: true},
						"updated":         {Type: schema.TypeString, Computed: true},
						"tag_ids": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func dataSourceTaggedAssetsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	var criteria []qps.Criterion
	if v := d.Get("tag_id").(string); v != "" {
		criteria = append(criteria, qps.Criterion{Field: "tagId", Operator: "EQUALS", Value: v})
	}
	if v := d.Get("tag_name").(string); v != "" {
		criteria = append(criteria, qps.Criterion{Field: "tagName", Operator: "EQUALS", Value: v})
	}

	assets, err := c.SearchHostAssets(ctx, &qps.Filters{Criteria: criteria})
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(assets, func(i, j int) bool { return assets[i].ID < assets[j].ID })

	out := make([]interface{}, 0, len(assets))
	ids := make([]string, 0, len(assets))
	for _, a := range assets {
		ids = append(ids, a.ID)
		tagIDs := make([]string, 0, len(a.Tags))
		for _, t := range a.Tags {
			tagIDs = append(tagIDs, t.ID)
		}
		out = append(out, map[string]interface{}{
			"id":              a.ID,
			"name":            a.Name,
			"dns_host_name":   a.DNSHostName,
			"netbios_name":    a.NetbiosName,
			"tracking_method": a.TrackingMethod,
			"os":              a.OS,
			"created":         a.Created,
			"updated":         a.Updated,
			"tag_ids":         tagIDs,
		})
	}

	if err := d.Set("host_assets", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
