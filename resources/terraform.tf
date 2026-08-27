locals {
  qualys_config = jsondecode(file(var.config_file))
}

# ---------------------------------------------------------------------------
# Reference resolution: name -> id lookups for objects this root does not
# manage but a workbook row can point at (external_*_name/_title fields).
#
# Mirrors the same pattern already proven in the alltimeuk/qualys MSP
# control-plane workbook (a real consumer of this provider): a `<name>_key`
# field points at another row's `key` in THIS workbook (Terraform resolves it
# to `qualys_<type>.managed[key].id`); an `external_<name>_name` field points
# at an object that already exists in the tenant but is not a row here,
# resolved through a guarded `data "qualys_<type>s"` lookup below. Neither
# ever accepts a raw Qualys id directly.
#
# Each `data` source is guarded by `count` so it only makes a live Qualys API
# call when var.resolve_references_by_name is true AND at least one table
# that could use it is non-empty. A single lookup commonly feeds several
# unrelated resource blocks (external_vault_name alone feeds all 30
# authentication record types), so this is one API call per object type,
# not one per field -- grouped here as a section rather than beside each
# caller for that reason.
# ---------------------------------------------------------------------------

data "qualys_networks" "all" {
  count = (
    var.resolve_references_by_name
    && (
      length(try(local.qualys_config.qualys_virtual_scanner, [])) > 0
      || length(try(local.qualys_config.qualys_asset_group, [])) > 0
      || length(try(local.qualys_config.qualys_ip_registration, [])) > 0
      || length(try(local.qualys_config.qualys_vm_scan_schedule, [])) > 0
      || length(try(local.qualys_config.qualys_excluded_ips, [])) > 0
    )
  ) ? 1 : 0
}

locals {
  network_ids_by_name = (
    length(data.qualys_networks.all) > 0
    ? { for n in data.qualys_networks.all[0].networks : n.name => n.id }
    : {}
  )
}

data "qualys_asset_groups" "all" {
  count = (
    var.resolve_references_by_name
    && (
      length(try(local.qualys_config.qualys_virtual_scanner, [])) > 0
      || length(try(local.qualys_config.qualys_vm_scan_schedule, [])) > 0
    )
  ) ? 1 : 0
}

locals {
  # asset_groups has no server-side exact-title filter (only title_contains),
  # so this fetches every group in the subscription and matches client-side.
  asset_group_ids_by_title = (
    length(data.qualys_asset_groups.all) > 0
    ? { for g in data.qualys_asset_groups.all[0].asset_groups : g.title => g.id }
    : {}
  )
}

data "qualys_scanner_appliances" "all" {
  count = (
    var.resolve_references_by_name
    && length(try(local.qualys_config.qualys_asset_group, [])) > 0
  ) ? 1 : 0
}

locals {
  appliance_ids_by_name = (
    length(data.qualys_scanner_appliances.all) > 0
    ? { for a in data.qualys_scanner_appliances.all[0].appliances : a.name => a.id }
    : {}
  )
}

data "qualys_asset_tags" "all" {
  count = (
    var.resolve_references_by_name
    && (
      length(try(local.qualys_config.qualys_asset_tag, [])) > 0
      || length(try(local.qualys_config.qualys_vm_scan_schedule, [])) > 0
      || length(try(local.qualys_config.qualys_user_scope_assignment, [])) > 0
      || length(try(local.qualys_config.qualys_host_tag_assignment, [])) > 0
      || length(try(local.qualys_config.qualys_was_application, [])) > 0
      || length(try(local.qualys_config.qualys_was_auth_record, [])) > 0
      || length(try(local.qualys_config.qualys_was_dns_override, [])) > 0
      || length(try(local.qualys_config.qualys_gcp_connector, [])) > 0
      || length(try(local.qualys_config.qualys_aws_connector, [])) > 0
      || length(try(local.qualys_config.qualys_azure_connector, [])) > 0
    )
  ) ? 1 : 0
}

locals {
  # Distinct from local.asset_tag_ids (below, by logical key, for tags this
  # root manages): this resolves external_*_tag_name fields by the tag's
  # actual Qualys name, for tags this root does not manage.
  external_tag_ids_by_name = (
    length(data.qualys_asset_tags.all) > 0
    ? { for t in data.qualys_asset_tags.all[0].tags : t.name => t.id }
    : {}
  )
}

data "qualys_vaults" "all" {
  count = (
    var.resolve_references_by_name
    && (
      length(try(local.qualys_config.qualys_auth_record_windows, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_unix, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_network_ssh, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_oracle, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_oracle_listener, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_snmp, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_mssql, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_mysql, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_postgresql, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_sybase, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_ibm_db2, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_informix, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_mariadb, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_mongodb, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_neo4j, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_sapiq, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_sap_hana, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_vmware, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_vcenter, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_http, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_apache, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_nginx, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_ms_iis, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_ibm_websphere, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_tomcat, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_oracle_weblogic, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_jboss, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_kubernetes, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_docker, [])) > 0
      || length(try(local.qualys_config.qualys_auth_record_palo_alto, [])) > 0
    )
  ) ? 1 : 0
}

