package provider

import (
	"context"
	"errors"

	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceScanSchedule() *schema.Resource {
	return &schema.Resource{
		Description: "A recurring Qualys VM scan.\n\n" +
			"Schedules are the declarative way to run scans. A scan launched directly is a " +
			"job with no stable identity — re-running it produces a new scan rather than " +
			"updating the old one — so this provider models schedules and not individual " +
			"scan runs.",

		CreateContext: resourceScanScheduleCreate,
		ReadContext:   resourceScanScheduleRead,
		UpdateContext: resourceScanScheduleUpdate,
		DeleteContext: resourceScanScheduleDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"title": {
				Description: "Name of the scheduled scan.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"active": {
				Description: "Whether the schedule runs.\n\n" +
					"~> Qualys deactivates a schedule automatically once `recurrence` " +
					"occurrences have elapsed. If you set `recurrence`, expect this " +
					"attribute to drift to `false` on its own.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"option_profile_id": {
				Description:  "ID of the option profile defining how the scan behaves.",
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"option_profile_id", "option_profile_title"},
			},
			"option_profile_title": {
				Description:  "Title of the option profile, as an alternative to its ID.",
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"option_profile_id", "option_profile_title"},
			},

			// Targets
			"ips": ipSetSchema("Addresses, ranges or CIDR blocks to scan.", false),
			"asset_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tag_include_ids": {
				Description: "Asset tag IDs to scan. Using tags switches the schedule to " +
					"tag-based targeting, and `ips` and `asset_group_ids` are then ignored.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tag_exclude_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tag_include_selector": {
				Description:  "Whether an asset must match all or any of the included tags.",
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "any",
				ValidateFunc: validation.StringInSlice([]string{"all", "any"}, false),
			},
			"exclude_ips": ipSetSchema("Addresses, ranges or CIDR blocks to exclude.", false),
			"network_id": {
				Description: "Qualys network to scan within. Declare this explicitly.",
				Type:        schema.TypeString,
				Optional:    true,
			},

			// Scanner selection
			"scanner_names": {
				Description: "Named scanner appliances to scan from.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"use_default_scanner": {
				Description: "Scan from Qualys' external scanners.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"use_asset_group_scanners": {
				Description: "Scan from the appliances assigned to the target asset groups.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"use_tagset_scanners": {
				Description: "Scan from the appliances matching the target tags.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},

			// Recurrence
			"occurrence": {
				Description:  "How often the scan repeats.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"daily", "weekly", "monthly"}, false),
			},
			"frequency_days": {
				Description:  "Run every N days. Required when `occurrence` is `daily`.",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 365),
			},
			"frequency_weeks": {
				Description:  "Run every N weeks. Required when `occurrence` is `weekly`.",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 52),
			},
			"weekdays": {
				Description: "Days to run on, as names (`monday`, `tuesday`, ...). Required " +
					"when `occurrence` is `weekly`.\n\n" +
					"Note this takes day *names*, while `day_of_week` below takes a number.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday",
					}, false),
				},
			},
			"frequency_months": {
				Description:  "Run every N months. Required when `occurrence` is `monthly`.",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 12),
			},
			"day_of_month": {
				Description:  "Day of the month to run on. Alternative to `day_of_week` with `week_of_month`.",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 31),
			},
			"day_of_week": {
				Description: "Day of the week as a number, 0 being Sunday. Used with " +
					"`week_of_month`.\n\n" +
					"Note this takes a *number*, while `weekdays` above takes names.",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 6),
			},
			"week_of_month": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"first", "second", "third", "fourth", "last"}, false),
			},
			"recurrence": {
				Description: "Stop after this many occurrences. Qualys deactivates the " +
					"schedule once the count is reached, so `active` will drift to `false`.",
				Type:     schema.TypeInt,
				Optional: true,
			},

			// Time
			"start_date": {
				Description: "Date of the first run, as `MM/DD/YYYY`.\n\n" +
					"Required because the Qualys API only accepts the schedule time as a " +
					"complete group — start date, hour, minute, timezone and DST flag " +
					"together — and an update omitting any of them is rejected or silently " +
					"ignored.",
				Type:     schema.TypeString,
				Required: true,
			},
			"start_hour": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(0, 23),
			},
			"start_minute": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(0, 59),
			},
			"time_zone_code": {
				Description: "Qualys timezone code, for example `US-NY`.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"observe_dst": {
				Description: "Adjust the start time for daylight saving in the chosen timezone.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},

			// Duration
			"end_after_hours": {
				Description: "Cap on how long the scan may run. This is a duration limit, not " +
					"an end date — the API has no end-date parameter.",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 119),
			},
			"end_after_minutes": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 59),
			},

			// Notifications
			"notify_before": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"notify_before_unit": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"days", "hours", "minutes"}, false),
			},
			"notify_before_time":    {Type: schema.TypeInt, Optional: true},
			"notify_before_message": {Type: schema.TypeString, Optional: true},
			"notify_after":          {Type: schema.TypeBool, Optional: true, Default: false},
			"notify_after_message":  {Type: schema.TypeString, Optional: true},
			"recipient_group_ids": {
				Description: "Qualys distribution group IDs to notify. Only used when " +
					"`notify_before` or `notify_after` is set.\n\n" +
					"Qualys notifies distribution groups, not arbitrary email addresses; " +
					"manage group membership in the Qualys UI.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"active_state": {
				Description: "Raw Qualys state: `0` deactivated, `1` active, `2` active and " +
					"not paused, `3` paused. Values 2 and 3 apply to continuous schedules, " +
					"which is why `active` alone cannot express the state.",
				Type:     schema.TypeInt,
				Computed: true,
			},
		},

		CustomizeDiff: resourceScanScheduleCustomizeDiff,
	}
}

