# Vaults configured for the subscription. Referencing one from an authentication
# record keeps the secret out of Terraform entirely.
data "qualys_vaults" "all" {}

output "vault_titles" {
  value = [for v in data.qualys_vaults.all.vaults : v.title]
}