locals {
  # Keyed to the full vault object, not just its id: vault_type is a fixed
  # property of the vault itself, so knowing which vault also gives us that,
  # rather than requiring a second, easy-to-mismatch manual vault_type field.
  vault_by_title = (
    length(data.qualys_vaults.all) > 0
    ? { for v in data.qualys_vaults.all[0].vaults : v.title => v }
    : {}
  )
}

data "qualys_was_applications" "all" {
  count = (
    var.resolve_references_by_name
    && (
      length(try(local.qualys_config.qualys_was_scan_schedule, [])) > 0
      || length(try(local.qualys_config.qualys_was_report_schedule, [])) > 0
    )
  ) ? 1 : 0
}

locals {
  web_app_ids_by_name = (
    length(data.qualys_was_applications.all) > 0
    ? { for a in data.qualys_was_applications.all[0].web_applications : a.name => a.id }
    : {}
  )
}

data "qualys_was_report_templates" "all" {
  count = (
    var.resolve_references_by_name
    && length(try(local.qualys_config.qualys_was_report_schedule, [])) > 0
  ) ? 1 : 0
}

locals {
  was_report_template_ids_by_name = (
    length(data.qualys_was_report_templates.all) > 0
    ? { for t in data.qualys_was_report_templates.all[0].templates : t.name => t.id }
    : {}
  )
}

data "qualys_was_auth_records" "all" {
  count = (
    var.resolve_references_by_name
    && (
      length(try(local.qualys_config.qualys_was_application, [])) > 0
      || length(try(local.qualys_config.qualys_was_scan_schedule, [])) > 0
    )
  ) ? 1 : 0
}

locals {
  was_auth_record_ids_by_name = (
    length(data.qualys_was_auth_records.all) > 0
    ? { for r in data.qualys_was_auth_records.all[0].auth_records : r.name => r.id }
    : {}
  )
}

data "qualys_was_dns_overrides" "all" {
  count = (
    var.resolve_references_by_name
    && (
      length(try(local.qualys_config.qualys_was_application, [])) > 0
      || length(try(local.qualys_config.qualys_was_scan_schedule, [])) > 0
    )
  ) ? 1 : 0
}

locals {
  was_dns_override_ids_by_name = (
    length(data.qualys_was_dns_overrides.all) > 0
    ? { for o in data.qualys_was_dns_overrides.all[0].dns_overrides : o.name => o.id }
    : {}
  )
}

data "qualys_static_search_lists" "all" {
  # No server-side exact-title filter (only title_contains — same
  # constraint as qualys_vm_option_profiles), so this fetches every static
  # list in the subscription and matches client-side.
  count = (
    var.resolve_references_by_name
    && length(try(local.qualys_config.qualys_vm_option_profile, [])) > 0
  ) ? 1 : 0
}

locals {
  static_search_list_ids_by_title = (
    length(data.qualys_static_search_lists.all) > 0
    ? { for l in data.qualys_static_search_lists.all[0].search_lists : l.title => l.id }
    : {}
  )
}

data "qualys_vm_option_profiles" "all" {
  count = (
    var.resolve_references_by_name
    && (
      length(try(local.qualys_config.qualys_vm_option_profile, [])) > 0
      || length(try(local.qualys_config.qualys_vm_scan_schedule, [])) > 0
    )
  ) ? 1 : 0
}

locals {
  vm_option_profile_ids_by_title = (
    length(data.qualys_vm_option_profiles.all) > 0
    ? { for p in data.qualys_vm_option_profiles.all[0].option_profiles : p.title => p.id }
    : {}
  )
}

data "qualys_was_option_profiles" "all" {
  count = (
    var.resolve_references_by_name
    && (
      length(try(local.qualys_config.qualys_was_option_profile, [])) > 0
      || length(try(local.qualys_config.qualys_was_scan_schedule, [])) > 0
    )
  ) ? 1 : 0
}

locals {
  was_option_profile_ids_by_name = (
    length(data.qualys_was_option_profiles.all) > 0
    ? { for p in data.qualys_was_option_profiles.all[0].option_profiles : p.name => p.id }
    : {}
  )
}

data "qualys_am_users" "all" {
  count = (
    var.resolve_references_by_name
    && length(try(local.qualys_config.qualys_user_scope_assignment, [])) > 0
  ) ? 1 : 0
}

locals {
  user_ids_by_username = (
    length(data.qualys_am_users.all) > 0
    ? { for u in data.qualys_am_users.all[0].users : u.username => u.id }
    : {}
  )
  # There is no dedicated "list roles" endpoint: role name -> id pairs are
  # read off of qualys_am_users.all[0].users[*].roles instead, the same
  # {id, name} shape the Administration RBAC API already returns per user.
  # This only surfaces a role if at least one user in the tenant already has
  # it assigned -- a real limitation, not a bug -- but needs no new provider
  # capability and no per-tenant hardcoded id map.
  role_ids_by_name = (
    length(data.qualys_am_users.all) > 0
    ? merge([
      for u in data.qualys_am_users.all[0].users : { for r in u.roles : r.name => r.id }
    ]...)
    : {}
  )
}

# ---------------------------------------------------------------------------
# Managed resources
# ---------------------------------------------------------------------------

