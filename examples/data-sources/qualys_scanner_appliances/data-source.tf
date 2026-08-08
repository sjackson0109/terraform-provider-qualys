# Find appliances, including physical ones the API cannot manage.
data "qualys_scanner_appliances" "all" {}

# Appliances that are online but running outdated signatures or modules.
output "appliances_needing_attention" {
  value = [
    for a in data.qualys_scanner_appliances.all.appliances :
    a.name if a.status == "Online" && !a.up_to_date
  ]
}
