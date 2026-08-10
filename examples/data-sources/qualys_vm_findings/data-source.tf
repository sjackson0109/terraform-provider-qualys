# Individual VM vulnerability detections, not aggregated by host or by QID.
data "qualys_vm_findings" "customer" {
  status = [
    "New",
    "Active",
    "Re-Opened",
  ]
}

output "vm_findings" {
  value     = data.qualys_vm_findings.customer.findings
  sensitive = true
}

# Filtering by asset group: resolved to that group's member IPs internally.
data "qualys_asset_groups" "customer" {
  title_contains = "Customer Production"
}

data "qualys_vm_findings" "customer_production" {
  asset_group_ids  = [data.qualys_asset_groups.customer.groups[0].id]
  minimum_severity = 2
}

output "customer_vm_findings" {
  value     = data.qualys_vm_findings.customer_production.findings
  sensitive = true
}

# Filtering by IP scope.
data "qualys_vm_findings" "subnet" {
  ips = [
    "10.20.0.0/24",
    "10.30.10.5",
  ]
}

# Recent findings, with KnowledgeBase enrichment for downstream reporting.
data "qualys_vm_findings" "recent_enriched" {
  last_found_after          = "2026-08-01T00:00:00Z"
  minimum_severity          = 3
  enrich_with_knowledgebase = true
}