resource "qualys_gcp_connector" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_gcp_connector, []) : item.key => item if try(item.enabled, true) }

  name                   = try(each.value.name, null)
  description            = try(each.value.description, null)
  gcp_credentials_json   = try(each.value.gcp_credentials_json, null)
  project_id             = try(each.value.project_id, null)
  run_frequency          = try(each.value.run_frequency, null)
  disabled               = try(each.value.disabled, null)
  is_remediation_enabled = try(each.value.is_remediation_enabled, null)
  activation             = try(each.value.activation, null)
  default_tag_ids = try(coalescelist(concat(
    [for k in try(each.value.default_tag_keys, []) : local.asset_tag_ids[k]],
    [for n in try(each.value.external_default_tag_names, []) : local.external_tag_ids_by_name[n]]
  )), null)
}

resource "qualys_aws_connector" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_aws_connector, []) : item.key => item if try(item.enabled, true) }

  name                   = try(each.value.name, null)
  description            = try(each.value.description, null)
  arn                    = try(each.value.arn, null)
  external_id            = try(each.value.external_id, null)
  all_regions            = try(each.value.all_regions, null)
  is_gov_cloud           = try(each.value.is_gov_cloud, null)
  run_frequency          = try(each.value.run_frequency, null)
  disabled               = try(each.value.disabled, null)
  is_remediation_enabled = try(each.value.is_remediation_enabled, null)
  activation             = try(each.value.activation, null)
  is_portal_connector    = try(each.value.is_portal_connector, null)
  default_tag_ids = try(coalescelist(concat(
    [for k in try(each.value.default_tag_keys, []) : local.asset_tag_ids[k]],
    [for n in try(each.value.external_default_tag_names, []) : local.external_tag_ids_by_name[n]]
  )), null)
}

resource "qualys_azure_connector" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_azure_connector, []) : item.key => item if try(item.enabled, true) }

  name                   = try(each.value.name, null)
  description            = try(each.value.description, null)
  application_id         = try(each.value.application_id, null)
  directory_id           = try(each.value.directory_id, null)
  subscription_id        = try(each.value.subscription_id, null)
  authentication_key     = try(each.value.authentication_key, null)
  run_frequency          = try(each.value.run_frequency, null)
  disabled               = try(each.value.disabled, null)
  is_remediation_enabled = try(each.value.is_remediation_enabled, null)
  activation             = try(each.value.activation, null)
  default_tag_ids = try(coalescelist(concat(
    [for k in try(each.value.default_tag_keys, []) : local.asset_tag_ids[k]],
    [for n in try(each.value.external_default_tag_names, []) : local.external_tag_ids_by_name[n]]
  )), null)
}

resource "qualys_asset_group" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_asset_group, []) : item.key => item if try(item.enabled, true) }

  title           = try(each.value.title, null)
  business_impact = try(each.value.business_impact, null)
  comments        = try(each.value.comments, null)
  cvss_enviro_ar  = try(each.value.cvss_enviro_ar, null)
  cvss_enviro_cdp = try(each.value.cvss_enviro_cdp, null)
  cvss_enviro_cr  = try(each.value.cvss_enviro_cr, null)
  cvss_enviro_ir  = try(each.value.cvss_enviro_ir, null)
  cvss_enviro_td  = try(each.value.cvss_enviro_td, null)
  division        = try(each.value.division, null)
  dns_names       = try(each.value.dns_names, null)
  domains         = try(each.value.domains, null)
  function        = try(each.value.function, null)
  ips             = try(each.value.ips, null)
  location        = try(each.value.location, null)
  netbios_names   = try(each.value.netbios_names, null)

  network_id = (
    try(each.value.network_key, null) != null
    ? qualys_network.managed[each.value.network_key].id
    : try(local.network_ids_by_name[each.value.external_network_name], null)
  )

  appliance_ids = try(coalescelist(concat(
    [for k in try(each.value.appliance_keys, []) : qualys_virtual_scanner.managed[k].id],
    [for n in try(each.value.external_appliance_names, []) : local.appliance_ids_by_name[n]]
  )), null)

  default_appliance_id = (
    try(each.value.default_appliance_key, null) != null
    ? qualys_virtual_scanner.managed[each.value.default_appliance_key].id
    : try(local.appliance_ids_by_name[each.value.external_default_appliance_name], null)
  )
}

resource "qualys_asset_tag" "managed" {
  for_each = {
    for item in try(local.qualys_config.qualys_asset_tag, []) :
    item.key => item
    if try(item.enabled, true) && try(item.parent_tag_key, null) == null
  }

  name      = try(each.value.name, null)
  color     = try(each.value.color, null)
  rule_text = try(each.value.rule_text, null)
  rule_type = try(each.value.rule_type, null)
  parent_tag_id = (
    try(each.value.external_parent_tag_name, null) != null
    ? local.external_tag_ids_by_name[each.value.external_parent_tag_name]
    : null
  )
}

