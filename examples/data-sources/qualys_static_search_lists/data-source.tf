data "qualys_static_search_lists" "pci" {
  title_contains = "PCI"
}

output "pci_search_list_id" {
  value = one([for l in data.qualys_static_search_lists.pci.search_lists : l.id])
}
