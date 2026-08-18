data "qualys_was_scans" "running" {
  state = "Running"
}

output "running_scan_refs" {
  value = [for s in data.qualys_was_scans.running.scans : s.ref]
}
