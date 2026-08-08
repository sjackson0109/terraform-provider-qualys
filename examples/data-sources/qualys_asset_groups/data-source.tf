# Reference asset groups managed outside Terraform.
data "qualys_asset_groups" "all" {}

data "qualys_asset_groups" "web" {
  title_contains = "web"
}

output "web_group_ids" {
  value = [for g in data.qualys_asset_groups.web.asset_groups : g.id]
}
