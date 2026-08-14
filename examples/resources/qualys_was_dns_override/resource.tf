resource "qualys_was_dns_override" "customer_production" {
  name = "Customer Production DNS"

  mapping {
    host_name  = "portal.customer.com"
    ip_address = "10.100.5.20"
  }
  mapping {
    host_name  = "api.customer.com"
    ip_address = "10.100.5.21"
  }

  comments = ["Created automatically during customer provisioning."]
}

resource "qualys_was_scan_schedule" "storefront_weekly" {
  name       = "Storefront weekly scan"
  web_app_id = qualys_was_application.storefront.id
  type       = "VULNERABILITY"

  dns_override_id = qualys_was_dns_override.customer_production.id

  start_date      = "2026-09-01T02:00:00Z"
  time_zone_code  = "Europe/London"
  occurrence_type = "ONCE"
}