# A managed resource cannot reference itself, so a tag whose parent is
# ANOTHER managed row (parent_tag_key set) is a second resource block —
# mirrors qualys_asset_tag.managed above exactly, plus the parent lookup.
# Managed nesting is one level deep this way; deeper trees need
# external_parent_tag_name.
resource "qualys_asset_tag" "managed_child" {
  for_each = {
    for item in try(local.qualys_config.qualys_asset_tag, []) :
    item.key => item
    if try(item.enabled, true) && try(item.parent_tag_key, null) != null
  }

  name          = try(each.value.name, null)
  color         = try(each.value.color, null)
  rule_text     = try(each.value.rule_text, null)
  rule_type     = try(each.value.rule_type, null)
  parent_tag_id = qualys_asset_tag.managed[each.value.parent_tag_key].id
}

locals {
  # Every managed tag id by logical key, whatever its position in the tree.
  asset_tag_ids = merge(
    { for k, r in qualys_asset_tag.managed : k => r.id },
    { for k, r in qualys_asset_tag.managed_child : k => r.id },
  )
}

resource "qualys_static_search_list" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_static_search_list, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  qids     = try(each.value.qids, null)
  comments = try(each.value.comments, null)
  global   = try(each.value.global, null)
}

resource "qualys_vm_option_profile" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_vm_option_profile, []) : item.key => item if try(item.enabled, true) }

  title                            = try(each.value.title, null)
  additional_certificate_detection = try(each.value.additional_certificate_detection, null)
  all_qrdi_checks                  = try(each.value.all_qrdi_checks, null)
  authentication                   = try(each.value.authentication, null)
  basic_host_information_checks    = try(each.value.basic_host_information_checks, null)
  close_vuln_on_dead_hosts         = try(each.value.close_vuln_on_dead_hosts, null)
  enable_dissolvable_agent         = try(each.value.enable_dissolvable_agent, null)
  global                           = try(each.value.global, null)
  include_system_auth              = try(each.value.include_system_auth, null)
  is_default                       = try(each.value.is_default, null)
  load_balancer_detection          = try(each.value.load_balancer_detection, null)
  offline_scanner                  = try(each.value.offline_scanner, null)
  oval_checks                      = try(each.value.oval_checks, null)
  owner                            = try(each.value.owner, null)
  password_brute_forcing_system    = try(each.value.password_brute_forcing_system, null)
  purge_host_data                  = try(each.value.purge_host_data, null)
  scan_dead_hosts                  = try(each.value.scan_dead_hosts, null)
  scan_external_scanners           = try(each.value.scan_external_scanners, null)
  scan_http_process                = try(each.value.scan_http_process, null)
  scan_intensity                   = try(each.value.scan_intensity, null)
  scan_overall_performance         = try(each.value.scan_overall_performance, null)
  scan_packet_delay                = try(each.value.scan_packet_delay, null)
  scan_parallel_scaling            = try(each.value.scan_parallel_scaling, null)
  scan_scanner_appliances          = try(each.value.scan_scanner_appliances, null)
  scan_tcp_ports                   = try(each.value.scan_tcp_ports, null)
  scan_tcp_ports_additional        = try(each.value.scan_tcp_ports_additional, null)
  scan_total_process               = try(each.value.scan_total_process, null)
  scan_udp_ports                   = try(each.value.scan_udp_ports, null)
  scan_udp_ports_additional        = try(each.value.scan_udp_ports_additional, null)
  test_authentication              = try(each.value.test_authentication, null)
  three_way_handshake              = try(each.value.three_way_handshake, null)
  use_system_auth_on_duplicate     = try(each.value.use_system_auth_on_duplicate, null)
  use_user_auth_on_duplicate       = try(each.value.use_user_auth_on_duplicate, null)
  vulnerability_detection          = try(each.value.vulnerability_detection, null)

  custom_search_list_ids = try(coalescelist(concat(
    [for k in try(each.value.custom_search_list_keys, []) : qualys_static_search_list.managed[k].id],
    [for n in try(each.value.external_custom_search_list_titles, []) : local.static_search_list_ids_by_title[n]]
  )), null)
  exclude_search_list_ids = try(coalescelist(concat(
    [for k in try(each.value.exclude_search_list_keys, []) : qualys_static_search_list.managed[k].id],
    [for n in try(each.value.external_exclude_search_list_titles, []) : local.static_search_list_ids_by_title[n]]
  )), null)
}

resource "qualys_network" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_network, []) : item.key => item if try(item.enabled, true) }

  name = try(each.value.name, null)
}

resource "qualys_ip_registration" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_ip_registration, []) : item.key => item if try(item.enabled, true) }

  ips               = try(each.value.ips, null)
  asset_group_title = try(each.value.asset_group_title, null)
  comment           = try(each.value.comment, null)
  enable_pc         = try(each.value.enable_pc, null)
  enable_vm         = try(each.value.enable_vm, null)
  owner             = try(each.value.owner, null)
  purge_on_destroy  = try(each.value.purge_on_destroy, null)
  tracking_method   = try(each.value.tracking_method, null)
  ud1               = try(each.value.ud1, null)
  ud2               = try(each.value.ud2, null)
  ud3               = try(each.value.ud3, null)

  network_id = (
    try(each.value.network_key, null) != null
    ? qualys_network.managed[each.value.network_key].id
    : try(local.network_ids_by_name[each.value.external_network_name], null)
  )
}

