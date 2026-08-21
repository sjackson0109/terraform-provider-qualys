package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sjackson0109/terraform-provider-qualys/qps"
)

func resourceWASDNSOverride() *schema.Resource {
	return &schema.Resource{
		Description: "A Qualys WAS DNS override record: resolves one or more hostnames to " +
			"user-defined IP addresses during crawling and scanning — for scanning " +
			"dev/UAT environments, private applications with no public DNS, or " +
			"redirecting a hostname to an alternative back-end. Reference the ID from " +
			"`qualys_was_scan_schedule`'s `dns_override_id` to use it on a schedule.\n\n" +
			"**Provenance:** Confirmed against a user-supplied walkthrough carrying " +
			"inline citations to specific docs.qualys.com pages — stronger provenance " +
			"than a plain transcription. It also corrected the endpoint path (`was/" +
			"dnsoverride`, not `was/dnsoverriderecord`, this discovery's earlier guess) " +
			"and surfaced a genuine API quirk: update and delete take no ID in the URL " +
			"path (unlike every other WAS resource this provider models) — the ID " +
			"travels in the request body for update, and as a filter criterion for " +
			"delete. That quirk is entirely internal to this resource's implementation " +
			"and does not affect how you use it in configuration.",

		CreateContext: resourceWASDNSOverrideCreate,
		ReadContext:   resourceWASDNSOverrideRead,
		UpdateContext: resourceWASDNSOverrideUpdate,
		DeleteContext: resourceWASDNSOverrideDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "DNS override record name.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"mapping": {
				Description: "One hostname/IP override. Repeat for each hostname to " +
					"redirect. Sent as a full authoritative replacement on every apply.",
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host_name": {
							Description: "Hostname or fully qualified domain name to override.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"ip_address": {
							Description: "IPv4 address to resolve `host_name` to.",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
			"comments": {
				Description: "Administrative notes. Sent as a full authoritative " +
					"replacement on every apply.",
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tag_ids": {
				Description: "Asset tag IDs (see `qualys_asset_tag`) associated with this record.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			"created": {Type: schema.TypeString, Computed: true},
			"updated": {Type: schema.TypeString, Computed: true},
		},
	}
}

func wasDNSMappingsFrom(raw []interface{}) []qps.WASDNSMapping {
	out := make([]qps.WASDNSMapping, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		out = append(out, qps.WASDNSMapping{
			HostName:  m["host_name"].(string),
			IPAddress: m["ip_address"].(string),
		})
	}
	return out
}

func wasDNSOverrideInputFrom(d *schema.ResourceData) qps.WASDNSOverrideInput {
	return qps.WASDNSOverrideInput{
		Name:     d.Get("name").(string),
		Mappings: wasDNSMappingsFrom(d.Get("mapping").([]interface{})),
		Comments: stringList(d, "comments"),
		TagIDs:   stringSet(d, "tag_ids"),
	}
}

func resourceWASDNSOverrideCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	override, err := c.CreateWASDNSOverride(ctx, wasDNSOverrideInputFrom(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(override.ID)
	return resourceWASDNSOverrideRead(ctx, d, meta)
}

func resourceWASDNSOverrideRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	override, err := c.GetWASDNSOverride(ctx, d.Id())
	if err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			return qpsNotFoundOnRead(d, "WAS DNS override")
		}
		return diag.FromErr(err)
	}

	mappings := make([]interface{}, 0, len(override.Mappings))
	for _, m := range override.Mappings {
		mappings = append(mappings, map[string]interface{}{
			"host_name":  m.HostName,
			"ip_address": m.IPAddress,
		})
	}
	tagIDs := make([]string, 0, len(override.Tags))
	for _, t := range override.Tags {
		tagIDs = append(tagIDs, t.ID)
	}

	return diag.FromErr(setAll(d, map[string]interface{}{
		"name":     override.Name,
		"mapping":  mappings,
		"comments": override.Comments,
		"tag_ids":  tagIDs,
		"created":  override.Created,
		"updated":  override.Updated,
	}))
}

func resourceWASDNSOverrideUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateWASDNSOverride(ctx, d.Id(), wasDNSOverrideInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceWASDNSOverrideRead(ctx, d, meta)
}

func resourceWASDNSOverrideDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteWASDNSOverride(ctx, d.Id()); err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
