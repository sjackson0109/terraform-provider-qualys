resource "qualys_asset_tag" "was_targets" {
  name = "WAS Targets"
}

resource "qualys_was_auth_record" "storefront_login" {
  name = "Storefront login"

  form_record {
    sub_type  = "STANDARD"
    login_url = "https://shop.example.com/login"
    username  = "scanner"
    password  = var.storefront_scanner_password
  }
}

resource "qualys_was_dns_override" "storefront_staging" {
  name = "Storefront staging DNS"

  mapping {
    host_name  = "shop.example.com"
    ip_address = "10.20.0.5"
  }
}

resource "qualys_web_application" "storefront" {
  name = "Storefront"
  url  = "https://shop.example.com"

  tag_ids         = [qualys_asset_tag.was_targets.id]
  auth_record_ids = [qualys_was_auth_record.storefront_login.id]

  attributes = {
    "Business Unit" = "E-commerce"
    "Cost Center"   = "CC-4471"
  }

  dns_override_ids        = [qualys_was_dns_override.storefront_staging.id]
  default_dns_override_id = qualys_was_dns_override.storefront_staging.id

  cancel_scans_after_hours = 8

  malware_monitoring   = true
  malware_notification = true
}
