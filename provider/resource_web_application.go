package provider

import (
	"context"
	"errors"

	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceWebApplication() *schema.Resource {
	fileElem := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {Type: schema.TypeString, Required: true},
			"content_base64": {
				Description: "Base64-encoded file content, e.g. `filebase64(\"openapi.yml\")`.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
		},
	}

	return &schema.Resource{
		Description: "A Qualys WAS web application: the scan target that option profiles, " +
			"auth records and scan schedules reference.\n\n" +
			"**Provenance:** `attributes`, `config` (`cancel_scans_at`/`cancel_scans_after_hours`/" +
			"`default_dns_override_id`), `dns_override_ids`, `swagger_file`, `postman_collection`, " +
			"`malware_monitoring`/`malware_notification` and `crawling_scripts` were all added on " +
			"the strength of a user-supplied \"Gap Review\" document quoting the official Qualys " +
			"WebApp reference. `malware_scheduling` recurrence detail is deliberately NOT modelled " +
			"— the same document says its exact structure should come from a dedicated example, " +
			"not be inferred, so only the flags that gate it are exposed here.\n\n" +
			"**A genuine API landmine, reproduced faithfully rather than hidden:** `config` " +
			"(covering `cancel_scans_at`/`cancel_scans_after_hours`/`default_dns_override_id`) is " +
			"only sent when at least one of those three is set in configuration. If you set " +
			"`default_dns_override_id` alone, `config` is sent without a cancellation element, " +
			"and Qualys clears any existing default-cancellation setting as a side effect — even " +
			"one configured outside Terraform. Set `cancel_scans_at` or `cancel_scans_after_hours` " +
			"explicitly if you need to preserve it.",

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
					"added/removed IDs. The response shape used to read this back is corroborated, " +
					"not fully confirmed — verify against a tenant.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"attributes": {
				Description: "Custom key/value attributes. Confirmed against the official Qualys " +
					"WebApp reference. Sent as a full authoritative replacement on every apply.",
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"cancel_scans_at": {
				Description: "Absolute RFC3339 time to cancel scans against this web " +
					"application, e.g. `2026-08-10T23:00:00Z`. Conflicts with " +
					"`cancel_scans_after_hours`. See the resource-level note on the `config` " +
					"landmine before using this together with `default_dns_override_id` alone.",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"cancel_scans_after_hours"},
			},
			"cancel_scans_after_hours": {
				Description: "Cancel scans against this web application this many hours after " +
					"they start. Conflicts with `cancel_scans_at`.",
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"cancel_scans_at"},
			},
			"default_dns_override_id": {
				Description: "ID of the `qualys_was_dns_override` (must also appear in " +
					"`dns_override_ids`) that crawls/scans resolve through by default. " +
					"Confirmed as `config.defaultDnsOverride.id`.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"dns_override_ids": {
				Description: "IDs of `qualys_was_dns_override`s associated with this web " +
					"application. Confirmed as `dnsOverrides`, sent as a full authoritative " +
					"replacement (the \"set\" idiom) on every apply — unlike `auth_record_ids`, " +
					"which uses incremental add/remove.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"swagger_file": {
				Description: "A Swagger/OpenAPI 2.0 or 3.0 definition (YAML or XML, max 5MB) " +
					"describing this web application's API surface. Confirmed against the " +
					"official Qualys reference. Conflicts with `postman_collection`.",
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"postman_collection"},
				Elem:          fileElem,
			},
			"postman_collection": {
				Description: "A Postman collection (v2.0.0/v2.1.0) describing this web " +
					"application's API surface, with optional environment/global variable " +
					"files. Confirmed against the official Qualys reference. Conflicts with " +
					"`swagger_file`.",
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"swagger_file"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"collection":           {Type: schema.TypeList, Required: true, MaxItems: 1, Elem: fileElem},
						"environment_variable": {Type: schema.TypeList, Optional: true, MaxItems: 1, Elem: fileElem},
						"global_variable":      {Type: schema.TypeList, Optional: true, MaxItems: 1, Elem: fileElem},
					},
				},
			},

			"malware_monitoring": {
				Description: "Enable malware monitoring for this web application. Confirmed " +
					"against the official Qualys reference. `malware_scheduling` recurrence is " +
					"not modelled — see the resource-level note.",
				Type:     schema.TypeBool,
				Optional: true,
			},
			"malware_notification": {
				Description: "Email notification when malware monitoring detects an issue.",
				Type:        schema.TypeBool,
				Optional:    true,
			},

			"crawling_scripts": {
				Description: "Selenium crawling scripts attached to this web application. " +
					"Read-only: this provider does not support creating or updating these, " +
					"only reading them back. Confirmed against the official Qualys reference.",
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":                      {Type: schema.TypeString, Computed: true},
						"name":                    {Type: schema.TypeString, Computed: true},
						"data":                    {Type: schema.TypeString, Computed: true},
						"requires_authentication": {Type: schema.TypeBool, Computed: true},
						"starting_url":            {Type: schema.TypeString, Computed: true},
						"starting_url_regex":      {Type: schema.TypeBool, Computed: true},
					},
				},
			},

			"created": {Type: schema.TypeString, Computed: true},
		},
	}
}

