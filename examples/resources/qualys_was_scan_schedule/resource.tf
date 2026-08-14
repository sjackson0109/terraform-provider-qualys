resource "qualys_was_scan_schedule" "storefront_weekly" {
  name       = "Storefront weekly scan"
  web_app_id = qualys_was_application.storefront.id
  type       = "VULNERABILITY"

  option_profile_id       = qualys_was_option_profile.standard.id
  web_app_auth_record_id  = qualys_was_auth_record.storefront_login.id
  scanner_type            = "EXTERNAL"

  start_date      = "2026-09-01T02:00:00Z"
  time_zone_code  = "Europe/London"
  occurrence_type = "WEEKLY"

  every_n_weeks = 1
  on_days       = ["SUNDAY"]

  cancel_after_hours = 8

  notification {
    active       = true
    delay_amount = 1
    delay_scale  = "DAY"
    recipients   = ["security@example.com"]
    message      = "A Qualys WAS scan is scheduled to start soon."
  }

  send_mail = true
}
