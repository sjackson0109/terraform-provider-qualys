package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjackson0109/terraform-provider-qualys/qps"
)

func resourceWASScanSchedule() *schema.Resource {
	return &schema.Resource{
		Description: "A recurring Qualys WAS scan against a single web application.\n\n" +
			"**Provenance:** the element model is Confirmed against a user-supplied " +
			"\"Create and Update Scan Schedule\" walkthrough (transcribed, not a verbatim " +
			"PDF quote, so treated as strong but not absolute evidence) covering " +
			"`ONCE`/`DAILY`/`WEEKLY` occurrences. `MONTHLY` is deliberately NOT exposed " +
			"here: an earlier version of this resource modelled it " +
			"(`day_of_month`/`every_n_months`) by analogy from a WAS *report* schedule " +
			"example, and a later user-supplied \"Gap Review\" document explicitly " +
			"corrected that: the payload shape is confirmed for report schedules but not " +
			"for this object, and should not be assumed to match. The underlying client " +
			"rejects a MONTHLY request rather than send an unconfirmed payload. Use " +
			"`qualys_was_report_schedule` if MONTHLY reporting (not scanning) is what you " +
			"need, or supply a confirmed WasScanSchedule MONTHLY example to close this " +
			"gap. Tag-based (multi-web-app) targeting is not modelled either: WAS " +
			"Multi-Scan launches confirm the shape exists, but not that WasScanSchedule " +
			"accepts the same payload.",

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
				Description: "ID of the `qualys_was_application` to scan.",
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
			"active": {
				Description: "Whether the schedule runs. Toggling this after creation uses " +
					"the dedicated activate/deactivate endpoints rather than a regular update.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"option_profile_id": {
				Description: "ID of the `qualys_was_option_profile` to scan with. Required " +
					"unless the target web application has a default option profile.",
				Type:     schema.TypeString,
				Optional: true,
			},

			"web_app_auth_record_id": {
				Description:   "ID of the `qualys_was_auth_record` to authenticate with.",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"web_app_auth_record_use_default"},
			},
			"web_app_auth_record_use_default": {
				Description:   "Use the web application's default authentication record.",
				Type:          schema.TypeBool,
				Optional:      true,
				ConflictsWith: []string{"web_app_auth_record_id"},
			},
			"scanner_type": {
				Description: "Scanner selection.",
				Type:        schema.TypeString,
				Optional:    true,
				ValidateFunc: validation.StringInSlice([]string{
					qps.WASScannerExternal, qps.WASScannerInternal,
				}, false),
			},
			"scanner_friendly_name": {
				Description: "Name of the internal scanner appliance. Used with " +
					"`scanner_type = \"INTERNAL\"`.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"cancel_option": {
				Description: "How a scan under this schedule is cancelled if it runs over " +
					"its window: `DEFAULT` (use the web application's configured behaviour) " +
					"or `SPECIFIC` (use this schedule's own behaviour).",
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					qps.WASCancelOptionDefault, qps.WASCancelOptionSpecific,
				}, false),
			},
			"proxy_id": {
				Description: "ID of a WAS proxy configuration to scan through. Not managed " +
					"by this provider; reference an ID configured outside Terraform.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"dns_override_id": {
				Description: "ID of a `qualys_was_dns_override` to resolve the target through.",
				Type:        schema.TypeString,
				Optional:    true,
			},

			"start_date": {
				Description: "When the schedule first runs, e.g. `2026-08-16T02:00:00Z`.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"time_zone_code": {
				Description: "Time zone for `start_date` and recurrence calculations, e.g. " +
					"`Europe/London`. Not validated client-side, matching " +
					"`qualys_vm_scan_schedule.time_zone_code`.",
				Type:     schema.TypeString,
				Required: true,
			},
			"occurrence_type": {
				Description: "How the schedule repeats: `ONCE`, `DAILY` or `WEEKLY`. " +
					"`MONTHLY` is not supported by this resource — see the resource-level note.",
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					qps.WASOccurrenceOnce, qps.WASOccurrenceDaily, qps.WASOccurrenceWeekly,
				}, false),
			},
			"cancel_after_hours": {
				Description: "Automatically terminate a scan under this schedule after this " +
					"many hours.",
				Type:     schema.TypeInt,
				Optional: true,
			},
			"every_n_days": {
				Description: "Repeat every N days. Used with `occurrence_type = \"DAILY\"`.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"every_n_weeks": {
				Description: "Repeat every N weeks. Used with `occurrence_type = \"WEEKLY\"`.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			"on_days": {
				Description: "Days of the week to run on, e.g. `[\"SATURDAY\", \"SUNDAY\"]`. " +
					"Used with `occurrence_type = \"WEEKLY\"`.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						qps.WASWeekDaySunday, qps.WASWeekDayMonday, qps.WASWeekDayTuesday,
						qps.WASWeekDayWednesday, qps.WASWeekDayThursday, qps.WASWeekDayFriday,
						qps.WASWeekDaySaturday,
					}, false),
				},
			},
			"occurrence_count": {
				Description: "End the schedule after this many occurrences. Used with " +
					"`occurrence_type = \"WEEKLY\"`.",
				Type:     schema.TypeInt,
				Optional: true,
			},

			"notification": {
				Description: "Email notification sent before a scan under this schedule runs.",
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"active": {Type: schema.TypeBool, Required: true},
						"reschedule": {
							Description: "Also notify when a scan is rescheduled rather " +
								"than run as planned.",
							Type:     schema.TypeBool,
							Optional: true,
						},
						"delay_amount": {
							Description: "How long before the scan to send the " +
								"notification, together with `delay_scale`.",
							Type:     schema.TypeInt,
							Optional: true,
						},
						"delay_scale": {
							Description: "Unit for `delay_amount`. The only value seen in " +
								"available evidence is `DAY`; not validated client-side.",
							Type:     schema.TypeString,
							Optional: true,
						},
						"recipients": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"message": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"send_mail": {
				Description: "Email a scan-completion notification for this schedule.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"send_one_mail": {
				Description: "Consolidate the completion notification into a single email.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"send_mail_from_address_option": {
				Description: "Sender for completion notifications. Requires `send_mail`.",
				Type:        schema.TypeString,
				Optional:    true,
				ValidateFunc: validation.StringInSlice([]string{
					qps.WASSendMailFromQualysSupport, qps.WASSendMailFromOwner,
				}, false),
			},

			"created": {Type: schema.TypeString, Computed: true},
		},
	}
}

