# 02 — Navigable Inventory of the VM and PA API Documentation

Deliverable 1: a navigable inventory of the relevant VM and PA API documentation at
<https://docs.qualys.com/en/vm/qweb-all-api/#t=get_started%2Fget_started.htm>.

> **Crawl caveat:** the discovery environment could not fetch pages from
> `docs.qualys.com` directly (network-policy block, see README). This inventory was
> reconstructed from confirmed page URLs surfaced through search against the official
> documentation, cross-checked against the published structure of the
> *Qualys API (VM and PA) User Guide v10.39.1*. URLs listed here were each observed as
> real documentation pages. Pages whose existence is inferred but was not observed are
> marked `Unverified`.

The all-in-one documentation is a MadCap Flare site composed of merged sub-projects.
Two URL grammars resolve to the same content set:

- `https://docs.qualys.com/en/vm/qweb-all-api/mergedProjects/<subproject>/<section>/<page>.htm`
- `https://docs.qualys.com/en/vm/api/<section>/<page>.htm` (per-module tree)

## Get Started (`get_started/`)

| Page | URL | Notes |
|---|---|---|
| Get Started | `https://docs.qualys.com/en/vm/qweb-all-api/get_started/get_started.htm` | Entry point named in the task |
| Authentication | `https://docs.qualys.com/en/vm/api/scans/get_started/authentication.htm` (also `/en/vm/api/assets/get_started/authentication.htm`, `/en/vm/api/users/get_started/authentication.htm`) | Basic vs session auth; mandatory `X-Requested-With` |
| API limits | `https://docs.qualys.com/en/vm/api/users/get_started/api_limits.htm` | Concurrency + rate limits, headers, HTTP 409 |
| API server URL / platforms | `https://docs.qualys.com/en/vm/api/scans/get_started/url_api_server.htm` + <https://www.qualys.com/platform-identification/> | Platform-specific base URLs |
| Appendix | `https://docs.qualys.com/en/vm/qweb-all-api/appendix/appendix_a.htm` | |

## Assets (`qapi-assets` / `en/vm/api/assets/`)

| Area | Confirmed pages |
|---|---|
| Host list | `assets/host_lists/host_list.htm` |
| Host list detection | `assets/host_lists/host_detection.htm` |
| Add IPs | `assets/asset_ips/add_ips.htm` |
| List IPs | `qweb-all-api/mergedProjects/qapi-assets/asset_ips/list_ips.htm` |
| Update IPs | `assets/asset_ips/update_ips.htm` |
| Excluded hosts (list/manage) | `assets/asset_ips/excl_host_list.htm`, `assets/asset_ips/Excluded_Hosts.htm`, `qweb-all-api/mergedProjects/qapi-assets/asset_ips/add_excl_hosts.htm` |
| Excluded hosts change history | User Guide §~p.1178 — exact page path `Unverified` |
| Asset groups | `assets/asset_groups/list_asset_groups.htm`, `add_new_asset_group.htm`, `edit_asset_group.htm`, `delete_asset_group.htm`, `asset_group_params.htm`, `asset_group_perms.htm` |
| Networks | `assets/networks/list_networks.htm`, `create_network.htm`, `update_network.htm`, `assign_appliance_network.htm` |
| Domains (V2) | `assets/domain_v2/Update_Domain.htm` (+ list/add/delete siblings — add/list/delete pages `Unverified` individually, family confirmed by 10.25 release notes) |
| Purge hosts | host purge action on `/api/2.0/fo/asset/host/` — official samples repo `github.com/QualysAPI/Qualys-API-Doc-Center` "Purge Host samples"; dedicated doc page path `Unverified` |

## Scans (`qapi-scan` / `en/vm/api/scans/`)

