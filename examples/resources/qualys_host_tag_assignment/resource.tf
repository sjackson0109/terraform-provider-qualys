resource "qualys_asset_tag" "pci_scope" {
  name = "pci-scope"
}

resource "qualys_host_tag_assignment" "server01" {
  host_asset_id = "1234567"
  tag_ids       = [qualys_asset_tag.pci_scope.id]
}
