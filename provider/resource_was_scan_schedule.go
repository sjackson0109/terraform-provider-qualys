package provider

import (
	"context"
	"errors"

	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceWASScanSchedule() *schema.Resource {
	return &schema.Resource{
		Description: "A recurring Qualys WAS scan against a single web application.\n\n" +
			"**Coverage note:** this resource covers the scan-schedule element model doc " +
			"11 (this provider's discovery notes) confirms by name from the official " +
			"Quick Reference — a single web-application target, type, option profile, " +
			"start date/time zone/occurrence type, and the simple id/boolean options. It " +
			"does **not** cover tag-based (multi-web-app) targeting, or the per-occurrence " +
			"recurrence detail (every N days/weeks/months, which weekdays, which day of " +
			"month) for `DAILY`/`WEEKLY`/`MONTHLY` schedules — those field names were not " +
			"reachable in any source obtained while building this (docs.qualys.com and " +
			"every mirror tried are blocked from this environment's network egress). A " +
			"recurring schedule created here uses whatever default cadence Qualys applies " +
			"when those fields are omitted; verify the resulting cadence in the Qualys UI " +
			"after the first apply. The JSON wrapper class name is inferred from this " +
			"API's naming convention, not confirmed from a sample payload.",

		CreateContext: resourceWASScanScheduleCreate,
		ReadContext:   resourceWASScanScheduleRead,
		UpdateContext: resourceWASScanScheduleUpdate,
		DeleteContext: resourceWASScanScheduleDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Schedule name.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"web_app_id": {
				Description: "ID of the `qualys_web_application` to scan.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"type": {
				Description: "Scan type.",
				Type:        schema.TypeString,
				Required:    true,
				ValidateFunc: validation.StringInSlice([]string{
					qps.WASScanTypeDiscovery, qps.WASScanTypeVulnerability,
				}, false),
			},
			"option_profile_id": {
				Description: "ID of the `qualys_was_option_profile` to scan with. Optional " +
					"only if the target web application has a default option profile " +
					"configured in Qualys; if it does not, the API rejects a schedule " +
					"without this set.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"start_date": {
				Description: "When the schedule first runs. Format is not corroborated " +
					"against a sample payload — pass whatever the API's error message asks " +
					"for if the first apply is rejected, and please report back what worked.",
				Type:     schema.TypeString,
				Required: true,
			},
			"time_zone": {
				Description: "Time zone for `start_date` and recurrence calculations. Not " +
					"validated client-side, matching `qualys_scan_schedule.time_zone_code`.",
				Type:     schema.TypeString,
				Required: true,
			},
			"occurrence_type": {
				Description: "How the schedule repeats. See the resource-level note: " +
					"`DAILY`/`WEEKLY`/`MONTHLY` cadence detail is not configurable here.",
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					qps.WASOccurrenceOnce, qps.WASOccurrenceDaily,
					qps.WASOccurrenceWeekly, qps.WASOccurrenceMonthly,
				}, false),
			},
			"notification": {
				Description: "Email the scan's summary when it completes.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"reschedule": {
				Description: "Re-run automatically if a scan under this schedule is skipped " +
					"(for example because a scanner was unavailable).",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"randomize_scan": {
				Description: "Start at a randomised time within the scheduled window, " +
					"rather than exactly on it.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"web_app_auth_record_id": {
				Description: "ID of the `qualys_was_auth_record` to authenticate with. " +
					"Omit to use the web application's default authentication record.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"proxy_id": {
				Description: "ID of a WAS proxy configuration to scan through. Not managed " +
					"by this provider; reference an ID configured outside Terraform.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"dns_override_id": {
				Description: "ID of a `qualys_was_dns_override_record` (not yet implemented " +
					"by this provider) to resolve the target through. Reference an ID " +
					"configured outside Terraform.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"cancel_option": {
				Description: "How a scan under this schedule is cancelled if it runs over " +
					"its window.",
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					"DEFAULT", "SPECIFIC",
				}, false),
			},
			"send_mail": {
				Description: "Email scan-launch notifications for this schedule.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},

			"active":  {Type: schema.TypeBool, Computed: true},
			"created": {Type: schema.TypeString, Computed: true},
		},
	}
}

func wasScanScheduleInputFrom(d *schema.ResourceData) qps.WASScanScheduleInput {
	return qps.WASScanScheduleInput{
		Name:               d.Get("name").(string),
		Type:               d.Get("type").(string),
		WebAppID:           d.Get("web_app_id").(string),
		OptionProfileID:    d.Get("option_profile_id").(string),
		StartDate:          d.Get("start_date").(string),
		TimeZone:           d.Get("time_zone").(string),
		OccurrenceType:     d.Get("occurrence_type").(string),
		Notification:       d.Get("notification").(bool),
		Reschedule:         d.Get("reschedule").(bool),
		RandomizeScan:      d.Get("randomize_scan").(bool),
		WebAppAuthRecordID: d.Get("web_app_auth_record_id").(string),
		ProxyID:            d.Get("proxy_id").(string),
		DNSOverrideID:      d.Get("dns_override_id").(string),
		CancelOption:       d.Get("cancel_option").(string),
		SendMail:           d.Get("send_mail").(bool),
	}
}

func resourceWASScanScheduleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	sched, err := c.CreateWASScanSchedule(ctx, wasScanScheduleInputFrom(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(sched.ID)
	return resourceWASScanScheduleRead(ctx, d, meta)
}

func resourceWASScanScheduleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	sched, err := c.GetWASScanSchedule(ctx, d.Id())
	if err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			d.SetId("")
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "WAS scan schedule no longer visible; removing it from state",
				Detail: "Qualys reported OBJECT_NOT_FOUND, which it also returns for objects " +
					"outside the caller's scope. If the schedule still exists and the credentials' " +
					"scope changed, restore the scope before applying, or Terraform will create a " +
					"duplicate.",
			}}
		}
		return diag.FromErr(err)
	}

	return diag.FromErr(setAll(d, map[string]interface{}{
		"name":                   sched.Name,
		"type":                   sched.Type,
		"web_app_id":             sched.WebAppID,
		"option_profile_id":      sched.OptionProfileID,
		"start_date":             sched.StartDate,
		"time_zone":              sched.TimeZone,
		"occurrence_type":        sched.OccurrenceType,
		"notification":           sched.Notification,
		"reschedule":             sched.Reschedule,
		"randomize_scan":         sched.RandomizeScan,
		"web_app_auth_record_id": sched.WebAppAuthRecordID,
		"proxy_id":               sched.ProxyID,
		"dns_override_id":        sched.DNSOverrideID,
		"cancel_option":          sched.CancelOption,
		"send_mail":              sched.SendMail,
		"active":                 sched.Active,
		"created":                sched.Created,
	}))
}

func resourceWASScanScheduleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateWASScanSchedule(ctx, d.Id(), wasScanScheduleInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceWASScanScheduleRead(ctx, d, meta)
}

func resourceWASScanScheduleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteWASScanSchedule(ctx, d.Id()); err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