// wasScheduleRecurrenceFrom builds a DAILY/WEEKLY recurrence only — MONTHLY
// (DayOfMonth/EveryNMonths) is deliberately never populated here. See the
// resource-level note on MONTHLY for why.
func wasScheduleRecurrenceFrom(d *schema.ResourceData) *qps.WASScheduleRecurrence {
	everyNDays := d.Get("every_n_days").(int)
	everyNWeeks := d.Get("every_n_weeks").(int)
	onDays := stringSet(d, "on_days")
	occurrenceCount := d.Get("occurrence_count").(int)
	if everyNDays == 0 && everyNWeeks == 0 && len(onDays) == 0 && occurrenceCount == 0 {
		return nil
	}
	return &qps.WASScheduleRecurrence{
		EveryNDays:      everyNDays,
		EveryNWeeks:     everyNWeeks,
		OnDays:          onDays,
		OccurrenceCount: occurrenceCount,
	}
}

func wasScheduleNotificationFrom(d *schema.ResourceData) *qps.WASScheduleNotification {
	raw := d.Get("notification").([]interface{})
	if len(raw) != 1 {
		return nil
	}
	m := raw[0].(map[string]interface{})
	recipients := make([]string, 0)
	if set, ok := m["recipients"].(*schema.Set); ok {
		for _, v := range set.List() {
			if s, ok := v.(string); ok && s != "" {
				recipients = append(recipients, s)
			}
		}
	}
	return &qps.WASScheduleNotification{
		Active:      m["active"].(bool),
		Reschedule:  m["reschedule"].(bool),
		DelayAmount: m["delay_amount"].(int),
		DelayScale:  m["delay_scale"].(string),
		Recipients:  recipients,
		Message:     m["message"].(string),
	}
}