resource "qualys_vm_scan_schedule" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_vm_scan_schedule, []) : item.key => item if try(item.enabled, true) }

  title                 = try(each.value.title, null)
  occurrence            = try(each.value.occurrence, null)
  start_date            = try(each.value.start_date, null)
  start_hour            = try(each.value.start_hour, null)
  start_minute          = try(each.value.start_minute, null)
  time_zone_code        = try(each.value.time_zone_code, null)
  active                = try(each.value.active, null)
  day_of_month          = try(each.value.day_of_month, null)
  day_of_week           = try(each.value.day_of_week, null)
  end_after_hours       = try(each.value.end_after_hours, null)
  end_after_minutes     = try(each.value.end_after_minutes, null)
  exclude_ips           = try(each.value.exclude_ips, null)
  frequency_days        = try(each.value.frequency_days, null)
  frequency_months      = try(each.value.frequency_months, null)
  frequency_weeks       = try(each.value.frequency_weeks, null)
  ips                   = try(each.value.ips, null)
  notify_after          = try(each.value.notify_after, null)
  notify_after_message  = try(each.value.notify_after_message, null)
  notify_before         = try(each.value.notify_before, null)
  notify_before_message = try(each.value.notify_before_message, null)
  notify_before_time    = try(each.value.notify_before_time, null)
  notify_before_unit    = try(each.value.notify_before_unit, null)
  option_profile_title  = try(each.value.option_profile_title, null)
  observe_dst           = try(each.value.observe_dst, null)
  # Distribution groups have no known list/search API (unlike everything
  # else here) -- stays a raw id until that changes.
  recipient_group_ids      = try(each.value.recipient_group_ids, null)
  recurrence               = try(each.value.recurrence, null)
  scanner_names            = try(each.value.scanner_names, null)
  tag_include_selector     = try(each.value.tag_include_selector, null)
  use_asset_group_scanners = try(each.value.use_asset_group_scanners, null)
  use_default_scanner      = try(each.value.use_default_scanner, null)
  use_tagset_scanners      = try(each.value.use_tagset_scanners, null)
  weekdays                 = try(each.value.weekdays, null)
  week_of_month            = try(each.value.week_of_month, null)

  network_id = (
    try(each.value.network_key, null) != null
    ? qualys_network.managed[each.value.network_key].id
    : try(local.network_ids_by_name[each.value.external_network_name], null)
  )

  option_profile_id = (
    try(each.value.option_profile_key, null) != null
    ? qualys_vm_option_profile.managed[each.value.option_profile_key].id
    : try(local.vm_option_profile_ids_by_title[each.value.external_option_profile_name], null)
  )

  asset_group_ids = try(coalescelist(concat(
    [for k in try(each.value.asset_group_keys, []) : qualys_asset_group.managed[k].id],
    [for n in try(each.value.external_asset_group_titles, []) : local.asset_group_ids_by_title[n]]
  )), null)

  tag_include_ids = try(coalescelist(concat(
    [for k in try(each.value.tag_include_keys, []) : local.asset_tag_ids[k]],
    [for n in try(each.value.external_tag_include_names, []) : local.external_tag_ids_by_name[n]]
  )), null)
  tag_exclude_ids = try(coalescelist(concat(
    [for k in try(each.value.tag_exclude_keys, []) : local.asset_tag_ids[k]],
    [for n in try(each.value.external_tag_exclude_names, []) : local.external_tag_ids_by_name[n]]
  )), null)
}

resource "qualys_virtual_scanner" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_virtual_scanner, []) : item.key => item if try(item.enabled, true) }

  name                    = try(each.value.name, null)
  comments                = try(each.value.comments, null)
  polling_interval        = try(each.value.polling_interval, null)
  wait_for_online         = try(each.value.wait_for_online, null)
  wait_for_online_timeout = try(each.value.wait_for_online_timeout, null)

  network_id = (
    try(each.value.network_key, null) != null
    ? qualys_network.managed[each.value.network_key].id
    : try(local.network_ids_by_name[each.value.external_network_name], null)
  )

  # External reference only: a managed asset-group reference here would
  # create a dependency cycle with qualys_asset_group.appliance_keys. A
  # data-source lookup by name carries no such resource dependency edge,
  # so external_asset_group_name is still fine here.
  asset_group_id = try(local.asset_group_ids_by_title[each.value.external_asset_group_name], null)

  dynamic "vlan" {
    for_each = try(each.value.vlan, [])
    content {
      vlan_id = vlan.value.vlan_id
      ip      = vlan.value.ip
      netmask = vlan.value.netmask
      name    = vlan.value.name
    }
  }
  dynamic "static_route" {
    for_each = try(each.value.static_route, [])
    content {
      gateway = static_route.value.gateway
      ip      = static_route.value.ip
      name    = static_route.value.name
      netmask = static_route.value.netmask
    }
  }
}

resource "qualys_vault" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_vault, []) : item.key => item if try(item.enabled, true) }

  title      = try(each.value.title, null)
  type       = try(each.value.type, null)
  parameters = try(each.value.parameters, null)
}

