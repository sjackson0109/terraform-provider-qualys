data "qualys_was_report_templates" "executive" {
  name_contains = "Executive"
  type          = "WAS_WEBAPP_REPORT"
}

resource "qualys_was_report_schedule" "storefront_weekly_exec" {
  name                = "Storefront weekly executive report"
  report_template_id  = one(data.qualys_was_report_templates.executive.templates).id
  web_app_id          = qualys_web_application.storefront.id
  output_format       = "PDF"
  recipients          = ["security@example.com"]

  start_date      = "2026-08-16T06:00:00Z"
  time_zone_code  = "Europe/London"
  occurrence_type = "WEEKLY"

  every_n_weeks = 1
  on_days       = ["MONDAY"]
}
