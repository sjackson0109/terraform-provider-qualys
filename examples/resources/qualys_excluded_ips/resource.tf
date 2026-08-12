resource "qualys_excluded_ips" "maintenance_window" {
  ips     = ["10.40.5.10", "10.40.5.0/28"]
  comment = "Under active maintenance; exclude from scanning"

  # Automatically re-include these IPs in scanning after 14 days, in case
  # the exclusion is forgotten.
  expiry_days = 14
}
