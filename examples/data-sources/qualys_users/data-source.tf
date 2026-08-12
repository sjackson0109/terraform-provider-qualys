data "qualys_users" "all" {}

output "manager_logins" {
  value = [for u in data.qualys_users.all.users : u.login if u.role == "Manager"]
}

# Asset groups look like the real access-scoping mechanism (a Manager-role
# user carries no assigned asset groups at all in the source sample, while a
# Scanner-role user does) — see the data source's own description for the
# caveat that this is an inference, not independently confirmed.
output "scanner_asset_group_assignments" {
  value = {
    for u in data.qualys_users.all.users : u.login => u.assigned_asset_groups
    if u.role == "Scanner"
  }
}
