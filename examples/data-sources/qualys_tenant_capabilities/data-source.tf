data "qualys_tenant_capabilities" "this" {}

output "portal_version_fields" {
  value = data.qualys_tenant_capabilities.this.raw
}
