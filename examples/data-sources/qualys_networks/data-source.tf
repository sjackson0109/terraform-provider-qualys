# Reference a network by name rather than hardcoding its ID.
data "qualys_networks" "dmz" {
  name = "dmz"
}

output "dmz_network_id" {
  value = one([for n in data.qualys_networks.dmz.networks : n.id])
}
