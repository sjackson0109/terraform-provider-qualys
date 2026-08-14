# STANDARD form record: a login page with a known username/password field.
# Sent through the record's generic field list, not as flat elements — see
# the provenance note in docs/resources/was_auth_record.md.
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

# SELENIUM form record: the crawler drives a Selenium IDE script.
resource "qualys_was_auth_record" "spa_login" {
  name = "Single-page app login"

  form_record {
    sub_type = "SELENIUM"
    selenium_script {
      name  = "spaLogin"
      data  = file("${path.module}/selenium/spa-login.side")
      regex = "logged-in"
    }
    selenium_creds = true
    username       = "scanner"
    password       = var.spa_scanner_password
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

# OAuth2 client-credentials grant: no interactive login at all.
resource "qualys_was_auth_record" "api_oauth2" {
  name = "Partner API OAuth2"

  oauth2_record {
    grant_type       = "CLIENT_CREDS"
    access_token_url = "https://auth.example.com/oauth/token"
    client_id        = "scanner-client"
    client_secret    = var.oauth2_client_secret
    scope            = "scan"
  }
}

# Associating a record with a web application is a separate step.
resource "qualys_was_application" "storefront" {
  name = "Storefront"
  url  = "https://shop.example.com"

  auth_record_ids = [qualys_was_auth_record.storefront_login.id]
}
