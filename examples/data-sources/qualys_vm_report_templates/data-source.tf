# Look up a scan report template's ID to pass to a report-launch runbook.
# There is no qualys_report_template resource: report launches are jobs, not
# configuration, so this provider only exposes lookups.
data "qualys_vm_report_templates" "scan" {
  template_type = "Scan"
}

output "scan_template_ids" {
  value = [for t in data.qualys_vm_report_templates.scan.report_templates : t.id]
}
