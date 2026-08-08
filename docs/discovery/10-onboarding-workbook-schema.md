# 10 — Onboarding Workbook (XLSX) Schema Implications

Companion to the discovery deliverables. The onboarding workbook is the human-facing
input that drives generated Terraform configuration. Its columns are constrained by
what the APIs actually accept (doc 03) and by two hard rules from the requirements:

1. **The workbook must declare the intended Qualys network explicitly.** Network
   selection is never derived from whether an address is public or private.
2. **No authentication secrets in the workbook** (or in Terraform configuration).
   Credentials are supplied as write-only arguments or, preferably, by reference to a
   Qualys vault record.

## Sheet: `networks`

| Column | Maps to | Notes |
|---|---|---|
| `network_name` | `/api/2.0/fo/network/` `create`/`update` `name` | |
| `network_id` | network ID | `0` = Global Default Network; required for import/reference |
| `intended_use` | documentation only | free text |

Validation: network must exist (or be declared for creation) **before** any IP or
scanner row references it. No delete path exists — removal is a manual/UI action.

## Sheet: `assets`

| Column | Maps to | Notes |
|---|---|---|
| `ips` | `asset/ip/?action=add` `ips` | single IPs and hyphenated ranges; **CIDR expanded client-side unless CIDR support is confirmed** (doc 08) |
| `network_name` | resolved to `network_id` | **explicit — never inferred from RFC1918 vs public** |
| `enable_vm` / `enable_pc` | add params | |
| `tracking_method` | `new_tracking_method` | IP / DNS / NETBIOS only (EC2, AGENT immutable) |
| `owner`, `ud1`–`ud3`, `comment` | `new_owner`, `new_ud1..3`, `new_comment` | update overwrites for all matched hosts |
| `asset_group` | asset group membership | resolved to group ID |

## Sheet: `asset_groups`

| Column | Maps to | Notes |
|---|---|---|
| `title` | `asset/group/` `add`/`edit` `title` | |
| `ips` | `set_ips` (authoritative) | additive/subtractive modes exist but Terraform uses `set_*` |
| `domains`, `dns_names`, `netbios_names` | sibling `set_*` params | **exact names pending verification (doc 08)** |
| `appliance_names` | appliance assignment | resolved to appliance IDs |
| `division`, `function`, `location`, `business_impact` | AG params | enumerated values |
| `cvss_enviro_cdp/td/cr/ir/ar` | AG params | enumerated values |

## Sheet: `tags`

| Column | Maps to | Notes |
|---|---|---|
| `tag_name`, `parent_tag_name` | `create/am/tag` `name`, `parentTagId` | hierarchy resolved by name → ID |
| `rule_type` | `ruleType` | blank/STATIC, or GROOVY / OS_REGEX / NETWORK_RANGE / NAME_CONTAINS / INSTALLED_SOFTWARE / OPEN_PORTS / VULN_EXIST / ASSET_SEARCH / GLOBAL_ASSET_VIEW |
| `rule_text` | `ruleText` | required for dynamic types |
| `color`, `description` | tag attributes | |

Note: adding and removing child tags in one call is not supported — the generator must
emit two operations.

## Sheet: `scanners`

| Column | Maps to | Notes |
|---|---|---|
| `scanner_name` | `appliance/?action=create` `name` | |
| `network_name` | `assign_network_id` | explicit |
| `polling_interval` | create/update | 60–360 |
| `asset_group` | create `asset_group_id` | **required when the API user is Unit Manager/Scanner, must be omitted for Managers** |
| `vlans`, `static_routes` | `set_vlans`, `set_routes` | pipe-delimited formats per doc 03 |
| `deployment_target` | **out of scope for this provider** | records which infrastructure provider/platform deploys the VM using the exported activation code |

The activation code is an **output**, never a workbook input.

## Sheet: `option_profiles`

Columns mirror the field-level VM option-profile parameters (`title`, `global`,
`scan_tcp_ports`, `scan_udp_ports`, `scan_dead_hosts`, `overall_performance`,
`vulnerability_detection`, `basic_information_gathering`, authentication toggles).
**Password brute-force lists and search-list selections must not be workbook columns**
until their round-trip behaviour is resolved — they are known not to survive
import/export faithfully.

## Sheet: `scan_schedules`

| Column | Maps to |
|---|---|
| `scan_title`, `option_profile`, `active` | create params |
| `target_type` (`assets` / `tags`), `asset_groups`, `ips`, `tag_includes`, `tag_excludes`, `tag_selector` | `target_from`, `asset_group_ids`, `ip`, `tag_set_include/exclude`, `tag_include_selector` |
| `scanner_selection` (`external` / `named` / `asset_group` / `tagset` / `network`) | `default_scanner`, `iscanner_name`, `scanners_in_ag`, `scanners_in_tagset`, `scanners_in_network` |
| `occurrence`, `frequency_days`, `frequency_weeks`, `weekdays`, `frequency_months`, `day_of_month`, `day_of_week`, `week_of_month` | recurrence params (weekdays numeric 0–6, 0 = Sunday) |
| `start_hour`, `start_minute`, `time_zone_code`, `observe_dst` | time params (time zone codes validated against `/msp/time_zone_code_list.php`) |
| `duration_hours`/`duration_mins`, `pause_after_*`, `resume_in_*` | `end_after`+`end_after_mins`, pause/resume pairs |
| `notify_before`, `notify_after`, `recipient_groups` | notification params — **distribution group names/IDs only, not raw email addresses** |

## Sheet: `auth_records`

| Column | Maps to | Notes |
|---|---|---|
| `record_title`, `record_type` | `/api/2.0/fo/auth/<type>/` `create` | Cisco IOS / Checkpoint use the `unix` slug |
| `ips`, `network_name` | targeting params | |
| `username` | type-specific | |
| `credential_source` | `vault` (preferred) or `write_only` | **no password column** |
| `vault_name`, `vault_secret_path` | vault reference params | vault records managed separately |
| `windows_domain_type` | Windows only | immutable after create → forces replacement |

## Validation pass (pre-apply)

The generator should run a `validation operation` sweep before emitting configuration:
network names → IDs, option-profile titles → IDs, scanner names → appliance IDs, tag
names → tag IDs, distribution-group names → IDs, report-template titles → IDs, plus
the tenant capability probe from doc 08 (Network Support, WAS licensing, purge
permission, QVSA license headroom).
