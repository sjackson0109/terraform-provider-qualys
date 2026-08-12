data "qualys_vaults" "corp" {
  title = "corp-vault"
}

resource "qualys_auth_record_kubernetes" "cluster01" {
  title    = "cluster01-service-account"
  username = "qualys-scanner"

  vault_id   = data.qualys_vaults.corp.vaults[0].id
  vault_type = data.qualys_vaults.corp.vaults[0].type

  ips = ["10.60.0.0/24"]
}
