resource "qualys_was_auth_record" "storefront_login" {
  name = "Storefront login"

  form_record {
    field {
      name  = "username"
      value = "scanner"
    }
    field {
      name    = "password"
      value   = var.storefront_scanner_password
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
