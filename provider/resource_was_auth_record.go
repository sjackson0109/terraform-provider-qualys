package provider

import (
	"context"
	"errors"

	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceWASAuthRecord() *schema.Resource {
	fieldElem := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The login form's field name (as it appears in the page's HTML).",
				Type:        schema.TypeString,
				Required:    true,
			},
			"value": {
				Description: "Value to fill in. Write-only — see the resource-level note on credentials.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			"secured": {
				Description: "Whether this field holds a credential (for example a password). " +
					"Qualys masks secured field values in the UI and API responses.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}

	return &schema.Resource{
		Description: "A Qualys WAS authentication record: form or server credentials the " +
			"crawler uses to authenticate against a web application during a scan.\n\n" +
			"**Provenance note:** the field names used here were first corroborated from " +
			"two independent non-official sources, then substantially revised against two " +
			"user-supplied excerpts of the official WAS API User Guide (Chapter 3): a " +
			"primary-source excerpt (p.102) confirmed the endpoint path (`/was/webauthrecord`) " +
			"and the record sub-type vocabulary, and a transcribed \"Create and Update " +
			"Authentication Records\" walkthrough showed `username`/`password`/`login_url` as " +
			"flat elements rather than the generic field list this resource originally sent " +
			"them through — the current shape. Treat the walkthrough as strong but not " +
			"absolute evidence (it reads as a transcription, not a verbatim quote) and verify " +
			"against a tenant before relying on this in production. Selenium script and " +
			"OAuth2 records are not implemented.\n\n" +
			"Associating a record with a web application is a separate step — see " +
			"`qualys_web_application`'s `auth_record_ids`.\n\n" +
			"Credentials are write-only: Qualys masks them on read, so this provider never " +
			"reads them back into state. A credential changed outside Terraform is not " +
			"detected as drift, and the values still live in Terraform state because " +
			"Terraform stores what you configured — protect state accordingly.",

		CreateContext: resourceWASAuthRecordCreate,
		ReadContext:   resourceWASAuthRecordRead,
		UpdateContext: resourceWASAuthRecordUpdate,
		DeleteContext: resourceWASAuthRecordDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Authentication record name.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"comments": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"tag_ids": {
				Description: "Asset tag IDs (see `qualys_asset_tag`) associated with this record.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			"form_record": {
				Description: "Form-based authentication: the crawler fills these in on a login " +
					"page. Conflicts with `server_record`. Set `login_url`/`username`/`password` " +
					"for a `STANDARD` record (a login page with a known username and password " +
					"field); set one or more `field` blocks instead for a `CUSTOM` record whose " +
					"field names aren't fixed.",
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"server_record"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sub_type": {
							Description: "Authentication style: `STANDARD` or `CUSTOM` " +
								"(corroborated by a create-example and by supported-types " +
								"documentation respectively), or `CERT`/`SELFINITIAL` " +
								"(corroborated only via a filter enum). Not validated " +
								"client-side; an invalid value is rejected by the API at " +
								"apply time.",
							Type:     schema.TypeString,
							Optional: true,
						},
						"login_url": {
							Description: "URL of the login page. Used with a `STANDARD` record.",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"username": {
							Description: "Used with a `STANDARD` record. Write-only — see the " +
								"resource-level note on credentials.",
							Type:     schema.TypeString,
							Optional: true,
						},
						"password": {
							Description: "Used with a `STANDARD` record. Write-only — see the " +
								"resource-level note on credentials.",
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
						"ssl_only": {
							Description: "Only submit credentials over HTTPS.",
							Type:        schema.TypeBool,
							Optional:    true,
						},
						"auth_vault": {
							Description: "Store this record in the Qualys password vault.",
							Type:        schema.TypeBool,
							Optional:    true,
						},
						"field": {
							Description: "One login-form field, for a `CUSTOM` record whose " +
								"field names aren't fixed. Repeat for each field the login " +
								"form needs.",
							Type:     schema.TypeList,
							Optional: true,
							Elem:     fieldElem,
						},
					},
				},
			},

			"server_record": {
				Description: "Server-based authentication (HTTP Basic/Digest-style). " +
					"Conflicts with `form_record`.",
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"form_record"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sub_type": {
							Description: "Authentication style: `BASIC` or `DIGEST`, " +
								"corroborated via a filter enum. Not validated client-side; " +
								"an invalid value is rejected by the API at apply time.",
							Type:     schema.TypeString,
							Optional: true,
						},
						"username": {
							Type:     schema.TypeString,
							Required: true,
						},
						"password": {
							Description: "Write-only — see the resource-level note on credentials.",
							Type:        schema.TypeString,
							Required:    true,
							Sensitive:   true,
						},
						"domain": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"created": {Type: schema.TypeString, Computed: true},
		},

		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			form := d.Get("form_record").([]interface{})
			server := d.Get("server_record").([]interface{})
			if len(form) == 0 && len(server) == 0 {
				return errors.New("one of form_record or server_record is required: " +
					"a record with neither has no credential for the crawler to use")
			}
			if len(form) == 1 {
				m := form[0].(map[string]interface{})
				hasStandard := m["username"].(string) != "" && m["password"].(string) != ""
				hasCustom := len(m["field"].([]interface{})) > 0
				if !hasStandard && !hasCustom {
					return errors.New("form_record needs either username and password " +
						"(a STANDARD record) or at least one field block (a CUSTOM record)")
				}
			}
			return nil
		},
	}
}

