# A network lets the same IP range exist more than once in a subscription, each
# with its own scanners. Requires the Network Support feature.
resource "qualys_network" "dmz" {
  name = "dmz"
}

# Scanner appliances are assigned to a network from the appliance side, so
# appliance_ids here is read-only. A network with no appliance cannot be scanned.
output "dmz_scanners" {
  value = qualys_network.dmz.appliance_ids
}

# Note: destroying this resource does not delete the network. The Qualys API has
# no delete action for networks; remove it from the Qualys UI if needed.
