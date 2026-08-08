# A search list names the vulnerabilities an option profile should test for.
resource "qualys_static_search_list" "critical_web" {
  title  = "critical-web-vulns"
  global = true

  qids = [
    "38170", # SSL certificate issues
    "86476",
    "150003",
  ]

  comments = "QIDs tracked for the public web tier"
}

# Referenced by an option profile using custom vulnerability detection.
resource "qualys_vm_option_profile" "web" {
  title                   = "web-tier-scan"
  vulnerability_detection = "custom"
  custom_search_list_ids  = [qualys_static_search_list.critical_web.id]
}
