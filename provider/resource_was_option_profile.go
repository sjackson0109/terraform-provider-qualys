package provider

import (
	"context"
	"errors"

	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceWASOptionProfile() *schema.Resource {
	return &schema.Resource{
		Description: "A Qualys WAS scan option profile, referenced by web application scans " +
			"and scan schedules.",

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
			"max_crawl_requests": {
				Description: "Maximum number of requests the crawler issues during a scan.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"performance": {
				Description: "Scan performance level.",
				Type:        schema.TypeString,
				Optional:    true,
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
			},
			"timeout_error_threshold": {
				Description: "Number of timeout errors tolerated before the scan engine backs off.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"unexpected_error_threshold": {
				Description: "Number of unexpected errors tolerated before the scan engine backs off.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"detect_credit_card_numbers": {
				Description: "Flag credit card numbers found in application responses as sensitive content.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"detect_social_security_numbers": {
				Description: "Flag social security numbers found in application responses as sensitive content.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
		},
	}
}

func wasOptionProfileInputFrom(d *schema.ResourceData) qps.WASOptionProfileInput {
	return qps.WASOptionProfileInput{
		Name:                     d.Get("name").(string),
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
