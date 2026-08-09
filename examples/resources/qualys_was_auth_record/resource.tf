# STANDARD form record: a login page with a known username/password field.
resource "qualys_was_auth_record" "storefront_login" {
  name = "Storefront login"

  form_record {
    sub_type  = "STANDARD"
    login_url = "https://shop.example.com/login"
    username  = "scanner"
    password  = var.storefront_scanner_password
    ssl_only  = true
  }
}

# CUSTOM form record: field names aren't fixed, so list them explicitly.
resource "qualys_was_auth_record" "legacy_app_login" {
  name = "Legacy app login"

  form_record {
    sub_type = "CUSTOM"
    field {
      name  = "user"
      value = "scanner"
    }
    field {
      name    = "pass"
      value   = var.legacy_app_scanner_password
      secured = true
    }
  }
}

resource "qualys_was_auth_record" "internal_app_basic_auth" {
  name = "Internal app basic auth"

  server_record {
    username = "scanner"
    password = var.internal_app_scanner_password
    domain   = "CORP"
  }
}

# Associating a record with a web application is a separate step.
resource "qualys_web_application" "storefront" {
  name = "Storefront"
  url  = "https://shop.example.com"

  auth_record_ids = [qualys_was_auth_record.storefront_login.id]
}
