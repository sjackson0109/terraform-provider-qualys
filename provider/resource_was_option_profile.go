package provider

import (
	"context"
	"errors"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceWASOptionProfile() *schema.Resource {
	return &schema.Resource{
		Description: "A Qualys WAS scan option profile, referenced by web application scans " +
			"and scan schedules. Also where WAS policy-compliance settings (PCI DSS, OWASP " +
			"Top 10, SSL/TLS, crawling, exclusions, …) live — Qualys does not model policy " +
			"compliance as a separate object; a user-supplied example confirmed this and " +
			"caught a real bug: the wire wrapper key is `OptionProfile`, not the " +
			"`WasOptionProfile` this resource had used since its first version, which would " +
			"have failed every create/update against a live tenant. `comments` is also new, " +
			"confirmed as a flat string by the same example.",

		CreateContext: resourceWASOptionProfileCreate,
		ReadContext:   resourceWASOptionProfileRead,
		UpdateContext: resourceWASOptionProfileUpdate,
		DeleteContext: resourceWASOptionProfileDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Option profile name.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"comments": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			// These seven fields are Optional+Computed, not just Optional. The
			// create encoder (qps.CreateWASOptionProfile) omits any of them left
			// unset so Qualys can apply its own server-side default; without
			// Computed, Terraform would expect the post-apply state to match the
			// config-implied zero value and fail every create that omits one of
			// them with "Provider produced inconsistent result after apply".
			// Computed also means an unconfigured field's value survives from
			// state into the next plan, so a later update (which sends every
			// field unconditionally, see qps.UpdateWASOptionProfile) sends the
			// real prior value rather than a zero value that would blank out or
			// get rejected for the two enum fields below.
			"max_crawl_requests": {
				Description: "Maximum number of requests the crawler issues during a scan.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"performance": {
				Description: "Scan performance level.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ValidateFunc: validation.StringInSlice([]string{
					qps.WASPerformanceLow, qps.WASPerformanceMedium, qps.WASPerformanceHigh,
				}, false),
			},
			"bruteforce_option": {
				Description: "Brute-force authentication testing mode. Accepted values are " +
					"not yet confirmed against official documentation, so this field is not " +
					"validated client-side; an invalid value is rejected by the API at apply time.",
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"timeout_error_threshold": {
				Description: "Number of timeout errors tolerated before the scan engine backs off.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"unexpected_error_threshold": {
				Description: "Number of unexpected errors tolerated before the scan engine backs off.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"detect_credit_card_numbers": {
				Description: "Flag credit card numbers found in application responses as sensitive content.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"detect_social_security_numbers": {
				Description: "Flag social security numbers found in application responses as sensitive content.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func wasOptionProfileInputFrom(d *schema.ResourceData) qps.WASOptionProfileInput {
	return qps.WASOptionProfileInput{
		Name:                     d.Get("name").(string),
		Comments:                 d.Get("comments").(string),
		MaxCrawlRequests:         d.Get("max_crawl_requests").(int),
		Performance:              d.Get("performance").(string),
		BruteforceOption:         d.Get("bruteforce_option").(string),
		TimeoutErrorThreshold:    d.Get("timeout_error_threshold").(int),
		UnexpectedErrorThreshold: d.Get("unexpected_error_threshold").(int),
		DetectCreditCardNumbers:  d.Get("detect_credit_card_numbers").(bool),
		DetectSocialSecurityNums: d.Get("detect_social_security_numbers").(bool),
	}
}

func resourceWASOptionProfileCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	profile, err := c.CreateWASOptionProfile(ctx, wasOptionProfileInputFrom(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(profile.ID)
	return resourceWASOptionProfileRead(ctx, d, meta)
}

func resourceWASOptionProfileRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	profile, err := c.GetWASOptionProfile(ctx, d.Id())
	if err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			d.SetId("")
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "WAS option profile no longer visible; removing it from state",
				Detail: "Qualys reported OBJECT_NOT_FOUND, which it also returns for objects " +
					"outside the caller's scope. If the profile still exists and the credentials' " +
					"scope changed, restore the scope before applying, or Terraform will create a " +
					"duplicate.",
			}}
		}
		return diag.FromErr(err)
	}

	return diag.FromErr(setAll(d, map[string]interface{}{
		"name":                           profile.Name,
		"comments":                       profile.Comments,
		"max_crawl_requests":             profile.MaxCrawlRequests,
		"performance":                    profile.Performance,
		"bruteforce_option":              profile.BruteforceOption,
		"timeout_error_threshold":        profile.TimeoutErrorThreshold,
		"unexpected_error_threshold":     profile.UnexpectedErrorThreshold,
		"detect_credit_card_numbers":     profile.DetectCreditCardNumbers,
		"detect_social_security_numbers": profile.DetectSocialSecurityNums,
	}))
}

func resourceWASOptionProfileUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateWASOptionProfile(ctx, d.Id(), wasOptionProfileInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceWASOptionProfileRead(ctx, d, meta)
}

func resourceWASOptionProfileDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteWASOptionProfile(ctx, d.Id()); err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