// resourceScanScheduleCustomizeDiff enforces the API's required-together groups
// at plan time. The API rejects an incomplete group, but the message it returns
// is not always clear about which field is missing.
func resourceScanScheduleCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	switch d.Get("occurrence").(string) {
	case "daily":
		if d.Get("frequency_days").(int) <= 0 {
			return errors.New("frequency_days is required when occurrence is \"daily\"")
		}
	case "weekly":
		if d.Get("frequency_weeks").(int) <= 0 {
			return errors.New("frequency_weeks is required when occurrence is \"weekly\"")
		}
		if s, ok := d.Get("weekdays").(*schema.Set); !ok || s.Len() == 0 {
			return errors.New("weekdays is required when occurrence is \"weekly\"")
		}
	case "monthly":
		if d.Get("frequency_months").(int) <= 0 {
			return errors.New("frequency_months is required when occurrence is \"monthly\"")
		}
		hasDayOfMonth := d.Get("day_of_month").(int) > 0
		hasWeekOfMonth := d.Get("week_of_month").(string) != ""
		if !hasDayOfMonth && !hasWeekOfMonth {
			return errors.New("a monthly schedule requires either day_of_month, or " +
				"day_of_week together with week_of_month")
		}
		if hasDayOfMonth && hasWeekOfMonth {
			return errors.New("day_of_month and week_of_month are alternatives; set one, not both")
		}
	}

	// A schedule with no target scans nothing.
	empty := func(key string) bool {
		s, ok := d.Get(key).(*schema.Set)
		return !ok || s.Len() == 0
	}
	if empty("ips") && empty("asset_group_ids") && empty("tag_include_ids") {
		return errors.New("a scan schedule needs a target: set ips, asset_group_ids or tag_include_ids")
	}

	// A schedule with no scanner cannot run.
	if empty("scanner_names") && !d.Get("use_default_scanner").(bool) &&
		!d.Get("use_asset_group_scanners").(bool) && !d.Get("use_tagset_scanners").(bool) {
		return errors.New("a scan schedule needs a scanner: set scanner_names, or one of " +
			"use_default_scanner, use_asset_group_scanners or use_tagset_scanners")
	}

	return nil
}

