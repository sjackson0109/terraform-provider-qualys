package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
)

func resourceExcludedIPs() *schema.Resource {
	return &schema.Resource{
		Description: "Excludes IPs from Qualys scanning (doc 03 §1, Confirmed: " +
			"`/api/2.0/fo/asset/excluded_ip/` list/add/remove/remove_all).\n\n" +
			"This resource is **write-only with respect to drift**: the excluded-IP list " +
			"endpoint's response shape (element names) is not documented anywhere this " +
			"session could evidence, only the request parameters and the fact that a list " +
			"action exists — so, rather than guess response field names, this resource does " +
			"not attempt to read back what Qualys currently has excluded. It reconciles by " +
			"re-issuing add/remove for whatever changes between plans, idempotently, but a " +
			"change made outside Terraform (someone re-including an IP via the UI, for " +
			"example) is not detected until the next apply overwrites it back to the " +
			"configured set.",

		CreateContext: resourceExcludedIPsCreate,
		ReadContext:   resourceExcludedIPsRead,
		UpdateContext: resourceExcludedIPsUpdate,
		DeleteContext: resourceExcludedIPsDelete,

		Schema: map[string]*schema.Schema{
			"ips": ipSetSchema("IPs, hyphenated ranges or CIDR blocks to exclude from "+
				"scanning. Managed authoritatively: removing an entry re-includes it in scans.", true),
			"comment": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"expiry_days": {
				Description: "Automatically re-include the IPs in scanning after this many " +
					"days. Zero (the default) leaves the exclusion open-ended.",
				Type:     schema.TypeInt,
				Optional: true,
			},
			"asset_group_names": {
				Description: "Scope the exclusion to these asset groups.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"network_id": {
				Description: "Network to scope the exclusion to, for subscriptions with " +
					"Network Support enabled. Changing it forces replacement, since there is " +
					"no confirmed way to move an existing exclusion between networks.",
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceExcludedIPsCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	in := vmdr.AddExcludedIPsInput{
		IPs:             stringSet(d, "ips"),
		Comment:         d.Get("comment").(string),
		ExpiryDays:      d.Get("expiry_days").(int),
		AssetGroupNames: stringSet(d, "asset_group_names"),
		NetworkID:       d.Get("network_id").(string),
	}
	if err := c.AddExcludedIPs(ctx, in); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(digestID(in.IPs))
	return nil
}

// resourceExcludedIPsRead is deliberately a no-op: see the resource's
// Description for why the remote list response is not decoded. Terraform
// state is treated as authoritative between applies.
func resourceExcludedIPsRead(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return nil
}

func resourceExcludedIPsUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	networkID := d.Get("network_id").(string)
	comment := d.Get("comment").(string)

	if d.HasChange("ips") {
		oldRaw, newRaw := d.GetChange("ips")
		oldIPs := stringSetFromInterface(oldRaw)
		newIPs := stringSetFromInterface(newRaw)

		newSet := make(map[string]bool, len(newIPs))
		for _, ip := range newIPs {
			newSet[ip] = true
		}
		var removed []string
		for _, ip := range oldIPs {
			if !newSet[ip] {
				removed = append(removed, ip)
			}
		}
		if len(removed) > 0 {
			if err := c.RemoveExcludedIPs(ctx, removed, comment, networkID); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	// Re-add the full current set unconditionally, not just newly-added
	// IPs: comment/expiry_days/asset_group_names may have changed too, and
	// `add` is the only way this endpoint accepts those fields.
	in := vmdr.AddExcludedIPsInput{
		IPs:             stringSet(d, "ips"),
		Comment:         comment,
		ExpiryDays:      d.Get("expiry_days").(int),
		AssetGroupNames: stringSet(d, "asset_group_names"),
		NetworkID:       networkID,
	}
	if err := c.AddExcludedIPs(ctx, in); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(digestID(in.IPs))
	return nil
}

func resourceExcludedIPsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	ips := stringSet(d, "ips")
	if len(ips) == 0 {
		return nil
	}
	if err := c.RemoveExcludedIPs(ctx, ips, d.Get("comment").(string), d.Get("network_id").(string)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
