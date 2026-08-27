terraform {
  required_version = ">= 1.5.0"

  required_providers {
    qualys = {
      source = "sjackson0109/qualys"
    }
  }
}

# Authentication is intentionally environment-driven:
#
#   QUALYS_URL
#   QUALYS_USERNAME
#   QUALYS_PASSWORD
#
# Do not place Qualys credentials in the XLSX, JSON, tfvars, or repository.
provider "qualys" {
  concurrency = var.qualys_concurrency
  max_retries = var.qualys_max_retries
}
