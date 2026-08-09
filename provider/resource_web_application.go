package provider

import (
	"context"
	"errors"

	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceWebApplication() *schema.Resource {
	return &schema.Resource{
		Description: "A Qualys WAS web application: the scan target that option profiles, " +
			"auth records and scan schedules reference.",

		CreateContext: resourceWebApplicationCreate,
		ReadContext:   resourceWebApplicationRead,
		UpdateContext: resourceWebApplicationUpdate,
		DeleteContext: resourceWebApplicationDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Web application name.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"url": {
				Description: "The application's base URL.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"tag_ids": {
				Description: "Asset tag IDs (see `qualys_asset_tag`) associated with this web " +
					"application. Tags are the cross-module targeting primitive shared with VMDR " +
					"and AssetView; a WAS scan schedule can target web applications by tag.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"auth_record_ids": {
				Description: "IDs of `qualys_was_auth_record`s to associate with this web " +
					"application. A user-supplied example shows this managed via a separate " +
					"add/remove call rather than the authoritative \"set\" idiom `tag_ids` uses " +
					"— this provider diffs the configured set against state and sends only the " +
					"added/removed IDs. The response shape used to read this back was not shown " +
					"in that example; it is inferred by analogy with how `tag_ids` round-trips.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"created": {Type: schema.TypeString, Computed: true},
		},
	}
}

func webAppInputFrom(d *schema.ResourceData) qps.WebAppInput {
	return qps.WebAppInput{
		Name:   d.Get("name").(string),
		URL:    d.Get("url").(string),
		TagIDs: stringSet(d, "tag_ids"),
	}
}

// diffStringSets returns the elements added and removed going from old to
// new, for callers that manage a relationship via incremental add/remove
// rather than an authoritative replace.
func diffStringSets(old, new []string) (added, removed []string) {
	oldSet := make(map[string]bool, len(old))
	for _, v := range old {
		oldSet[v] = true
	}
	newSet := make(map[string]bool, len(new))
	for _, v := range new {
		newSet[v] = true
		if !oldSet[v] {
			added = append(added, v)
		}
	}
	for _, v := range old {
		if !newSet[v] {
			removed = append(removed, v)
		}
	}
	return added, removed
}

func resourceWebApplicationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	app, err := c.CreateWebApp(ctx, webAppInputFrom(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(app.ID)

	// A user-supplied example states that creating an authentication record
	// does not associate it with a web application, and vice versa: the
	// association is always this separate call, even on first creation.
	if authRecordIDs := stringSet(d, "auth_record_ids"); len(authRecordIDs) > 0 {
		if err := c.UpdateWebAppAuthRecordAssociations(ctx, app.ID, authRecordIDs, nil); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceWebApplicationRead(ctx, d, meta)
}

func resourceWebApplicationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	app, err := c.GetWebApp(ctx, d.Id())
	if err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			d.SetId("")
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "Web application no longer visible; removing it from state",
				Detail: "Qualys reported OBJECT_NOT_FOUND, which it also returns for objects " +
					"outside the caller's scope. If the web application still exists and the " +
					"credentials' scope changed, restore the scope before applying, or Terraform " +
					"will create a duplicate.",
			}}
		}
		return diag.FromErr(err)
	}

	tagIDs := make([]string, 0, len(app.Tags))
	for _, t := range app.Tags {
		tagIDs = append(tagIDs, t.ID)
	}

	return diag.FromErr(setAll(d, map[string]interface{}{
		"name":            app.Name,
		"url":             app.URL,
		"tag_ids":         tagIDs,
		"auth_record_ids": app.AuthRecordIDs,
		"created":         app.Created,
	}))
}

func resourceWebApplicationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateWebApp(ctx, d.Id(), webAppInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("auth_record_ids") {
		oldRaw, newRaw := d.GetChange("auth_record_ids")
		added, removed := diffStringSets(
			stringSetFromInterface(oldRaw), stringSetFromInterface(newRaw))
		if len(added) > 0 || len(removed) > 0 {
			if err := c.UpdateWebAppAuthRecordAssociations(ctx, d.Id(), added, removed); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return resourceWebApplicationRead(ctx, d, meta)
}

func resourceWebApplicationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteWebApp(ctx, d.Id()); err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
