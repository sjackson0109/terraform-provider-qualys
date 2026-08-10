data "qualys_was_report_templates" "executive" {
  name_contains = "Executive"
  type          = "WAS_WEBAPP_REPORT"
}

output "executive_template_id" {
  value = one(data.qualys_was_report_templates.executive.templates).id
}
