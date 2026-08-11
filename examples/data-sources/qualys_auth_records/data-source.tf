data "qualys_auth_records" "windows" {
  type = "windows"
}

output "windows_record_titles" {
  value = [for r in data.qualys_auth_records.windows.auth_records : r.title]
}
