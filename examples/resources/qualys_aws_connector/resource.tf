resource "qualys_aws_connector" "dev" {
  name        = "dev_aws"
  description = "Development account connector"
  arn         = "arn:aws:iam::123456789012:role/qualys-connector-role"
  external_id = "US1-123456-1234567890123"

  run_frequency = 240
  activation    = ["VM", "CLOUDVIEW"]
}