resource "qualys_excluded_ips" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_excluded_ips, []) : item.key => item if try(item.enabled, true) }

  ips               = try(each.value.ips, null)
  asset_group_names = try(each.value.asset_group_names, null)
  comment           = try(each.value.comment, null)
  expiry_days       = try(each.value.expiry_days, null)

  network_id = (
    try(each.value.network_key, null) != null
    ? qualys_network.managed[each.value.network_key].id
    : try(local.network_ids_by_name[each.value.external_network_name], null)
  )
}

resource "qualys_host_tag_assignment" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_host_tag_assignment, []) : item.key => item if try(item.enabled, true) }

  # AssetView/CSAM host asset id -- external by nature, no name-based lookup.
  host_asset_id = try(each.value.host_asset_id, null)

  tag_ids = concat(
    [for k in try(each.value.tag_keys, []) : local.asset_tag_ids[k]],
    [for n in try(each.value.external_tag_names, []) : local.external_tag_ids_by_name[n]]
  )
}

resource "qualys_user_scope_assignment" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_user_scope_assignment, []) : item.key => item if try(item.enabled, true) }

  # Users cannot be created through the Qualys API; user and role
  # references are always external, resolved by username/role name via
  # data.qualys_am_users above rather than a workbook-managed key.
  user_id = try(local.user_ids_by_username[each.value.external_user_name], null)

  role_ids = try(
    [for n in each.value.external_role_names : local.role_ids_by_name[n]],
    null
  )

  tag_ids = try(coalescelist(concat(
    [for k in try(each.value.tag_keys, []) : local.asset_tag_ids[k]],
    [for n in try(each.value.external_tag_names, []) : local.external_tag_ids_by_name[n]]
  )), null)
}

resource "qualys_was_application" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_was_application, []) : item.key => item if try(item.enabled, true) }

  name                     = try(each.value.name, null)
  url                      = try(each.value.url, null)
  attributes               = try(each.value.attributes, null)
  cancel_scans_at          = try(each.value.cancel_scans_at, null)
  cancel_scans_after_hours = try(each.value.cancel_scans_after_hours, null)
  malware_monitoring       = try(each.value.malware_monitoring, null)
  malware_notification     = try(each.value.malware_notification, null)

  tag_ids = try(coalescelist(concat(
    [for k in try(each.value.tag_keys, []) : local.asset_tag_ids[k]],
    [for n in try(each.value.external_tag_names, []) : local.external_tag_ids_by_name[n]]
  )), null)

  auth_record_ids = try(coalescelist(concat(
    [for k in try(each.value.auth_record_keys, []) : qualys_was_auth_record.managed[k].id],
    [for n in try(each.value.external_auth_record_names, []) : local.was_auth_record_ids_by_name[n]]
  )), null)

  dns_override_ids = try(coalescelist(concat(
    [for k in try(each.value.dns_override_keys, []) : qualys_was_dns_override.managed[k].id],
    [for n in try(each.value.external_dns_override_names, []) : local.was_dns_override_ids_by_name[n]]
  )), null)

  default_dns_override_id = (
    try(each.value.default_dns_override_key, null) != null
    ? qualys_was_dns_override.managed[each.value.default_dns_override_key].id
    : try(local.was_dns_override_ids_by_name[each.value.external_default_dns_override_name], null)
  )

  dynamic "swagger_file" {
    for_each = try(each.value.swagger_file, null) == null ? [] : [each.value.swagger_file]
    content {
      name           = try(swagger_file.value.name, null)
      content_base64 = try(swagger_file.value.content_base64, null)
    }
  }
  dynamic "postman_collection" {
    for_each = try(each.value.postman_collection, null) == null ? [] : [each.value.postman_collection]
    content {
      dynamic "collection" {
        for_each = try(postman_collection.value.collection, null) == null ? [] : [postman_collection.value.collection]
        content {
          name           = try(collection.value.name, null)
          content_base64 = try(collection.value.content_base64, null)
        }
      }
      dynamic "environment_variable" {
        for_each = try(postman_collection.value.environment_variable, null) == null ? [] : [postman_collection.value.environment_variable]
        content {
          name           = try(environment_variable.value.name, null)
          content_base64 = try(environment_variable.value.content_base64, null)
        }
      }
      dynamic "global_variable" {
        for_each = try(postman_collection.value.global_variable, null) == null ? [] : [postman_collection.value.global_variable]
        content {
          name           = try(global_variable.value.name, null)
          content_base64 = try(global_variable.value.content_base64, null)
        }
      }
    }
  }
}

resource "qualys_was_option_profile" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_was_option_profile, []) : item.key => item if try(item.enabled, true) }

  name                           = try(each.value.name, null)
  comments                       = try(each.value.comments, null)
  bruteforce_option              = try(each.value.bruteforce_option, null)
  detect_credit_card_numbers     = try(each.value.detect_credit_card_numbers, null)
  detect_social_security_numbers = try(each.value.detect_social_security_numbers, null)
  max_crawl_requests             = try(each.value.max_crawl_requests, null)
  performance                    = try(each.value.performance, null)
  timeout_error_threshold        = try(each.value.timeout_error_threshold, null)
  unexpected_error_threshold     = try(each.value.unexpected_error_threshold, null)
}

