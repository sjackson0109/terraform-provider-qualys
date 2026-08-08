# A nightly scan of an asset group, from Qualys' external scanners.
resource "qualys_scan_schedule" "nightly_web" {
  title             = "nightly-web-scan"
  option_profile_id = qualys_vm_option_profile.web_tier.id

  asset_group_ids     = [qualys_asset_group.prod_web.id]
  use_default_scanner = true

  occurrence     = "daily"
  frequency_days = 1

  start_date     = "01/15/2026"
  start_hour     = 2
  start_minute   = 30
  time_zone_code = "US-NY"
  observe_dst    = true

  # A duration cap, not an end date — the API has no end-date parameter.
  end_after_hours = 6

  notify_before       = true
  notify_before_unit  = "hours"
  notify_before_time  = 1
  recipient_group_ids = [var.ops_distribution_group_id]
}

# A weekly scan targeting assets by tag. Note weekdays takes day *names*, while
# day_of_week (used with week_of_month for monthly schedules) takes a number.
resource "qualys_scan_schedule" "weekly_tagged" {
  title             = "weekly-production-scan"
  option_profile_id = qualys_vm_option_profile.web_tier.id

  tag_include_ids      = [qualys_asset_tag.production.id]
  tag_include_selector = "all"
  use_tagset_scanners  = true

  occurrence      = "weekly"
  frequency_weeks = 1
  weekdays        = ["saturday"]

  start_hour     = 23
  start_minute   = 0
  time_zone_code = "US-NY"
}
