data "qualys_was_findings" "storefront_active" {
  web_app_id = qualys_was_application.storefront.id
  status     = ["ACTIVE"]
  is_ignored = false
}

output "storefront_high_severity_count" {
  value = length([
    for f in data.qualys_was_findings.storefront_active.findings : f
    if f.severity >= 4
  ])
}

# Multiple web applications, feeding a downstream remediation dataset.
data "qualys_was_findings" "customer" {
  web_app_ids = [
    qualys_was_application.customer_portal.id,
    qualys_was_application.staff_portal.id,
    qualys_was_application.partner_portal.id,
  ]
  minimum_severity = 2
  is_ignored       = false
}

output "customer_was_findings" {
  value     = data.qualys_was_findings.customer.findings
  sensitive = true
}

# KnowledgeBase-enriched, high-severity, recently detected.
data "qualys_was_findings" "recent_high" {
  minimum_severity          = 4
  last_detected_after       = "2026-08-01T00:00:00Z"
  enrich_with_knowledgebase = true
}
