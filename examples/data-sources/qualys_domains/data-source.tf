# Look up a domain's netblocks to reference in an asset group.
data "qualys_domains" "corp" {
  name_contains = "example.com"
}

output "corp_netblocks" {
  value = flatten([for d in data.qualys_domains.corp.domains : d.netblocks])
}