resource "qualys_was_auth_record" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_was_auth_record, []) : item.key => item if try(item.enabled, true) }

  name     = try(each.value.name, null)
  comments = try(each.value.comments, null)

  tag_ids = try(coalescelist(concat(
    [for k in try(each.value.tag_keys, []) : local.asset_tag_ids[k]],
    [for n in try(each.value.external_tag_names, []) : local.external_tag_ids_by_name[n]]
  )), null)

  dynamic "server_record" {
    for_each = try(each.value.server_record, null) == null ? [] : [each.value.server_record]
    content {
      username = try(server_record.value.username, null)
      password = try(server_record.value.password, null)
      domain   = try(server_record.value.domain, null)
      sub_type = try(server_record.value.sub_type, null)
    }
  }
  dynamic "form_record" {
    for_each = try(each.value.form_record, null) == null ? [] : [each.value.form_record]
    content {
      sub_type       = try(form_record.value.sub_type, null)
      login_url      = try(form_record.value.login_url, null)
      username       = try(form_record.value.username, null)
      password       = try(form_record.value.password, null)
      ssl_only       = try(form_record.value.ssl_only, null)
      auth_vault     = try(form_record.value.auth_vault, null)
      selenium_creds = try(form_record.value.selenium_creds, null)
      dynamic "field" {
        for_each = try(form_record.value.field, [])
        content {
          name    = field.value.name
          value   = field.value.value
          secured = try(field.value.secured, null)
        }
      }
      dynamic "selenium_script" {
        for_each = try(form_record.value.selenium_script, null) == null ? [] : [form_record.value.selenium_script]
        content {
          name  = selenium_script.value.name
          data  = selenium_script.value.data
          regex = try(selenium_script.value.regex, null)
        }
      }
    }
  }
  dynamic "oauth2_record" {
    for_each = try(each.value.oauth2_record, null) == null ? [] : [each.value.oauth2_record]
    content {
      grant_type                       = oauth2_record.value.grant_type
      access_token_url                 = try(oauth2_record.value.access_token_url, null)
      client_id                        = try(oauth2_record.value.client_id, null)
      client_secret                    = try(oauth2_record.value.client_secret, null)
      scope                            = try(oauth2_record.value.scope, null)
      redirect_url                     = try(oauth2_record.value.redirect_url, null)
      username                         = try(oauth2_record.value.username, null)
      password                         = try(oauth2_record.value.password, null)
      access_token_expired_msg_pattern = try(oauth2_record.value.access_token_expired_msg_pattern, null)
      dynamic "selenium_script" {
        for_each = try(oauth2_record.value.selenium_script, null) == null ? [] : [oauth2_record.value.selenium_script]
        content {
          name  = selenium_script.value.name
          data  = selenium_script.value.data
          regex = try(selenium_script.value.regex, null)
        }
      }
    }
  }
}

resource "qualys_was_scan_schedule" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_was_scan_schedule, []) : item.key => item if try(item.enabled, true) }

  name                            = try(each.value.name, null)
  type                            = try(each.value.type, null)
  start_date                      = try(each.value.start_date, null)
  time_zone_code                  = try(each.value.time_zone_code, null)
  occurrence_type                 = try(each.value.occurrence_type, null)
  active                          = try(each.value.active, null)
  web_app_auth_record_use_default = try(each.value.web_app_auth_record_use_default, null)
  scanner_type                    = try(each.value.scanner_type, null)
  scanner_friendly_name           = try(each.value.scanner_friendly_name, null)
  cancel_option                   = try(each.value.cancel_option, null)
  cancel_after_hours              = try(each.value.cancel_after_hours, null)
  every_n_days                    = try(each.value.every_n_days, null)
  every_n_weeks                   = try(each.value.every_n_weeks, null)
  on_days                         = try(each.value.on_days, null)
  occurrence_count                = try(each.value.occurrence_count, null)
  send_mail                       = try(each.value.send_mail, null)
  send_one_mail                   = try(each.value.send_one_mail, null)
  send_mail_from_address_option   = try(each.value.send_mail_from_address_option, null)
  # WAS proxies are external/read-only objects with no known way to list or
  # search them via API (unlike every other reference field here) -- stays
  # a raw id until that changes.
  proxy_id = try(each.value.proxy_id, null)

  web_app_id = (
    try(each.value.web_app_key, null) != null
    ? qualys_was_application.managed[each.value.web_app_key].id
    : try(local.web_app_ids_by_name[each.value.external_web_app_name], null)
  )

  option_profile_id = (
    try(each.value.option_profile_key, null) != null
    ? qualys_was_option_profile.managed[each.value.option_profile_key].id
    : try(local.was_option_profile_ids_by_name[each.value.external_option_profile_name], null)
  )

  web_app_auth_record_id = (
    try(each.value.auth_record_key, null) != null
    ? qualys_was_auth_record.managed[each.value.auth_record_key].id
    : try(local.was_auth_record_ids_by_name[each.value.external_auth_record_name], null)
  )

  dns_override_id = (
    try(each.value.dns_override_key, null) != null
    ? qualys_was_dns_override.managed[each.value.dns_override_key].id
    : try(local.was_dns_override_ids_by_name[each.value.external_dns_override_name], null)
  )

  dynamic "notification" {
    for_each = try(each.value.notification, null) == null ? [] : [each.value.notification]
    content {
      active       = try(notification.value.active, null)
      reschedule   = try(notification.value.reschedule, null)
      delay_amount = try(notification.value.delay_amount, null)
      delay_scale  = try(notification.value.delay_scale, null)
      recipients   = try(notification.value.recipients, null)
      message      = try(notification.value.message, null)
    }
  }
}

