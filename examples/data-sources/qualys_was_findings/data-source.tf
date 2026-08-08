data "qualys_was_findings" "storefront_active" {
  web_app_id = qualys_web_application.storefront.id
  status     = "ACTIVE"
  is_ignored = false
}

output "storefront_high_severity_count" {
  value = length([
    for f in data.qualys_was_findings.storefront_active.findings : f
    if f.severity >= 4
  ])
}
