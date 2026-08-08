resource "qualys_asset_tag" "was_targets" {
  name = "WAS Targets"
}

resource "qualys_web_application" "storefront" {
  name = "Storefront"
  url  = "https://shop.example.com"

  tag_ids = [qualys_asset_tag.was_targets.id]
}
