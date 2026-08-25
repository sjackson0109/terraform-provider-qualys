data "qualys_was_auth_records" "staging" {
  name_contains = "Staging"
}

output "staging_auth_record_id" {
  value = one([for r in data.qualys_was_auth_records.staging.auth_records : r.id])
}
