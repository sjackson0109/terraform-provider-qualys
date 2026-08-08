# An asset group collects scan targets under a name that scans, schedules and
# reports can reference.
resource "qualys_asset_group" "prod_web" {
  title = "prod-web"

  # Declare the intended Qualys network explicitly. Do not rely on the default,
  # and do not infer it from whether the addresses are public or private.
  network_id = var.qualys_network_id

  # CIDR is accepted and converted to the hyphenated ranges the Qualys API
  # expects, so this will not produce a permanent diff.
  ips = [
    "10.20.0.0/24",
    "10.20.4.10-10.20.4.20",
    "10.20.9.7",
  ]

  domains       = ["example.com"]
  appliance_ids = [var.scanner_appliance_id]

  division        = "Platform"
  function        = "Web tier"
  location        = "eu-west-1"
  business_impact = "high"

  comments = "Managed by Terraform"
}

# Member lists are authoritative. Removing an entry above removes it from the
# group in Qualys on the next apply.
