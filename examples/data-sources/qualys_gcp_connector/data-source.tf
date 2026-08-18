data "qualys_gcp_connector" "dev" {
  name = "dev_gcp"
}

output "dev_gcp_project" {
  value = data.qualys_gcp_connector.dev.project_id
}