func wasAuthFieldsFrom(raw []interface{}) []qps.WASAuthField {
	out := make([]qps.WASAuthField, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		out = append(out, qps.WASAuthField{
			Name:    m["name"].(string),
			Value:   m["value"].(string),
			Secured: m["secured"].(bool),
		})
	}
	return out
}

func wasAuthRecordInputFrom(d *schema.ResourceData) qps.WASAuthRecordInput {
	in := qps.WASAuthRecordInput{
		Name:     d.Get("name").(string),
		Comments: d.Get("comments").(string),
		TagIDs:   stringSet(d, "tag_ids"),
	}

	if raw := d.Get("form_record").([]interface{}); len(raw) == 1 {
		m := raw[0].(map[string]interface{})
		in.Form = &qps.WASFormRecord{
			SubType:   m["sub_type"].(string),
			LoginURL:  m["login_url"].(string),
			Username:  m["username"].(string),
			Password:  m["password"].(string),
			SSLOnly:   m["ssl_only"].(bool),
			AuthVault: m["auth_vault"].(bool),
			Fields:    wasAuthFieldsFrom(m["field"].([]interface{})),
		}
	}

	if raw := d.Get("server_record").([]interface{}); len(raw) == 1 {
		m := raw[0].(map[string]interface{})
		in.Server = &qps.WASServerRecord{
			SubType:  m["sub_type"].(string),
			Username: m["username"].(string),
			Password: m["password"].(string),
			Domain:   m["domain"].(string),
		}
	}

	return in
}

func resourceWASAuthRecordCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	rec, err := c.CreateWASAuthRecord(ctx, wasAuthRecordInputFrom(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(rec.ID)
	return resourceWASAuthRecordRead(ctx, d, meta)
}

func resourceWASAuthRecordRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	rec, err := c.GetWASAuthRecord(ctx, d.Id())
	if err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			d.SetId("")
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "WAS authentication record no longer visible; removing it from state",
				Detail: "Qualys reported OBJECT_NOT_FOUND, which it also returns for objects " +
					"outside the caller's scope. If the record still exists and the credentials' " +
					"scope changed, restore the scope before applying, or Terraform will create a " +
					"duplicate.",
			}}
		}
		return diag.FromErr(err)
	}

	tagIDs := make([]string, 0, len(rec.Tags))
	for _, t := range rec.Tags {
		tagIDs = append(tagIDs, t.ID)
	}

	// form_record, server_record and their credential contents are
	// deliberately not set here: Qualys masks them on read, so there is
	// nothing trustworthy to refresh state with. Leaving them out of this
	// map keeps whatever the last apply configured.
	return diag.FromErr(setAll(d, map[string]interface{}{
		"name":     rec.Name,
		"comments": rec.Comments,
		"tag_ids":  tagIDs,
		"created":  rec.Created,
	}))
}

func resourceWASAuthRecordUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateWASAuthRecord(ctx, d.Id(), wasAuthRecordInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceWASAuthRecordRead(ctx, d, meta)
}

func resourceWASAuthRecordDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteWASAuthRecord(ctx, d.Id()); err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "WAS authentication record deleted; scans referencing it lose credentialed access",
		Detail: "Scan schedules and option profiles that referenced this record will now crawl " +
			"unauthenticated. They will still run, but find considerably less.",
	}}
}
