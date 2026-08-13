data "qualys_am_users" "all" {}

output "user_scope_tags" {
  value = {
    for u in data.qualys_am_users.all.users : u.username => [for t in u.scope_tags : t.name]
  }
}
