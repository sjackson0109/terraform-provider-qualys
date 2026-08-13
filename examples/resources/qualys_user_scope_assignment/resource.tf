resource "qualys_asset_tag" "customer_acme" {
  name = "customer-acme"
}

data "qualys_am_users" "acme_contact" {
  username = "acme-portal-user"
}

resource "qualys_user_scope_assignment" "acme_contact" {
  user_id = data.qualys_am_users.acme_contact.users[0].id
  tag_ids = [qualys_asset_tag.customer_acme.id]
}