func scanScheduleInputFrom(d *schema.ResourceData) vmdr.ScanScheduleInput {
	tagIncludes := stringSet(d, "tag_include_ids")
	return vmdr.ScanScheduleInput{
		Title:  d.Get("title").(string),
		Active: d.Get("active").(bool),

		OptionProfileID:    d.Get("option_profile_id").(string),
		OptionProfileTitle: d.Get("option_profile_title").(string),

		IPs:                stringSet(d, "ips"),
		AssetGroupIDs:      stringSet(d, "asset_group_ids"),
		TargetFromTags:     len(tagIncludes) > 0,
		TagSetInclude:      tagIncludes,
		TagSetExclude:      stringSet(d, "tag_exclude_ids"),
		TagIncludeSelector: d.Get("tag_include_selector").(string),
		ExcludeIPs:         stringSet(d, "exclude_ips"),
		NetworkID:          d.Get("network_id").(string),

		ScannerNames:     stringSet(d, "scanner_names"),
		DefaultScanner:   d.Get("use_default_scanner").(bool),
		ScannersInAG:     d.Get("use_asset_group_scanners").(bool),
		ScannersInTagset: d.Get("use_tagset_scanners").(bool),

		Occurrence:      d.Get("occurrence").(string),
		FrequencyDays:   d.Get("frequency_days").(int),
		FrequencyWeeks:  d.Get("frequency_weeks").(int),
		Weekdays:        stringSet(d, "weekdays"),
		FrequencyMonths: d.Get("frequency_months").(int),
		DayOfMonth:      d.Get("day_of_month").(int),
		DayOfWeek:       d.Get("day_of_week").(int),
		WeekOfMonth:     d.Get("week_of_month").(string),
		Recurrence:      d.Get("recurrence").(int),

		StartDate:    d.Get("start_date").(string),
		StartHour:    d.Get("start_hour").(int),
		StartMinute:  d.Get("start_minute").(int),
		TimeZoneCode: d.Get("time_zone_code").(string),
		ObserveDST:   d.Get("observe_dst").(bool),

		EndAfterHours:   d.Get("end_after_hours").(int),
		EndAfterMinutes: d.Get("end_after_minutes").(int),

		BeforeNotify:        d.Get("notify_before").(bool),
		BeforeNotifyUnit:    d.Get("notify_before_unit").(string),
		BeforeNotifyTime:    d.Get("notify_before_time").(int),
		BeforeNotifyMessage: d.Get("notify_before_message").(string),
		AfterNotify:         d.Get("notify_after").(bool),
		AfterNotifyMessage:  d.Get("notify_after_message").(string),
		RecipientGroupIDs:   stringSet(d, "recipient_group_ids"),
	}
}

func resourceScanScheduleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	id, err := c.CreateScanSchedule(ctx, scanScheduleInputFrom(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)
	return resourceScanScheduleRead(ctx, d, meta)
}

func resourceScanScheduleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	s, err := c.GetScanSchedule(ctx, d.Id())
	if err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	values := map[string]interface{}{
		"title":            s.Title,
		"active":           s.Active.Enabled(),
		"active_state":     int(s.Active),
		"occurrence":       s.Occurrence,
		"frequency_days":   s.FrequencyDays,
		"frequency_weeks":  s.FrequencyWeeks,
		"frequency_months": s.FrequencyMonths,
		"day_of_month":     s.DayOfMonth,
		"week_of_month":    s.WeekOfMonth,
		"start_hour":       s.StartHour,
		"start_minute":     s.StartMinute,
		"time_zone_code":   s.TimeZoneCode,
		"observe_dst":      s.ObserveDST,
	}

	// option_profile_id and option_profile_title are alternatives (ExactlyOneOf).
	// Refresh whichever one the configuration uses; writing both would give the
	// unused attribute a value it never had and diff forever. On import neither is
	// set, so the ID is used, being the stable identifier.
	if d.Get("option_profile_title").(string) != "" {
		values["option_profile_title"] = s.OptionProfileTitle
	} else {
		values["option_profile_id"] = s.OptionProfileID
	}

	return diag.FromErr(setAll(d, values))
}

func resourceScanScheduleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateScanSchedule(ctx, d.Id(), scanScheduleInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceScanScheduleRead(ctx, d, meta)
}

func resourceScanScheduleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteScanSchedule(ctx, d.Id()); err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
