resource "qualys_azure_connector" "dev" {
  name        = "dev_azure"
  description = "Development subscription connector"

  application_id     = "00000000-0000-0000-0000-000000000000"
  directory_id       = "11111111-1111-1111-1111-111111111111"
  subscription_id    = "22222222-2222-2222-2222-222222222222"
  authentication_key = var.azure_client_secret

  run_frequency = 240
  activation    = ["VM", "CLOUDVIEW"]
}
