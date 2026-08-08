# Identify hosts not scanned in the last 90 days, the input to a stale-asset
# review. This data source is read-only: acting on the result (e.g. purging
# stale hosts) is a deliberately separate, human-confirmed operation this
# provider does not perform.
data "qualys_host_detections" "stale" {
  vm_scan_date_before = "2025-05-01"
}

output "stale_host_ids" {
  value = [for h in data.qualys_host_detections.stale.hosts : h.id]
}
