resource "qualys_auth_record_windows" "corp_domain" {
  title    = "corp-domain"
  username = "svc-qualys"
  password = var.scanner_password # write-only; prefer vault_id

  domain = "corp.example.com"
  # Qualys fixes the domain type at creation, so changing it replaces the record.
  domain_type = "ad_domain"

  ips = ["10.20.0.0/24"]
}