func webAppFileFrom(raw []interface{}) *qps.WebAppFile {
	if len(raw) != 1 {
		return nil
	}
	m, ok := raw[0].(map[string]interface{})
	if !ok {
		return nil
	}
	return &qps.WebAppFile{Name: m["name"].(string), Content: m["content_base64"].(string)}
}

func webAppInputFrom(d *schema.ResourceData) qps.WebAppInput {
	in := qps.WebAppInput{
		Name:                   d.Get("name").(string),
		URL:                    d.Get("url").(string),
		TagIDs:                 stringSet(d, "tag_ids"),
		DNSOverrideIDs:         stringSet(d, "dns_override_ids"),
		CancelScansAt:          d.Get("cancel_scans_at").(string),
		CancelScansAfterNHours: d.Get("cancel_scans_after_hours").(int),
		DefaultDNSOverrideID:   d.Get("default_dns_override_id").(string),
		MalwareMonitoring:      d.Get("malware_monitoring").(bool),
		MalwareNotification:    d.Get("malware_notification").(bool),
	}

	if raw, ok := d.Get("attributes").(map[string]interface{}); ok && len(raw) > 0 {
		attrs := make([]qps.WebAppAttribute, 0, len(raw))
		for k, v := range raw {
			attrs = append(attrs, qps.WebAppAttribute{Name: k, Value: v.(string)})
		}
		in.Attributes = attrs
	}

	if raw := d.Get("swagger_file").([]interface{}); len(raw) == 1 {
		in.SwaggerFile = webAppFileFrom(raw)
	} else if raw := d.Get("postman_collection").([]interface{}); len(raw) == 1 {
		m := raw[0].(map[string]interface{})
		in.PostmanCollection = &qps.WebAppPostmanCollection{
			Collection:          webAppFileFrom(m["collection"].([]interface{})),
			EnvironmentVariable: webAppFileFrom(m["environment_variable"].([]interface{})),
			GlobalVariable:      webAppFileFrom(m["global_variable"].([]interface{})),
		}
	}

	return in
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

	attributes := make(map[string]interface{}, len(app.Attributes))
	for _, a := range app.Attributes {
		attributes[a.Name] = a.Value
	}

	crawlingScripts := make([]interface{}, 0, len(app.CrawlingScripts))
	for _, s := range app.CrawlingScripts {
		crawlingScripts = append(crawlingScripts, map[string]interface{}{
			"id":                      s.ID,
			"name":                    s.Name,
			"data":                    s.Data,
			"requires_authentication": s.RequiresAuthentication,
			"starting_url":            s.StartingURL,
			"starting_url_regex":      s.StartingURLRegex,
		})
	}

	// swagger_file/postman_collection are deliberately not set here: like
	// credentials elsewhere in this provider, there is no confirmed read
	// shape to refresh them from. Leaving them out keeps whatever the last
	// apply configured.
	return diag.FromErr(setAll(d, map[string]interface{}{
		"name":                     app.Name,
		"url":                      app.URL,
		"tag_ids":                  tagIDs,
		"auth_record_ids":          app.AuthRecordIDs,
		"attributes":               attributes,
		"cancel_scans_at":          app.CancelScansAt,
		"cancel_scans_after_hours": app.CancelScansAfterNHours,
		"default_dns_override_id":  app.DefaultDNSOverrideID,
		"dns_override_ids":         app.DNSOverrideIDs,
		"malware_monitoring":       app.MalwareMonitoring,
		"malware_notification":     app.MalwareNotification,
		"crawling_scripts":         crawlingScripts,
		"created":                  app.Created,
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
