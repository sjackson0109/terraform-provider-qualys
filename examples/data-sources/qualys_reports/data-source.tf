data "qualys_reports" "recent" {
  state = "Finished"
}

output "report_ids" {
  value = [for r in data.qualys_reports.recent.reports : r.id]
}
