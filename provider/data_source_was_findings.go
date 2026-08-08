package provider

import (
	"context"
	"sort"

	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceWASFindings() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys WAS scan findings: vulnerabilities, sensitive-content " +
			"exposures and information-gathered items reported against web applications. " +
			"Read-only — finding lifecycle actions (ignore, retest, severity overrides) are " +
			"not implemented by this provider.",

		ReadContext: dataSourceWASFindingsRead,

		Schema: map[string]*schema.Schema{
			"web_app_id": {
				Description: "Restrict to findings on this web application.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"severity": {
				Description: "Restrict to this severity level (1-5).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"status": {
				Description: "Restrict to this status: one of `NEW`, `ACTIVE`, `REOPENED`, " +
					"`FIXED`, `PROTECTED`.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"type": {
				Description: "Restrict to this finding type: one of `VULNERABILITY`, " +
					"`SENSITIVE_CONTENT`, `INFORMATION_GATHERED`.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"finding_type": {
				Description: "Restrict to this source: one of `QUALYS`, `MANUAL`, `BURP`.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"is_ignored": {
				Description: "Restrict to findings with this ignored state.",
				Type:        schema.TypeBool,
				Optional:    true,
			},

			"findings": {
				Description: "Matching findings, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":                  {Type: schema.TypeString, Computed: true},
						"qid":                 {Type: schema.TypeString, Computed: true},
						"name":                {Type: schema.TypeString, Computed: true},
						"type":                {Type: schema.TypeString, Computed: true},
						"severity":            {Type: schema.TypeInt, Computed: true},
						"status":              {Type: schema.TypeString, Computed: true},
						"finding_type":        {Type: schema.TypeString, Computed: true},
						"url":                 {Type: schema.TypeString, Computed: true},
						"web_app_id":          {Type: schema.TypeString, Computed: true},
						"web_app_name":        {Type: schema.TypeString, Computed: true},
						"first_detected_date": {Type: schema.TypeString, Computed: true},
						"last_detected_date":  {Type: schema.TypeString, Computed: true},
						"is_ignored":          {Type: schema.TypeBool, Computed: true},
						"cvss_v3_base":        {Type: schema.TypeFloat, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceWASFindingsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	filter := qps.WASFindingFilter{
		WebAppID:    d.Get("web_app_id").(string),
		Severity:    d.Get("severity").(string),
		Status:      d.Get("status").(string),
		Type:        d.Get("type").(string),
		FindingType: d.Get("finding_type").(string),
	}
	if v, ok := d.GetOkExists("is_ignored"); ok {
		b := v.(bool)
		filter.IsIgnored = &b
	}

	findings, err := c.SearchWASFindings(ctx, filter)
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(findings, func(i, j int) bool { return findings[i].ID < findings[j].ID })

	out := make([]interface{}, 0, len(findings))
	ids := make([]string, 0, len(findings))
	for _, f := range findings {
		ids = append(ids, f.ID)
		out = append(out, map[string]interface{}{
			"id":                  f.ID,
			"qid":                 f.QID,
			"name":                f.Name,
			"type":                f.Type,
			"severity":            f.Severity,
			"status":              f.Status,
			"finding_type":        f.FindingType,
			"url":                 f.URL,
			"web_app_id":          f.WebAppID,
			"web_app_name":        f.WebAppName,
			"first_detected_date": f.FirstDetectedDate,
			"last_detected_date":  f.LastDetectedDate,
			"is_ignored":          f.IsIgnored,
			"cvss_v3_base":        f.CVSSV3Base,
		})
	}

	if err := d.Set("findings", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
