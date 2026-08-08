# Register a virtual scanner in Qualys and obtain its activation code.
resource "qualys_virtual_scanner" "dmz" {
  name       = "dmz-scanner-01"
  network_id = qualys_network.dmz.id

  polling_interval = 180

  vlan {
    vlan_id = "100"
    ip      = "10.20.100.5"
    netmask = "255.255.255.0"
    name    = "dmz-vlan"
  }
}

# The Qualys side is only half the job. Deploying the appliance VM belongs to
# whichever infrastructure provider hosts it, and consumes the activation code
# exported above.
#
# For example, with vSphere:
#
#   resource "vsphere_virtual_machine" "qualys_scanner" {
#     name = qualys_virtual_scanner.dmz.name
#     # ... sizing, network, datastore ...
#     vapp {
#       properties = {
#         "perscode" = qualys_virtual_scanner.dmz.activation_code
#       }
#     }
#   }
#
# Qualys stops returning the activation code once the appliance activates, so it
# cannot be recovered from the API afterwards.
output "scanner_activation_code" {
  value     = qualys_virtual_scanner.dmz.activation_code
  sensitive = true
}