func wasScanScheduleInputFrom(d *schema.ResourceData) qps.WASScanScheduleInput {
	return qps.WASScanScheduleInput{
		Name:                      d.Get("name").(string),
		Type:                      d.Get("type").(string),
		Active:                    d.Get("active").(bool),
		WebAppID:                  d.Get("web_app_id").(string),
		OptionProfileID:           d.Get("option_profile_id").(string),
		WebAppAuthRecordID:        d.Get("web_app_auth_record_id").(string),
		WebAppAuthRecordIsDefault: d.Get("web_app_auth_record_use_default").(bool),
		ScannerType:               d.Get("scanner_type").(string),
		ScannerFriendlyName:       d.Get("scanner_friendly_name").(string),
		CancelOption:              d.Get("cancel_option").(string),
		ProxyID:                   d.Get("proxy_id").(string),
		DNSOverrideID:             d.Get("dns_override_id").(string),
		StartDate:                 d.Get("start_date").(string),
		TimeZoneCode:              d.Get("time_zone_code").(string),
		OccurrenceType:            d.Get("occurrence_type").(string),
		CancelAfterHours:          d.Get("cancel_after_hours").(int),
		Recurrence:                wasScheduleRecurrenceFrom(d),
		Notification:              wasScheduleNotificationFrom(d),
		SendMail:                  d.Get("send_mail").(bool),
		SendOneMail:               d.Get("send_one_mail").(bool),
		SendMailFromAddressOption: d.Get("send_mail_from_address_option").(string),
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
			return qpsNotFoundOnRead(d, "WAS scan schedule")
		}
		return diag.FromErr(err)
	}

	values := map[string]interface{}{
		"name":                            sched.Name,
		"type":                            sched.Type,
		"active":                          sched.Active,
		"web_app_id":                      sched.WebAppID,
		"option_profile_id":               sched.OptionProfileID,
		"web_app_auth_record_id":          sched.WebAppAuthRecordID,
		"web_app_auth_record_use_default": sched.WebAppAuthRecordIsDefault,
		"scanner_type":                    sched.ScannerType,
		"scanner_friendly_name":           sched.ScannerFriendlyName,
		"cancel_option":                   sched.CancelOption,
		"proxy_id":                        sched.ProxyID,
		"dns_override_id":                 sched.DNSOverrideID,
		"start_date":                      sched.StartDate,
		"time_zone_code":                  sched.TimeZoneCode,
		"occurrence_type":                 sched.OccurrenceType,
		"cancel_after_hours":              sched.CancelAfterHours,
		"send_mail":                       sched.SendMail,
		"send_one_mail":                   sched.SendOneMail,
		"send_mail_from_address_option":   sched.SendMailFromAddressOption,
		"created":                         sched.Created,
	}

	if sched.Recurrence != nil {
		values["every_n_days"] = sched.Recurrence.EveryNDays
		values["every_n_weeks"] = sched.Recurrence.EveryNWeeks
		values["on_days"] = sched.Recurrence.OnDays
		values["occurrence_count"] = sched.Recurrence.OccurrenceCount
	}

	if sched.Notification != nil {
		values["notification"] = []interface{}{map[string]interface{}{
			"active":       sched.Notification.Active,
			"reschedule":   sched.Notification.Reschedule,
			"delay_amount": sched.Notification.DelayAmount,
			"delay_scale":  sched.Notification.DelayScale,
			"recipients":   sched.Notification.Recipients,
			"message":      sched.Notification.Message,
		}}
	}

	return diag.FromErr(setAll(d, values))
}

func resourceWASScanScheduleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateWASScanSchedule(ctx, d.Id(), wasScanScheduleInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}

	// active is toggled through the dedicated activate/deactivate endpoints,
	// confirmed as separate from a regular field update.
	if d.HasChange("active") {
		var toggleErr error
		if d.Get("active").(bool) {
			toggleErr = c.ActivateWASScanSchedule(ctx, d.Id())
		} else {
			toggleErr = c.DeactivateWASScanSchedule(ctx, d.Id())
		}
		if toggleErr != nil {
			return diag.FromErr(toggleErr)
		}
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
