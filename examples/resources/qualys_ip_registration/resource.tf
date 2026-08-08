# Register addresses with the subscription and enable them for scanning.
resource "qualys_ip_registration" "web_tier" {
  ips = ["10.20.0.0/24"]

  # Declare the network explicitly rather than relying on the default.
  network_id = qualys_network.dmz.id

  enable_vm = true
  enable_pc = false

  tracking_method = "DNS"
  owner           = "platform-team"
  comment         = "Managed by Terraform"

  # Destroying this resource cannot un-register the addresses — the Qualys API
  # has no such action. Opt in to purging their vulnerability data instead.
  #
  # Purging is destructive and irreversible: it deletes vulnerability and
  # compliance data, tickets and exceptions, and removes the assets from
  # AssetView. Leave it false unless you mean it.
  purge_on_destroy = false
}
