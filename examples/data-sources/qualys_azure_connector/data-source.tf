data "qualys_azure_connector" "dev" {
  name = "dev_azure"
}

output "dev_subscription" {
  value = data.qualys_azure_connector.dev.subscription_name
}
