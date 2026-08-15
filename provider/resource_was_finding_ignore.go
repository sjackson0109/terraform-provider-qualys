package provider

import (
	"context"
	"errors"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceWASFindingIgnore() *schema.Resource {
	return &schema.Resource{
		Description: "Declares a Qualys WAS finding ignored — a genuine accepted risk or " +
			"confirmed false positive, with `comment` as the audit-trail justification. " +
			"Deleting this resource reopens the finding.\n\n" +
			"**Only `ignore`/`reopen` are modelled as a resource.** A user-supplied " +
			"\"Findings Lifecycle Actions\" walkthrough confirms `fix` too " +
			"(`POST fix/was/finding/<id>`), but the same source recommends against " +
			"using lifecycle actions as a substitute for rescanning — \"fixed\" is meant " +
			"to be scan-verified (a later scan can revert it if the vulnerability is " +
			"still detected), not administratively declared, so it doesn't fit this " +
			"provider's declarative resource model the way `ignore` legitimately does.\n\n" +
			"Findings are discovered by scans, not created by Terraform: find the " +
			"`finding_id` via `data.qualys_was_findings`, then manage its ignored state " +
			"here. If Qualys reports the finding as no longer ignored (for example a " +
			"rescan rediscovered it), this resource is removed from state so the next " +
			"apply re-ignores it.",

		CreateContext: resourceWASFindingIgnoreCreate,
		ReadContext:   resourceWASFindingIgnoreRead,
		UpdateContext: resourceWASFindingIgnoreUpdate,
		DeleteContext: resourceWASFindingIgnoreDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"finding_id": {
				Description: "ID of the WAS finding to ignore (see `data.qualys_was_findings`).",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"comment": {
				Description: "Justification for ignoring this finding, e.g. \"Accepted " +
					"business risk\" or \"False positive, verified by security team\". " +
					"Sent again on every change, so editing it updates the recorded reason.",
				Type:     schema.TypeString,
				Required: true,
			},

			"qid":      {Type: schema.TypeString, Computed: true},
			"name":     {Type: schema.TypeString, Computed: true},
			"url":      {Type: schema.TypeString, Computed: true},
			"severity": {Type: schema.TypeInt, Computed: true},
			"status":   {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceWASFindingIgnoreCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	findingID := d.Get("finding_id").(string)
	if err := c.IgnoreWASFinding(ctx, findingID, d.Get("comment").(string)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(findingID)
	return resourceWASFindingIgnoreRead(ctx, d, meta)
}

func resourceWASFindingIgnoreRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	finding, err := c.GetWASFinding(ctx, d.Id())
	if err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			d.SetId("")
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "WAS finding no longer visible; removing it from state",
				Detail: "Qualys reported OBJECT_NOT_FOUND, which it also returns for objects " +
					"outside the caller's scope. If the finding still exists and the credentials' " +
					"scope changed, restore the scope before applying, or Terraform will attempt " +
					"to re-ignore it.",
			}}
		}
		return diag.FromErr(err)
	}

	if !finding.IsIgnored {
		// Reopened outside Terraform — most plausibly a rescan rediscovering the
		// issue. Removing from state lets the next apply re-ignore it rather
		// than silently leaving it active while state claims otherwise.
		d.SetId("")
		return diag.Diagnostics{{
			Severity: diag.Warning,
			Summary:  "WAS finding is no longer ignored; removing it from state",
			Detail: "Qualys reports this finding as not ignored — most likely a rescan " +
				"rediscovered it. Re-apply to ignore it again, or remove it from configuration " +
				"if the risk should no longer be accepted.",
		}}
	}

	return diag.FromErr(setAll(d, map[string]interface{}{
		"finding_id": finding.ID,
		"qid":        finding.QID,
		"name":       finding.Name,
		"url":        finding.URL,
		"severity":   finding.Severity,
		"status":     finding.Status,
	}))
}

func resourceWASFindingIgnoreUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.IgnoreWASFinding(ctx, d.Id(), d.Get("comment").(string)); err != nil {
		return diag.FromErr(err)
	}
	return resourceWASFindingIgnoreRead(ctx, d, meta)
}

func resourceWASFindingIgnoreDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.ReopenWASFinding(ctx, d.Id(), "Un-ignored via Terraform"); err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
