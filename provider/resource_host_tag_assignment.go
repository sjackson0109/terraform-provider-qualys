package provider

import (
	"context"
	"errors"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceHostTagAssignment() *schema.Resource {
	return &schema.Resource{
		Description: "Manages which asset tags are assigned to an AssetView/CSAM host asset " +
			"(doc 03 §14, Confirmed: `POST /qps/rest/2.0/update/am/hostasset` with " +
			"`tags.add`/`tags.remove`). Tag-based scan and report targeting depends on this " +
			"association.\n\n" +
			"One resource manages the full set of `tag_ids` on one `host_asset_id` " +
			"authoritatively: on every apply it reconciles by adding tags present in " +
			"configuration but not currently assigned, and removing tags assigned but no " +
			"longer in configuration (computed from a search read on every plan, unlike " +
			"`qualys_excluded_ips` — the AM search response shape for host assets is " +
			"corroborated well enough to decode; see `qps.HostAsset`'s doc comment for the " +
			"evidence tier).",

		CreateContext: resourceHostTagAssignmentCreate,
		ReadContext:   resourceHostTagAssignmentRead,
		UpdateContext: resourceHostTagAssignmentUpdate,
		DeleteContext: resourceHostTagAssignmentDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"host_asset_id": {
				Description: "AssetView/CSAM host asset ID (`qwebHostId`) to assign tags to.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"tag_ids": {
				Description: "Asset tag IDs assigned to this host asset. Managed " +
					"authoritatively: removing an ID here unassigns that tag.",
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceHostTagAssignmentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	hostAssetID := d.Get("host_asset_id").(string)
	tagIDs := stringSet(d, "tag_ids")

	if len(tagIDs) > 0 {
		if err := c.UpdateHostAssetTags(ctx, hostAssetID, tagIDs, nil); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(hostAssetID)
	return resourceHostTagAssignmentRead(ctx, d, meta)
}

func resourceHostTagAssignmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	assets, err := c.SearchHostAssets(ctx, &qps.Filters{Criteria: []qps.Criterion{
		{Field: "id", Operator: "EQUALS", Value: d.Id()},
	}})
	if err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	if len(assets) == 0 {
		d.SetId("")
		return nil
	}

	tagIDs := make([]string, 0, len(assets[0].Tags))
	for _, t := range assets[0].Tags {
		tagIDs = append(tagIDs, t.ID)
	}

	return diag.FromErr(setAll(d, map[string]interface{}{
		"host_asset_id": assets[0].ID,
		"tag_ids":       tagIDs,
	}))
}

func resourceHostTagAssignmentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	oldRaw, newRaw := d.GetChange("tag_ids")
	oldIDs := stringSetFromInterface(oldRaw)
	newIDs := stringSetFromInterface(newRaw)

	newSet := make(map[string]bool, len(newIDs))
	for _, id := range newIDs {
		newSet[id] = true
	}
	oldSet := make(map[string]bool, len(oldIDs))
	for _, id := range oldIDs {
		oldSet[id] = true
	}

	var add, remove []string
	for _, id := range newIDs {
		if !oldSet[id] {
			add = append(add, id)
		}
	}
	for _, id := range oldIDs {
		if !newSet[id] {
			remove = append(remove, id)
		}
	}

	if err := c.UpdateHostAssetTags(ctx, d.Id(), add, remove); err != nil {
		return diag.FromErr(err)
	}
	return resourceHostTagAssignmentRead(ctx, d, meta)
}

func resourceHostTagAssignmentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	tagIDs := stringSet(d, "tag_ids")
	if len(tagIDs) == 0 {
		return nil
	}
	if err := c.UpdateHostAssetTags(ctx, d.Id(), nil, tagIDs); err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
