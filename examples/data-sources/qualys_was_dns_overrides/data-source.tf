data "qualys_was_dns_overrides" "staging" {
  name_contains = "Staging"
}

output "staging_dns_override_id" {
  value = one([for o in data.qualys_was_dns_overrides.staging.dns_overrides : o.id])
}
