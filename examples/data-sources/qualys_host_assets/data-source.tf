# Identify stale assets: hosts not scanned since a given date. This is the input
# to a review — decide what to purge from the result, rather than purging
# automatically.
data "qualys_host_assets" "stale" {
  no_vm_scan_since = "2026-01-01"
}

output "stale_hosts" {
  value = [
    for h in data.qualys_host_assets.stale.hosts :
    { ip = h.ip, last_scan = h.last_vm_scan, os = h.os }
  ]
}
