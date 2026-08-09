resource "qualys_was_scan_schedule" "storefront_weekly" {
  name       = "Storefront weekly scan"
  web_app_id = qualys_web_application.storefront.id
  type       = "VULNERABILITY"

  option_profile_id = qualys_was_option_profile.standard.id

  start_date      = "2026-09-01T02:00:00Z"
  time_zone       = "UTC"
  occurrence_type = "WEEKLY"

  web_app_auth_record_id = qualys_was_auth_record.storefront_login.id

  notification = true
}
