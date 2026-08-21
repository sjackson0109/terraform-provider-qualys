package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjackson0109/terraform-provider-qualys/qps"
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

	seleniumScriptElem := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"data": {
				Description: "Selenium IDE script content (HTML/XML), as exported from Selenium IDE.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"regex": {
				Description: "Pattern Qualys evaluates against the resulting page to confirm the " +
					"script authenticated successfully.",
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}

	return &schema.Resource{
		Description: "A Qualys WAS authentication record: form, server or OAuth2 credentials the " +
			"crawler uses to authenticate against a web application during a scan.\n\n" +
			"**Provenance note:** this resource's shape went through two corrections as better " +
			"evidence arrived. A transcribed \"Create and Update Authentication Records\" " +
			"walkthrough first established the endpoint path (`/was/webauthrecord`) and record " +
			"sub-type vocabulary, and — incorrectly — modelled `STANDARD` form credentials as " +
			"flat `username`/`password` elements. A later \"Gap Review\" document, quoting the " +
			"official Qualys create example directly, corrected this: `STANDARD` uses the same " +
			"generic `fields`/`set`/`WebAppAuthFormRecordField` list as `CUSTOM`. That same " +
			"document confirmed `SELENIUM` form records and a dedicated `oauth2_record` object " +
			"(not modelled through the field list). Treat all of this as strong but not absolute " +
			"evidence and verify against a tenant before relying on it in production.\n\n" +
			"Associating a record with a web application is a separate step — see " +
			"`qualys_was_application`'s `auth_record_ids`.\n\n" +
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
					"page, drives a Selenium script, or both. Conflicts with `server_record` and " +
					"`oauth2_record`. Set `login_url`/`username`/`password` for a `STANDARD` " +
					"record (a login page with a known username and password field, sent through " +
					"the same field list as `CUSTOM` — see the resource-level provenance note); " +
					"set one or more `field` blocks instead for a `CUSTOM` record whose field " +
					"names aren't fixed; set `selenium_script` for a `SELENIUM` record.",
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"server_record", "oauth2_record"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sub_type": {
							Description: "Authentication style: `STANDARD`, `CUSTOM` or `SELENIUM` " +
								"(all Confirmed against the official Qualys reference), or " +
								"`CERT`/`SELFINITIAL` (corroborated only via a filter enum). Not " +
								"validated client-side; an invalid value is rejected by the API " +
								"at apply time.",
							Type:     schema.TypeString,
							Optional: true,
						},
						"login_url": {
							Description: "URL of the login page. Used with a `STANDARD` record.",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"username": {
							Description: "Used with a `STANDARD` record, or with a `SELENIUM` " +
								"record that sets `selenium_creds` to substitute the script's " +
								"`@@authusername@@` placeholder. Sent through the record's " +
								"generic field list, not as a flat element. Write-only — see the " +
								"resource-level note on credentials.",
							Type:     schema.TypeString,
							Optional: true,
						},
						"password": {
							Description: "Used with a `STANDARD` record, or with a `SELENIUM` " +
								"record that sets `selenium_creds` to substitute the script's " +
								"`@@authpassword@@` placeholder. Sent through the record's " +
								"generic field list, not as a flat element. Write-only — see the " +
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
							Description: "One additional login-form field, for a `CUSTOM` record " +
								"whose field names aren't `username`/`password`. Repeat for each " +
								"field the login form needs.",
							Type:     schema.TypeList,
							Optional: true,
							Elem:     fieldElem,
						},
						"selenium_script": {
							Description: "Selenium IDE script used to authenticate, for a " +
								"`SELENIUM` record. Confirmed against the official Qualys " +
								"reference.",
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem:     seleniumScriptElem,
						},
						"selenium_creds": {
							Description: "When the Selenium script contains " +
								"`@@authusername@@`/`@@authpassword@@` placeholders, set this so " +
								"`username`/`password` fill them in without editing the script.",
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"server_record": {
				Description: "Server-based authentication (HTTP Basic/Digest-style). " +
					"Conflicts with `form_record` and `oauth2_record`.",
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"form_record", "oauth2_record"},
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

			"oauth2_record": {
				Description: "OAuth2 authentication. Confirmed as a dedicated `oauth2Record` " +
					"object with flat, grant-specific fields — not modelled through the generic " +
					"CUSTOM field list. Conflicts with `form_record` and `server_record`. Which " +
					"fields the API requires depends on `grant_type`: `AUTH_CODE`/`IMPLICIT` need " +
					"`selenium_script` and `redirect_url`; `PASSWORD` needs `access_token_url`, " +
					"`username` and `password`; `CLIENT_CREDS` needs `access_token_url`. " +
					"`client_id`/`client_secret`/`scope` are optional wherever the grant accepts " +
					"them. Not validated client-side beyond requiring `grant_type` — the API " +
					"enforces the rest at apply time.",
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"form_record", "server_record"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"grant_type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								qps.WASOAuth2GrantNone, qps.WASOAuth2GrantAuthCode,
								qps.WASOAuth2GrantImplicit, qps.WASOAuth2GrantPassword,
								qps.WASOAuth2GrantClientCreds,
							}, false),
						},
						"access_token_url": {Type: schema.TypeString, Optional: true},
						"client_id":        {Type: schema.TypeString, Optional: true},
						"client_secret": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
						"scope":        {Type: schema.TypeString, Optional: true},
						"redirect_url": {Type: schema.TypeString, Optional: true},
						"username":     {Type: schema.TypeString, Optional: true},
						"password": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
						"access_token_expired_msg_pattern": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"selenium_script": {
							Description: "Selenium script that drives the login flow. Used with " +
								"`AUTH_CODE`/`IMPLICIT` grants.",
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem:     seleniumScriptElem,
						},
					},
				},
			},

			"created": {Type: schema.TypeString, Computed: true},
		},

		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			form := d.Get("form_record").([]interface{})
			server := d.Get("server_record").([]interface{})
			oauth2 := d.Get("oauth2_record").([]interface{})
			if len(form) == 0 && len(server) == 0 && len(oauth2) == 0 {
				return errors.New("one of form_record, server_record or oauth2_record is required: " +
					"a record with none has no credential for the crawler to use")
			}
			if len(form) == 1 {
				m := form[0].(map[string]interface{})
				hasStandard := m["username"].(string) != "" && m["password"].(string) != ""
				hasCustom := len(m["field"].([]interface{})) > 0
				hasSelenium := len(m["selenium_script"].([]interface{})) > 0
				if !hasStandard && !hasCustom && !hasSelenium {
					return errors.New("form_record needs either username and password " +
						"(a STANDARD record), at least one field block (a CUSTOM record), " +
						"or a selenium_script block (a SELENIUM record)")
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

func wasSeleniumScriptFrom(raw []interface{}) *qps.WASSeleniumScript {
	if len(raw) != 1 {
		return nil
	}
	m, ok := raw[0].(map[string]interface{})
	if !ok {
		return nil
	}
	return &qps.WASSeleniumScript{
		Name:  m["name"].(string),
		Data:  m["data"].(string),
		Regex: m["regex"].(string),
	}
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
			SubType:        m["sub_type"].(string),
			LoginURL:       m["login_url"].(string),
			Username:       m["username"].(string),
			Password:       m["password"].(string),
			SSLOnly:        m["ssl_only"].(bool),
			AuthVault:      m["auth_vault"].(bool),
			Fields:         wasAuthFieldsFrom(m["field"].([]interface{})),
			SeleniumScript: wasSeleniumScriptFrom(m["selenium_script"].([]interface{})),
			SeleniumCreds:  m["selenium_creds"].(bool),
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

	if raw := d.Get("oauth2_record").([]interface{}); len(raw) == 1 {
		m := raw[0].(map[string]interface{})
		in.OAuth2 = &qps.WASOAuth2Record{
			GrantType:                    m["grant_type"].(string),
			AccessTokenURL:               m["access_token_url"].(string),
			ClientID:                     m["client_id"].(string),
			ClientSecret:                 m["client_secret"].(string),
			Scope:                        m["scope"].(string),
			RedirectURL:                  m["redirect_url"].(string),
			Username:                     m["username"].(string),
			Password:                     m["password"].(string),
			SeleniumScript:               wasSeleniumScriptFrom(m["selenium_script"].([]interface{})),
			AccessTokenExpiredMsgPattern: m["access_token_expired_msg_pattern"].(string),
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
			return qpsNotFoundOnRead(d, "WAS authentication record")
		}
		return diag.FromErr(err)
	}

	tagIDs := make([]string, 0, len(rec.Tags))
	for _, t := range rec.Tags {
		tagIDs = append(tagIDs, t.ID)
	}

	// form_record, server_record, oauth2_record and their credential
	// contents are deliberately not set here: Qualys masks them on read, so
	// there is nothing trustworthy to refresh state with. Leaving them out
	// of this map keeps whatever the last apply configured.
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
