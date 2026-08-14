package provider

import (
	"context"
	"errors"

	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceWASReportSchedule() *schema.Resource {
	return &schema.Resource{
		Description: "A recurring, automated Qualys WAS report generation and delivery — " +
			"distinct from `qualys_was_scan_schedule`, which runs scans. Both share the " +
			"same recurrence model.\n\n" +
			"**Provenance:** Confirmed against a user-supplied \"Create and Update Report " +
			"Schedule\" walkthrough (transcribed, not a verbatim PDF quote, so treated as " +
			"strong but not absolute evidence). That walkthrough is also the source of " +
			"`qualys_was_scan_schedule`'s `MONTHLY` recurrence support, applied there by " +
			"analogy. Only a single `web_app_id` target is modelled.",

		CreateContext: resourceWASReportScheduleCreate,
		ReadContext:   resourceWASReportScheduleRead,
		UpdateContext: resourceWASReportScheduleUpdate,
		DeleteContext: resourceWASReportScheduleDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Schedule name.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"report_template_id": {
				Description: "ID of the WAS report template to use. Report templates are " +
					"read-only through the API (see `data.qualys_was_report_templates`) — " +
					"create them in the Qualys UI and reference the ID here.",
				Type:     schema.TypeString,
				Required: true,
			},
			"web_app_id": {
				Description: "ID of the `qualys_was_application` to report on.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"active": {
				Description: "Whether the schedule runs. Toggling this after creation uses " +
					"the dedicated activate/deactivate endpoints rather than a regular update.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"output_format": {
				Description: "Report output format.",
				Type:        schema.TypeString,
				Required:    true,
				ValidateFunc: validation.StringInSlice([]string{
					qps.WASReportFormatPDF, qps.WASReportFormatHTML,
					qps.WASReportFormatXML, qps.WASReportFormatCSV,
				}, false),
			},
			"recipients": {
				Description: "Email addresses to deliver the generated report to.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			"start_date": {
				Description: "When the schedule first runs, e.g. `2026-08-16T06:00:00Z`.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"time_zone_code": {
				Description: "Time zone for `start_date` and recurrence calculations, e.g. " +
					"`Europe/London`. Not validated client-side.",
				Type:     schema.TypeString,
				Required: true,
			},
			"occurrence_type": {
				Description: "How the schedule repeats.",
				Type:        schema.TypeString,
				Required:    true,
				ValidateFunc: validation.StringInSlice([]string{
					qps.WASOccurrenceOnce, qps.WASOccurrenceDaily,
					qps.WASOccurrenceWeekly, qps.WASOccurrenceMonthly,
				}, false),
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
				Description: "Days of the week to run on, e.g. `[\"MONDAY\"]`. Used with " +
					"`occurrence_type = \"WEEKLY\"`.",
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
			"day_of_month": {
				Description: "Day of the month to run on. Used with " +
					"`occurrence_type = \"MONTHLY\"`.",
				Type:     schema.TypeInt,
				Optional: true,
			},
			"every_n_months": {
				Description: "Repeat every N months. Used with `occurrence_type = \"MONTHLY\"`.",
				Type:        schema.TypeInt,
				Optional:    true,
			},

			"created": {Type: schema.TypeString, Computed: true},
		},
	}
}

func wasReportScheduleInputFrom(d *schema.ResourceData) qps.WASReportScheduleInput {
	return qps.WASReportScheduleInput{
		Name:             d.Get("name").(string),
		Active:           d.Get("active").(bool),
		ReportTemplateID: d.Get("report_template_id").(string),
		WebAppID:         d.Get("web_app_id").(string),
		OutputFormat:     d.Get("output_format").(string),
		Recipients:       stringSet(d, "recipients"),
		StartDate:        d.Get("start_date").(string),
		TimeZoneCode:     d.Get("time_zone_code").(string),
		OccurrenceType:   d.Get("occurrence_type").(string),
		Recurrence:       wasScheduleRecurrenceFrom(d),
	}
}

func resourceWASReportScheduleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	sched, err := c.CreateWASReportSchedule(ctx, wasReportScheduleInputFrom(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(sched.ID)
	return resourceWASReportScheduleRead(ctx, d, meta)
}

func resourceWASReportScheduleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	sched, err := c.GetWASReportSchedule(ctx, d.Id())
	if err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			d.SetId("")
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "WAS report schedule no longer visible; removing it from state",
				Detail: "Qualys reported OBJECT_NOT_FOUND, which it also returns for objects " +
					"outside the caller's scope. If the schedule still exists and the credentials' " +
					"scope changed, restore the scope before applying, or Terraform will create a " +
					"duplicate.",
			}}
		}
		return diag.FromErr(err)
	}

	values := map[string]interface{}{
		"name":               sched.Name,
		"active":             sched.Active,
		"report_template_id": sched.ReportTemplateID,
		"web_app_id":         sched.WebAppID,
		"output_format":      sched.OutputFormat,
		"recipients":         sched.Recipients,
		"start_date":         sched.StartDate,
		"time_zone_code":     sched.TimeZoneCode,
		"occurrence_type":    sched.OccurrenceType,
		"created":            sched.Created,
	}
	if sched.Recurrence != nil {
		values["every_n_days"] = sched.Recurrence.EveryNDays
		values["every_n_weeks"] = sched.Recurrence.EveryNWeeks
		values["on_days"] = sched.Recurrence.OnDays
		values["occurrence_count"] = sched.Recurrence.OccurrenceCount
		values["day_of_month"] = sched.Recurrence.DayOfMonth
		values["every_n_months"] = sched.Recurrence.EveryNMonths
	}

	return diag.FromErr(setAll(d, values))
}

func resourceWASReportScheduleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateWASReportSchedule(ctx, d.Id(), wasReportScheduleInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("active") {
		var toggleErr error
		if d.Get("active").(bool) {
			toggleErr = c.ActivateWASReportSchedule(ctx, d.Id())
		} else {
			toggleErr = c.DeactivateWASReportSchedule(ctx, d.Id())
		}
		if toggleErr != nil {
			return diag.FromErr(toggleErr)
		}
	}

	return resourceWASReportScheduleRead(ctx, d, meta)
}

func resourceWASReportScheduleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteWASReportSchedule(ctx, d.Id()); err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
