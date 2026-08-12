data "qualys_tagged_assets" "pci_scope" {
  tag_name = "pci-scope"
}

output "pci_host_ids" {
  value = [for a in data.qualys_tagged_assets.pci_scope.host_assets : a.id]
}