| Area | Confirmed pages |
|---|---|
| VM scans | `scans/vm_scans/launch_vm_scan.htm`, `manage_vm_scans.htm`, `scan_summary.htm`, `vm_scan_summary.htm`, `scan_statistics.htm`, appendix `scans/appendix/scan_results_json.htm` |
| Scan schedules | `scans/vm_schedules/list_scan_schedules.htm`, `create_scan_schedule.htm`, `update_scan_schedule.htm`, `schedule_params.htm`; delete at `qweb-all-api/mergedProjects/qapi-scan/vm_schedules/delete_scan_schedule.htm`; PC variants `scans/pc_schedules/...` |
| Scheduled scans & maps (legacy V1) | `qweb-all-api/mergedProjects/qapi-scan/maps/scheduled_scans_and_maps.htm` (`/msp/scheduled_scans.php`) |
| Scanner appliances | `scans/appliances/list_appliances.htm`, `update_scanner.htm`, `vlans.htm`, `delete_virtual_scanner.htm`, `scanner_details.htm`; create at `qweb-all-api/mergedProjects/qapi-scan/appliances/add_new_virtual_scanner.htm`; physical update (platform mirror) `qwebhelp/fo_portal/api_doc/scans/appliances/update_physical_scanner.htm` |
| Option profiles | `scans/ops/op_export.htm` (export/import); VM: `scans/ops/ops_vm/create_vm_op.htm`, `vm_op_params.htm`, list at `qweb-all-api/mergedProjects/qapi-scan/ops/ops_vm/list_vm_op.htm`; PC: `scans/ops/ops_pc/create_pc_op.htm`, `list_pc_op.htm`, update at `qweb-all-api/mergedProjects/qapi-scan/ops/ops_pc/update_pc_op.htm` |
| KnowledgeBase | `scans/kbase/knowledgebase.htm`, `qweb-all-api/mergedProjects/qapi-scan/kbase/knowledgebase_qvs.htm` |

## Reports (`qapi-rep` / `en/vm/api/reports/`)

| Area | Confirmed pages |
|---|---|
| Reports | `reports/reports/list_reports.htm`, `launch_report_tags.htm`, `fetch_report.htm` (cancel/delete confirmed via Quick Reference) |
| Scheduled reports | `reports/reports/list_sched_reports.htm` (list + `launch_now` only — **no create/update/delete pages exist**) |
| Scorecards | platform mirror `qwebhelp/fo_portal/api_doc/reports/reports/launch_scorecard.htm` |
| Asset search report | `reports/asset_search/asset_search_report.htm` |
| Report templates | `reports/templ/list_report_templates.htm` (V1 `/msp/report_template_list.php`); scan template CRUD `reports/templ_scan/create_scan_template.htm`, `update_scan_template.htm`, `export_scan_template.htm`, `scan_template_params.htm`; patch template `reports/templ_patch/create_patch_template.htm`, `update_patch_template.htm`; coverage statement `qweb-all-api/mergedProjects/qapi-rep/reports/api_support_for_report_templates.htm` (Scan, PCI Scan, Patch, Map only) |

## Scan Authentication (`qapi-auth` / `en/vm/api/scanauth/`)

| Area | Confirmed pages |
|---|---|
| List records (all types / per type) | `scanauth/record_list/list_record_types.htm` |
| Record types | `qweb-all-api/mergedProjects/qapi-auth/record_types/windows.htm`, `unix.htm`, `palo_alto.htm`, `Cisco_APIC_4.htm` (+ ~30 type pages; full page list `Unverified`, slug list in doc 03) |
| Delete record | `qweb-all-api/mergedProjects/qapi-auth/deletion/delete_record.htm` |
| Vaults | `qweb-all-api/mergedProjects/qapi-auth/vaults/list_vaults.htm`, `view_vault_settings.htm`, vault support matrix (platform mirror) `qwebhelp/fo_portal/api_doc/scanauth/vaults/vault_support_matrix.htm` |

## Users (`qapi-user` / `en/vm/api/users/`)

| Area | Confirmed pages |
|---|---|
| Users (V1 `/msp/user.php`, `/msp/user_list.php`) | `users/users/add_user.htm`, `edit_user.htm`, `editing_users.htm`, `list_users.htm`, `params_users.htm`, `perms_users.htm`; activate/deactivate `users/users_mgmt/user_activate.htm` (+ deactivate on platform mirror) |

## Supporting official sources referenced by the matrix

- Official API samples: <https://github.com/QualysAPI/Qualys-API-Doc-Center> (Network API
  samples, Purge Host samples, Host List Detection samples)
- API release notes: `https://cdn2.qualys.com/docs/release-notes/...` (e.g. 8.9, 8.10,
  10.20, 10.25 API release notes)
- API notifications feed: <https://notifications.qualys.com/api/>
- Knowledge base: `https://success.qualys.com/support/s/article/...` (e.g. 000005895 API
  limit headers, 000003463 purge-vs-delete)
