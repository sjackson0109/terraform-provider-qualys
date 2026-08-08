# A static tag: membership is assigned explicitly, not evaluated.
resource "qualys_asset_tag" "environment" {
  name  = "Environment"
  color = "#3B82F6"
}

resource "qualys_asset_tag" "production" {
  name          = "Production"
  parent_tag_id = qualys_asset_tag.environment.id
}

# A dynamic tag: Qualys evaluates rule_text to decide membership, so the members
# are not managed by Terraform.
resource "qualys_asset_tag" "linux_hosts" {
  name      = "Linux Hosts"
  rule_type = "OS_REGEX"
  rule_text = ".*Linux.*"
}

# The same tag is used by VM scan targeting, AssetView and WAS — there is one
# subscription-wide tag tree, not one per module.