resource "qualys_was_report_schedule" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_was_report_schedule, []) : item.key => item if try(item.enabled, true) }

  name             = try(each.value.name, null)
  output_format    = try(each.value.output_format, null)
  recipients       = try(each.value.recipients, null)
  start_date       = try(each.value.start_date, null)
  time_zone_code   = try(each.value.time_zone_code, null)
  occurrence_type  = try(each.value.occurrence_type, null)
  active           = try(each.value.active, null)
  every_n_days     = try(each.value.every_n_days, null)
  every_n_weeks    = try(each.value.every_n_weeks, null)
  on_days          = try(each.value.on_days, null)
  occurrence_count = try(each.value.occurrence_count, null)
  day_of_month     = try(each.value.day_of_month, null)
  every_n_months   = try(each.value.every_n_months, null)

  # Report templates are read-only through the API (created in the Qualys
  # UI), so the template reference is always resolved by name via the
  # lookup above, never a workbook-managed key.
  report_template_id = local.was_report_template_ids_by_name[each.value.external_report_template_name]

  web_app_id = (
    try(each.value.web_app_key, null) != null
    ? qualys_was_application.managed[each.value.web_app_key].id
    : try(local.web_app_ids_by_name[each.value.external_web_app_name], null)
  )
}

resource "qualys_was_finding_ignore" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_was_finding_ignore, []) : item.key => item if try(item.enabled, true) }

  # Findings are discovered by scans, never created here; finding_id is an
  # external reference. Destroying this resource reopens the finding.
  finding_id = try(each.value.finding_id, null)
  comment    = try(each.value.comment, null)
}

resource "qualys_was_dns_override" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_was_dns_override, []) : item.key => item if try(item.enabled, true) }

  name     = try(each.value.name, null)
  comments = try(each.value.comments, null)

  tag_ids = try(coalescelist(concat(
    [for k in try(each.value.tag_keys, []) : local.asset_tag_ids[k]],
    [for n in try(each.value.external_tag_names, []) : local.external_tag_ids_by_name[n]]
  )), null)

  dynamic "mapping" {
    for_each = try(each.value.mapping, [])
    content {
      host_name  = mapping.value.host_name
      ip_address = mapping.value.ip_address
    }
  }
}

# ---------------------------------------------------------------------------
# Authentication records
# ---------------------------------------------------------------------------
# All 30 record types share one shape. vault_key/external_vault_name resolve
# both vault_id and vault_type together (see local.vault_by_title above) —
# vault_type is a fixed property of the vault itself, never a separate
# manually-entered value.

resource "qualys_auth_record_windows" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_windows, []) : item.key => item if try(item.enabled, true) }

  title       = try(each.value.title, null)
  username    = try(each.value.username, null)
  comments    = try(each.value.comments, null)
  ips         = try(each.value.ips, null)
  password    = try(each.value.password, null)
  domain      = try(each.value.domain, null)
  domain_type = try(each.value.domain_type, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_unix" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_unix, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_network_ssh" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_network_ssh, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_oracle" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_oracle, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_oracle_listener" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_oracle_listener, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_snmp" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_snmp, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_mssql" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_mssql, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_mysql" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_mysql, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_postgresql" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_postgresql, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_sybase" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_sybase, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_ibm_db2" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_ibm_db2, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_informix" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_informix, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_mariadb" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_mariadb, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_mongodb" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_mongodb, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_neo4j" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_neo4j, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_sapiq" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_sapiq, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_sap_hana" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_sap_hana, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_vmware" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_vmware, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_vcenter" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_vcenter, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_http" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_http, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_apache" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_apache, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_nginx" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_nginx, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_ms_iis" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_ms_iis, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_ibm_websphere" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_ibm_websphere, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_tomcat" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_tomcat, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_oracle_weblogic" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_oracle_weblogic, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_jboss" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_jboss, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_kubernetes" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_kubernetes, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_docker" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_docker, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}

resource "qualys_auth_record_palo_alto" "managed" {
  for_each = { for item in try(local.qualys_config.qualys_auth_record_palo_alto, []) : item.key => item if try(item.enabled, true) }

  title    = try(each.value.title, null)
  username = try(each.value.username, null)
  comments = try(each.value.comments, null)
  ips      = try(each.value.ips, null)
  password = try(each.value.password, null)

  vault_id = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].id
    : try(local.vault_by_title[each.value.external_vault_name].id, null)
  )
  vault_type = (
    try(each.value.vault_key, null) != null
    ? qualys_vault.managed[each.value.vault_key].type
    : try(local.vault_by_title[each.value.external_vault_name].type, null)
  )
}
