# Preferred: take the credential from a Qualys vault, so the secret never passes
# through Terraform configuration or state.
data "qualys_vaults" "cyberark" {
  title = "corp-cyberark"
}

resource "qualys_auth_record_unix" "linux_fleet" {
  title    = "linux-fleet"
  username = "qualys-scanner"

  vault_id   = one([for v in data.qualys_vaults.cyberark.vaults : v.id])
  vault_type = "CyberArk AIM"

  ips = ["10.20.0.0/24"]
}

# Alternative: supply a password directly. It is write-only — sent to Qualys and
# never read back — so a password changed outside Terraform is not detected.
# Source it from a secret manager rather than hardcoding it.
resource "qualys_auth_record_unix" "legacy" {
  title    = "legacy-hosts"
  username = "qualys-scanner"
  password = var.scanner_password # sensitive

  ips = ["10.30.0.0/24"]
}
