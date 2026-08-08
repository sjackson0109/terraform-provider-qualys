# Look up an existing tag to target scans or reports without managing it here.
data "qualys_asset_tags" "production" {
  name = "Production"
}

output "production_tag_id" {
  value = one([for t in data.qualys_asset_tags.production.tags : t.id])
}

# All children of a parent tag.
data "qualys_asset_tags" "environments" {
  parent_tag_id = data.qualys_asset_tags.production.tags[0].id
}
