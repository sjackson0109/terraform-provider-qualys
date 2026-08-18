resource "qualys_gcp_connector" "dev" {
  name                 = "dev_gcp"
  description          = "Development project connector"
  gcp_credentials_json = file("./service_account.json")

  run_frequency = 240
  activation    = ["VM", "CLOUDVIEW"]
}
