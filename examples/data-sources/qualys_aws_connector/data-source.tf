data "qualys_aws_connector" "dev" {
  name = "dev_aws"
}

output "qualys_trust_account" {
  value = data.qualys_aws_connector.dev.qualys_aws_account_id
}
