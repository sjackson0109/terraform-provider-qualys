package provider

import (
	"context"
	"errors"

	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVault() *schema.Resource {
	return &schema.Resource{
		Description: "A Qualys credential vault. Referencing a vault from an authentication " +
			"record's `vault_id` keeps the secret out of Terraform configuration and state " +
			"entirely.\n\n" +
			"Only `title` and `type` are Confirmed request fields (doc 03 §11); every vault " +
			"type's own connection parameters (endpoint URL, safe name, auth method, and so " +
			"on) are passed through `parameters` verbatim, using whatever field names your " +
			"vault type's own Qualys documentation specifies — this provider does not assert " +
			"per-vault-type field names it has no confirmed evidence for. Verify against your " +
			"tenant before relying on this in production.",

		CreateContext: resourceVaultCreate,
		ReadContext:   resourceVaultRead,
		UpdateContext: resourceVaultUpdate,
		DeleteContext: resourceVaultDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"title": {
				Description: "Name of the vault.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"type": {
				Description: "Vault type, for example `CyberArk AIM`, `HashiCorp` or " +
					"`Azure Key Vault`. Qualys does not document a way to change this after " +
					"creation, so changing it forces replacement.",
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"parameters": {
				Description: "Vault-type-specific connection parameters, sent verbatim as " +
					"additional fields on create/update. Field names are not validated by " +
					"this provider — use the names your vault type's Qualys documentation " +
					"specifies. Values may include secrets (an API token, for example); mark " +
					"them sensitive at the call site if so.",
				Type:      schema.TypeMap,
				Optional:  true,
				Elem:      &schema.Schema{Type: schema.TypeString},
				Sensitive: true,
			},
		},
	}
}

func vaultInputFrom(d *schema.ResourceData) vmdr.VaultInput {
	params := map[string]string{}
	for k, v := range d.Get("parameters").(map[string]interface{}) {
		if s, ok := v.(string); ok {
			params[k] = s
		}
	}
	return vmdr.VaultInput{
		Title:      d.Get("title").(string),
		Type:       d.Get("type").(string),
		Parameters: params,
	}
}

func resourceVaultCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	id, err := c.CreateVault(ctx, vaultInputFrom(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)
	return resourceVaultRead(ctx, d, meta)
}

func resourceVaultRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	v, err := c.GetVault(ctx, d.Id())
	if err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	// parameters is deliberately not refreshed here: ListVaults (which
	// GetVault is built on) returns only id/title/type — the per-type
	// connection parameters are not part of the Confirmed list response
	// shape, so there is nothing to read them back from. Drift in a
	// vault's connection parameters is therefore not detected, the same
	// documented trade-off as authentication record passwords.
	return diag.FromErr(setAll(d, map[string]interface{}{
		"title": v.Title,
		"type":  v.Type,
	}))
}

func resourceVaultUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateVault(ctx, d.Id(), vaultInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceVaultRead(ctx, d, meta)
}

func resourceVaultDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteVault(ctx, d.Id()); err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "Vault deleted; authentication records referencing it lose their credential",
		Detail: "Any authentication record whose vault_id pointed at this vault can no " +
			"longer authenticate. Scans using it will fall back to unauthenticated scanning.",
	}}
}
