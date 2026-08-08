# An option profile defines how a scan behaves: which ports, how deep, how fast,
# and whether it authenticates.
resource "qualys_vm_option_profile" "web_tier" {
  title  = "web-tier-scan"
  global = true

  # Ports
  scan_tcp_ports            = "standard"
  scan_tcp_ports_additional = "8080,8443"
  scan_udp_ports            = "light"

  # Performance. "normal" suits most subscriptions; "high" needs scanner capacity
  # to match.
  scan_overall_performance = "normal"

  # Detection. "custom" requires custom_search_list_ids: without them Qualys
  # silently falls back to complete detection, so the provider rejects that
  # combination at plan time.
  vulnerability_detection = "complete"

  # Authenticated scanning finds materially more than an unauthenticated scan.
  # This is the list of technologies to attempt, not per-technology switches.
  authentication      = ["Windows", "Unix"]
  test_authentication = true

  basic_host_information_checks = true
}

# Marking a profile default also makes it global in Qualys, so set both.
resource "qualys_vm_option_profile" "default" {
  title      = "org-default-scan"
  is_default = true
  global     = true
}
