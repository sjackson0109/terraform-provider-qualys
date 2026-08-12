resource "qualys_vault" "prod_hashicorp" {
  title = "prod-hashicorp-vault"
  type  = "HashiCorp"

  # Field names here are not validated by the provider: use whatever names
  # your vault type's own Qualys documentation specifies for its connection
  # parameters (endpoint URL, safe/namespace, auth method, and so on).
  parameters = {
    url = "https://vault.example.com:8200"
  }
}

resource "qualys_auth_record_unix" "via_vault" {
  title    = "unix-via-vault"
  username = "qualys-scanner"

  vault_id   = qualys_vault.prod_hashicorp.id
  vault_type = qualys_vault.prod_hashicorp.type

  ips = ["10.30.0.0/24"]
}
