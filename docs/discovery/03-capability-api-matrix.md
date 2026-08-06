# 03 — Expanded Capability-to-API Matrix

Deliverable 2. Every discovered operation is recorded and classified, including
operations that do not map to a Terraform resource.

**Classification values:** `resource` | `data source` | `provider helper` |
`validation operation` | `imperative operation` | `unsupported` | `legacy` | `deferred`.

**Shared defaults for the VM/PA (VMDR) family** — stated once, apply to every
`/api/2.0/fo/...` row unless the row says otherwise:

- *API generation/family:* VMDR XML API, version 2.0 (`/api/2.0/fo/`).
- *Request encoding:* URL/form-encoded parameters (GET or POST; POST for mutations).
- *Response encoding:* XML with a `<!DOCTYPE ... SYSTEM "<dtd-url>">` reference.
  Mutations return `SIMPLE_RETURN` (`https://qualysapi.<platform>/api/2.0/simple_return.dtd`)
  with `DATETIME`, `TEXT`, and `ITEM_LIST/ITEM/KEY=ID` carrying created-object IDs.
  **The client must never fetch the referenced DTD; XML must be parsed with external
  entities and external DTD resolution disabled** (see doc 05).
- *Authentication:* HTTP Basic **or** session (`/api/2.0/fo/session/` → `QualysSession`
  cookie). Exception rows note where only Basic works.
- *Required headers:* `X-Requested-With: <anything>` on every v2 call;
  `Content-Type: application/x-www-form-urlencoded` for form POSTs (XML body posts use
  `text/xml`).
- *API limits:* subscription concurrency limit (default 2 per API) + rate limit
  (Standard: 300 calls/hour sliding window); breach → HTTP **409** with
  `X-RateLimit-*`/`X-Concurrency-Limit-*` headers (doc 05).
- *Verification:* `Confirmed` unless marked `Unverified` (doc 08 aggregates all
  `Unverified` items).

---

## 1. Host assets (module: VMDR / Assets)

Doc section: Assets → Host Lists / Asset IPs.

| Operation | Method | Path | Action | Key params | Response root / DTD | Status |
|---|---|---|---|---|---|---|
| List host assets | GET/POST | `/api/2.0/fo/asset/host/` | `list` | `ids`, `ips`, `ag_ids`, `use_tags`+`tag_set_by`, `no_vm_scan_since`, `vm_scan_since`, `details`, `truncation_limit` | `HOST_LIST_OUTPUT` / `.../asset/host/dtd/list/output.dtd` (renamed in 10.9) | Confirmed |
| List IPs | GET/POST | `/api/2.0/fo/asset/ip/` | `list` | `ips` filter | IP list output | Page confirmed; params `Unverified` |
| Add IPs / ranges | POST | `/api/2.0/fo/asset/ip/` | `add` | `ips` (single IPs + hyphenated ranges, comma-separated), `enable_vm`, `enable_pc`, `ag_title`, `comment` | SIMPLE_RETURN | Confirmed. **CIDR input `Unverified` — official examples show ranges only; plan to expand CIDR client-side** |
| Update host metadata | POST | `/api/2.0/fo/asset/ip/` | `update` | select by `ids`/`ips` (+`network_id`, `tracking_method`); set `new_tracking_method` (IP\|DNS\|NETBIOS only), `new_owner`, `new_ud1..3`, `new_comment` — values **overwrite all matched hosts** | SIMPLE_RETURN | Confirmed |
| Remove assets | — | — | — | **No delete API exists for subscription IPs** (official article 000003463); purge is the only API-side reduction | — | Confirmed absence |
| Purge host data | POST | `/api/2.0/fo/asset/host/` | `purge` | selectors `ids`/`ips`/`ag_ids`; `data_scope=vm|pc` and `echo_request` names `Unverified` | SIMPLE_RETURN-style | Endpoint/action/permissions confirmed; exact params `Unverified` |
| Host detection list | GET/POST | `/api/2.0/fo/asset/host/vm/detection/` | `list` | `status=New,Active,Re-Opened,Fixed`, `detection_updated_since/before`, `vm_scan_date_before/after`, `vm_processed_before/after`, `truncation_limit`, `id_min` | detection output (root name + DTD path `Unverified`); fields `FIRST_FOUND_DATETIME`, `LAST_FOUND_DATETIME`, `LAST_SCAN_DATETIME` | Confirmed; `include_ignored` param `Unverified` |
| Excluded IPs list | GET/POST | `/api/2.0/fo/asset/excluded_ip/` | `list` | `ips` | excluded host list XML | Confirmed |
| Excluded IPs add | POST | `/api/2.0/fo/asset/excluded_ip/` | `add` | `ips`, `comment`, `expiry_days` | SIMPLE_RETURN | Confirmed |
| Excluded IPs remove | POST | `/api/2.0/fo/asset/excluded_ip/` | `remove` | `ips`, `comment` | SIMPLE_RETURN | Action confirmed; params `Unverified` |
| Excluded hosts history | GET | path `Unverified` | — | — | — | Feature in guide TOC; `Unverified` |

- *Pagination:* host list & detection list default 1,000 records; `truncation_limit`
  override; continuation via `WARNING` (code 1980) containing next-URL with `id_min`.
- *Permissions:* add IPs — Managers (Unit Managers may need two-step add-to-AG);
  purge — Managers, or granted "Purge host information/history".
- *Stable identifier:* host ID (numeric) per network; IPs are per-subscription (and
  per-network when Network Support is on).
- *Tracking methods:* IP, DNS, NETBIOS mutable; EC2/AGENT immutable in both directions.

**Classification & Terraform mapping**

| Operation | Classification | Terraform mapping | Import format | Destructive effects | Priority | Unresolved questions |
|---|---|---|---|---|---|---|
| List hosts | data source | `qualys_host_assets` | n/a | none | P1 | — |
| Add/update IPs | resource | `qualys_ip_registration` (create=add, update=update; **destroy is a no-op or optional purge — no delete API**) | `ips` string (+`network_id`) | update overwrites metadata on all matched hosts | P1 | CIDR handling client-side; purge-on-destroy opt-in flag |
| Purge host data | imperative operation | NOT a resource; expose via separate stale-asset review/purge subsystem | n/a | **destructive** — removes vuln/compliance data, tickets, removes asset from AssetView; never blind-retry | P2 | exact param names (`data_scope`) |
| Host detections | data source | `qualys_host_detections` (feeds stale-asset subsystem) | n/a | none | P2 | `include_ignored` |
| Excluded IPs | resource | `qualys_excluded_ips` | `ips` | removing exclusions re-enables scanning | P3 | remove params |

---

## 2. Asset groups (module: VMDR / Assets)

Doc section: Assets → Asset Groups. Path `/api/2.0/fo/asset/group/`.

| Operation | Method | Action | Key params | Response / DTD | Status |
|---|---|---|---|---|---|
| List | GET/POST | `list` | `ids`, filters | `ASSET_GROUP_LIST_OUTPUT` / `.../asset/group/asset_group_list_output.dtd` | Confirmed |
| Create | POST | `add` | `title` (req), `network_id`, `ips`, `comments`, `division`, `function`, `location`, `business_impact`, `cvss_enviro_cdp/td/cr/ir/ar` | SIMPLE_RETURN with new ID | Confirmed |
| Update | POST | `edit` | `id` (req); member lists support **both additive (`add_ips`/`remove_ips`) and authoritative (`set_ips`) semantics**; official params page confirms set/remove pattern extends to domains, DNS/NetBIOS names, appliances (exact `set_domains`/`set_appliance_ids` names `Unverified`) | SIMPLE_RETURN | Confirmed (ips); sibling param names `Unverified` |
| Delete | POST | `delete` | `id` | SIMPLE_RETURN | Confirmed |
| Legacy V1 list | GET | `/msp/asset_group_list.php` | — | V1 XML | legacy — superseded by 2.0; formal deprecation status `Unverified` |

- **Answer to the discovery question:** asset group update is *parameter-selectable* —
  `set_*` = authoritative overwrite (can empty the group), `add_*`/`remove_*` = additive/
  subtractive. **Terraform must use `set_*` (authoritative) to converge on declared
  state.**
- *Permissions:* dedicated perms page; Unit Managers restricted to their business unit;
  exact API matrix `Unverified`.
- *Stable identifier:* numeric group ID (returned on create).

| Classification | Terraform mapping | Import | Destructive effects | Priority | Unresolved |
|---|---|---|---|---|---|
| resource + data source | `qualys_asset_group` (use `set_*`); `data.qualys_asset_groups` | group ID | delete removes group (not member hosts); `set_*` can empty member lists | **P0** (dependency of scans/schedules) | exact `set_*` names for domains/dns/netbios/appliances; scanner-appliance assignment params |

---

## 3. Networks (module: VMDR / Assets)

Doc section: Assets → Networks. Path `/api/2.0/fo/network/`. Requires the **Network
Support** subscription feature; Global Default Network is `network_id=0`.

| Operation | Method | Action | Key params | Status |
|---|---|---|---|---|
| List | GET/POST | `list` | `ids` | Confirmed — `NETWORK_LIST` / `network_list_output.dtd`, includes `SCANNER_APPLIANCE_LIST` |
| Create | POST | `create` | `name`; response includes new network ID | Confirmed |
| Update (rename) | POST | `update` | `id`, `name` | Confirmed |
| Assign appliance to network | POST | `/api/2.0/fo/appliance/` `assign_network_id` | `appliance_id`, `network_id`; one network per appliance; Managers only | Confirmed |
| Remove appliance from network | — | re-assign to another network / default | `Unverified` (no explicit un-assign action found) |
| Delete network | — | **No API delete found** (UI-only delete exists with conflict report + ~1h cleanup) | Confirmed absence at API level (`Unverified` residual) |

- *Overlapping IPs:* same IP in different networks = separate host identity per network
  (this is the feature's purpose). A network without an appliance cannot be scanned.
- *Permissions:* Managers (API-level statement `Unverified`).
- *Stable identifier:* numeric network ID (stability guarantee `Unverified`).
- **Design rule carried from requirements:** the onboarding workbook must declare the
  target network explicitly; the provider must never infer network from public/private
  address classification.

| Classification | Terraform mapping | Import | Destructive effects | Priority | Unresolved |
|---|---|---|---|---|---|
| resource (create/update only, **no destroy**) + data source | `qualys_network` (destroy = warn + remove from state); `data.qualys_networks` | network ID | none via API (no delete) | P1 | un-assign appliance mechanics; ID stability; Manager requirement |

---

## 4. Domains (module: VMDR / Assets)

Doc section: Assets → Domain V2. Path `/api/2.0/fo/asset/domain/`.

| Operation | Action | Notes | Status |
|---|---|---|---|
| List / Add / Update / Delete | `list`/`add`/`update`/`delete` | manages domains + netblocks; bulk delete since 10.25 (Dec 2023); special `none` domain for unregistered netblocks; **Basic auth only — session auth NOT supported**; update requires Manager | Confirmed (family); individual add/list/delete page URLs `Unverified` |

| Classification | Terraform mapping | Import | Priority |
|---|---|---|---|
| resource + data source | `qualys_domain` | domain name | P2 (needed for DNS-tracked scanning / asset-group domains) |

---

## 5. Scanner appliances (module: VMDR / Scans)

Doc section: Scans → Appliances. Path `/api/2.0/fo/appliance/`.

| Operation | Method | Action | Key params | Response | Status |
|---|---|---|---|---|---|
| List | GET/POST | `list` | `output_mode=brief|full`, `type=physical|virtual|offline`, `network_id`, `busy`, `scan_detail`, `include_license_info`, `show_tags` | `APPLIANCE_LIST_OUTPUT` / `appliance_list_output.dtd`; STATUS `Online|Offline`, SOFTWARE_VERSION, RUNNING_SCAN_COUNT, NETWORK_ID | Confirmed |
| Read details | GET | scanner details page | returns `IP_SCANNERS_LIST_OUTPUT` | — | Confirmed existence; params `Unverified` |
| Create virtual scanner | POST | `create` | `name`, `polling_interval` (60–360, default 180), `asset_group_id` (required for UM/Scanner; Managers must omit) | `APPLIANCE_CREATE_OUTPUT` — **includes ACTIVATION CODE (the personalisation code / "perscode") and REMAINING_QVSA_LICENSES** | Confirmed |
| Update virtual | POST | `update` | `id`, `name`, `comment`, `polling_interval`, `set_vlans`, `set_routes` (formats: VLAN `id\|ip\|netmask\|name[...ipv6]`, route `ip\|netmask\|gateway\|name`; up to 4094 each), `set_tags`/`add_tags`/`remove_tags`, `enable_ipv6` | SIMPLE_RETURN | Confirmed (`set_*` replace semantics implied, `Unverified` whether merge) |
| Update physical | POST | `/api/2.0/fo/appliance/physical/` `update` | `id`, `name`, `polling_interval`, `comment`, `set_vlans`, tags | SIMPLE_RETURN | Confirmed |
| Delete virtual | POST | `delete` | `id`; virtual only; fails while scans run; side effects: removed from asset groups, dependent schedules deactivated | SIMPLE_RETURN | Confirmed |
| Replace appliance | — | **UI-only wizard** (config transfer incl. VLANs/routes/heartbeat) | — | Confirmed UI-only |
| Assign to network | POST | `assign_network_id` | `appliance_id`, `network_id`; Managers only | SIMPLE_RETURN | Confirmed |

**Lifecycle separation (as required by the task):**

1. *Configuration inside Qualys* — API-driven: create virtual appliance → obtain
   activation code (perscode) from `APPLIANCE_CREATE_OUTPUT`; configure VLANs/routes;
   assign network. → Terraform resource territory.
2. *Deployment of the appliance VM* — **outside the Qualys provider**: requires an
   infrastructure provider (vSphere/AWS/Azure/GCP/OCI...) consuming the perscode
   exported by step 1.
3. *Connection & activation* — happens when the deployed VM personalises itself with
   the code; not a Qualys API call.
4. *Readiness verification* — poll `action=list` until STATUS=`Online` (readiness
   values `SS_OK`/`SS_CONNECTED` are **not** confirmed API values — console codes;
   `Unverified`). → provider helper / validation operation, not a resource.

| Operation | Classification | Terraform mapping | Import | Destructive | Priority | Unresolved |
|---|---|---|---|---|---|---|
| Create/update/delete virtual | resource | `qualys_virtual_scanner` — exports `activation_code` (sensitive), `status` | appliance ID | delete deactivates schedules, alters asset groups | **P1** | `set_vlans`/`set_routes` replace-vs-merge; full-output element list |
| List/details | data source | `data.qualys_scanner_appliances` | n/a | none | P1 | details params |
| Network assignment | part of resource (separate call) | attribute `network_id` on `qualys_virtual_scanner` | — | — | P1 | un-assign mechanics |
| Readiness wait | validation operation | optional `wait_for_online` timeout on resource / standalone helper | — | — | P2 | polling interval etiquette |
| Physical appliance mgmt | deferred | update-only; no create/delete API | — | — | deferred | — |
| Replace appliance | unsupported (UI-only) | document runbook | — | — | — | — |

---

## 6. Option profiles (module: VMDR / Scans)

Doc section: Scans → Option Profiles. **Two parallel API surfaces:**

| Operation | Method | Path | Action | Notes | Status |
|---|---|---|---|---|---|
| Export (all/one) | GET/POST | `/api/2.0/fo/subscription/option_profile/` | `export` | `option_profile_id`/`option_profile_title`; whole-XML `OPTION_PROFILES` / `option_profile_info.dtd`; **Manager only** | Confirmed |
| Import | POST | `/api/2.0/fo/subscription/option_profile/` | `import` | body = `OPTION_PROFILES` XML (`text/xml`); returns new ID; **Manager only**; since ~8.10 | Confirmed |
| List (VM) | GET/POST | `/api/2.0/fo/subscription/option_profile/vm/` | `list` | full XML incl. `BASIC_INFO/ID`, `IS_DEFAULT`, `IS_GLOBAL`, `USER_ID`, scan sections | Confirmed |
| Create (VM) | POST | `/api/2.0/fo/subscription/option_profile/vm/` | `create` | **field-level form params** (`title`, `global`, `scan_tcp_ports=full`, `vulnerability_detection=complete`, ...) per `vm_op_params.htm` | Confirmed |
| Update (VM) | POST | `/api/2.0/fo/subscription/option_profile/vm/` | `update` | `id` + create-params | Pattern confirmed via PC twin + quick ref; VM page `Unverified` |
| Delete (VM) | POST | `/api/2.0/fo/subscription/option_profile/vm/` | `delete` | `id` | Confirmed (quick reference) |
| PC variants | — | `/subscription/option_profile/pc/` | `list/create/update/delete` | parallel | Confirmed |
| v3 variant | — | `/api/3.0/fo/subscription/option_profile/vm/` | — | referenced in docs; details `Unverified` | Partially confirmed |

- **Managed lifecycle is viable — option profiles need not be read-only.** The viable
  Terraform schema is the *field-level* create/update/delete surface (form params),
  using `list` for read/drift. The XML import/export surface suits backup/migration
  (provider helper), not resource CRUD.
- **Known non-round-trippable fields (Confirmed, 8.10 release notes):** Password Brute
  Force Lists always import empty; search lists are matched by title with silent
  fallback of Vulnerability Detection to "Complete" → these fields must be
  `ignore_changes`-style suppressed or write-only in the schema.
- *Stable identifier:* numeric profile ID (in `BASIC_INFO/ID`).
- *Default designation:* `IS_DEFAULT` readable; whether it is settable via
  create/update params `Unverified`.

| Classification | Terraform mapping | Import | Destructive | Priority | Unresolved |
|---|---|---|---|---|---|
| resource (field-level API) + data source; XML import/export = provider helper | `qualys_vm_option_profile`; `data.qualys_option_profiles` | profile ID | delete breaks referencing scans/schedules | **P0–P1** (dependency of scans/schedules) | VM update page; replacement-forcing fields; `IS_DEFAULT` writability; v3 endpoint differences |

---

## 7. VM scans (module: VMDR / Scans)

Doc section: Scans → VM Scans. Path `/api/2.0/fo/scan/`.

| Operation | Method | Action | Key params | Status |
|---|---|---|---|---|
| List | GET/POST | `list` | `state`, `processed`, `type`, `target`, `user_login`, `launched_after/before_datetime`, `show_ags/op/status/last` | Confirmed — `SCAN_LIST_OUTPUT` / `scan_list_output.dtd` |
| Launch | POST | `launch` | `scan_title`, `option_id|option_title`; scanner selection: `iscanner_id|iscanner_name` (named), `default_scanner` (external), `scanners_in_ag`, `scanners_in_tagset`, `scanners_in_network`; targets: `ip`, `asset_group_ids|asset_groups`, `target_from=assets|tags` + `tag_set_by`/`tag_include_selector`/`tag_set_include`/`tag_set_exclude`, `exclude_ip_per_scan`, `ip_network_id` (default 0), `priority` 0–9, `runtime_http_header`, `client_id|client_name` (Consultant); EC2: `connector_name`, `ec2_endpoint`, `ec2_instance_ids` | Confirmed |
| Cancel / Pause / Resume / Delete | POST | `cancel`/`pause`/`resume`/`delete` | `scan_ref` | Confirmed |
| Status | GET/POST | `list` (`show_status`, mode brief/extended) | `scan_ref` | Confirmed |
| Fetch results | GET/POST | `fetch` | `scan_ref`, `mode=brief|extended`, `output_format=csv|json|csv_extended|json_extended` | Confirmed |
| Scan summary | GET/POST | `/api/2.0/fo/scan/summary/` `list` | `scan_date_since`, include flags; Manager only | Confirmed |
| VM scan summary (new) | GET/POST | `/api/2.0/fo/scan/vm/summary/` `list` | `scan_reference`, `scan_datetime_since/until`, include flags; Manager only | Confirmed |
| Scan statistics | GET/POST | `scan_statistics.htm` page | DTD fragment `/api/2.0/fo/scan/stats/vm_recrypt_results.dtd` | Existence confirmed; endpoint/params `Unverified` |

- *Asynchronous:* launch returns `SIMPLE_RETURN` immediately with REF
  `scan/<epoch>.<suffix>`; status must be polled. **All scanner-selection modes named in
  the requirements are supported** (external default, named appliance, AG scanners,
  tag-matched scanners, network scanners).

| Operation | Classification | Terraform mapping | Priority | Notes |
|---|---|---|---|---|
| Launch/cancel/pause/resume | **imperative operation** | NOT a Terraform resource (no stable lifecycle: a scan is a job, re-runs are new refs, destroy is meaningless). Expose via runbook/CLI or an ephemeral "action" if ever justified | P3 | classified per task instruction |
| Delete scan / fetch results | imperative operation | stale-scan cleanup subsystem | P3 | destructive delete — never blind-retry |
| Scan list/status/summary | data source / validation operation | `data.qualys_scans` (list); summary Manager-only | P3 | — |

---

## 8. VM scan schedules (module: VMDR / Scans)

Doc section: Scans → Schedules. Path `/api/2.0/fo/schedule/scan/` (PC variant
`/api/2.0/fo/schedule/scan/compliance/`).

| Operation | Method | Action | Key params | Status |
|---|---|---|---|---|
| List | GET/POST | `list` | `id`, `is_active`-style filters | Confirmed — `SCHEDULE_SCAN_LIST_OUTPUT` / `schedule_scan_list_output.dtd` |
| Create | POST | `create` | `scan_title`, `option_id|option_title`, `active=0|1`; targets/scanners as scan launch (`ip`, `asset_group_ids`, `target_from=tags` + tag params, `iscanner_name`, `priority`); recurrence: `occurrence=daily|weekly|monthly`, `frequency_days`, `frequency_weeks`+`weekdays` (numeric 0–6, 0=Sunday), `frequency_months` + (`day_of_month` XOR `day_of_week`+`week_of_month`); time: `start_hour` 0–23, `start_minute`, `time_zone_code`, `observe_dst=yes|no`; duration: `end_after` (0–119)+`end_after_mins`, `pause_after_hours`+`pause_after_mins`, `resume_in_days`+`resume_in_hours` (resume relative to start); notifications: `before_notify=1`+`before_notify_unit`+`before_notify_time`, `after_notify=1`, `recipient_group_ids` (distribution-group IDs, only with a notify flag) | Confirmed (`start_date` param name `Unverified`; `client_id/client_name` as inputs `Unverified`) |
| Update | POST | `update` | `id` + fields; `set_start_time` documented for start-time changes | Confirmed |
| Delete | POST | `delete` | `id` | Confirmed |
| Enable/disable | POST | `update` with `active=0|1` | — | Confirmed |
| Launch schedule now | — | **does not exist** (no `action=launch` on schedule/scan; asymmetric with report schedules) | — | Confirmed absence |
| Legacy V1 | GET/POST | `/msp/scheduled_scans.php` | `add_task`/`drop_task`/`save_task` | legacy; **only** surface for scheduled *maps*; no formal deprecation notice found (`Unverified`) |

- *Timezones:* `time_zone_code` values from V1 helper `/msp/time_zone_code_list.php`
  (DTD `time_zone_code_list.dtd`); DST via `observe_dst`.
- *Recipients:* distribution groups only — **no free-form email addresses**; no
  distribution-group CRUD API found (doc 08).

| Classification | Terraform mapping | Import | Priority | Unresolved |
|---|---|---|---|---|
| resource + data source; legacy `/msp/scheduled_scans.php` = legacy (not used); timezone list = provider helper/validation | `qualys_scan_schedule`; `data.qualys_scan_schedules` | schedule ID | **P0–P1** (core of recurring-scan onboarding) | `start_date`/`end date` params; ACTIVE output values 2/3; introduction version |

---

## 9. Reports (module: VMDR / Reports)

Doc section: Reports. Path `/api/2.0/fo/report/`.

| Operation | Method | Action | Key params | Status |
|---|---|---|---|---|
| List | GET/POST | `list` | `id`, `state` (Running/Finished/Submitted/Canceled/Errors), `EXPIRATION_DATETIME` in output | Confirmed — `REPORT_LIST_OUTPUT` / `report_list_output.dtd` (an `/api/3.0/fo/report/` DTD also referenced — details `Unverified`) |
| Launch | POST | `launch` | `template_id`, `report_title`, `output_format`, `report_type`, `report_refs` (scan-based), `ips`/`asset_group_ids` (host-based), `pdf_password`, `recipient_group` (distribution groups, max 50) | Confirmed |
| Launch with tags | POST | `launch` | `use_tags=1`, `tag_set_by=id|name`, `tag_include_selector=all|any`, `tag_set_include/exclude` — vulnerability & compliance reports (other types `Unverified`) | Confirmed |
| Status | GET/POST | `list` | poll by `id` | Confirmed |
| Cancel | POST | `cancel` | `id` | Confirmed |
| Fetch/download | GET/POST | `fetch` | `id`; requires Report Share; >1 GB auto-ZIP | Confirmed |
| Delete saved report | POST | `delete` | `id` | Confirmed |
| Scorecard launch | POST | `/api/2.0/fo/report/scorecard/` `launch` | `name`, `report_title`, `output_format`, `asset_groups`/`business_unit`; **vulnerability scorecards only** | Confirmed |
| Asset search report | POST | `/api/2.0/fo/report/asset/` `search` | IP-range filters, `output_format` | Confirmed |

**Report templates** (separate object class, as the task requires):

| Operation | Path | Actions | Status |
|---|---|---|---|
| List all templates | `/msp/report_template_list.php` (V1) | — | Confirmed (legacy generation, still the only list) |
| Scan template CRUD | `/api/2.0/fo/report/template/scan/` | `create`/`update`/`export`/`delete` (XML body, `report_format=xml`) | Confirmed |
| PCI scan template | `/api/2.0/fo/report/template/pciscan/` | export family | Confirmed |
| Patch template | `/api/2.0/fo/report/template/patch/` | `create` (POST), `update` (PUT) | Confirmed |
| Map template | per coverage statement | — | Confirmed coverage; endpoint `Unverified` |
| Remediation/compliance templates | — | **no API found** | Confirmed absence (search-level) |

| Operation | Classification | Terraform mapping | Priority |
|---|---|---|---|
| Report launch/cancel/fetch/delete | **imperative operation** (async job artefacts) | stale-report/reporting subsystem, not resources | P3 |
| Report list | data source | `data.qualys_reports` | P3 |
| Scan/patch template CRUD | resource (deferred) | `qualys_report_template_*` — XML-body resources, complex schema | deferred |
| Template list | data source | `data.qualys_report_templates` (needed to reference `template_id`) | **P1** (schedules/reports depend on template IDs) |
| Scorecard / asset-search | imperative operation | — | deferred |

---

## 10. Report schedules & recipients (module: VMDR / Reports)

Path `/api/2.0/fo/schedule/report/`.

| Operation | Action | Status |
|---|---|---|
| List | `list` (`id`, `is_active=true|false`) | Confirmed — `schedule_report_list_output.dtd` |
| Launch scheduled report now | `launch_now` (`id`) | Confirmed |
| Create / Update / Delete / Enable / Disable | — | **No API exists — GUI-only** (docs tree has only list + launch_now; multiple community requests unanswered) | Confirmed absence |

- *Recipients:* notifications go to **distribution groups** (users/emails configured in
  UI; "Send as Bcc" available). `recipient_group` on report launch caps at 50 groups.
  **Whether arbitrary external addresses are accepted:** only via the members of a
  distribution group maintained in the UI — the API accepts group IDs, not raw
  addresses. Distribution-group CRUD API: none found (`Unverified` absence).

| Classification | Terraform mapping | Priority |
|---|---|---|
| data source (`data.qualys_report_schedules`) + imperative `launch_now`; schedule definition itself **unsupported** (GUI-only) | no `qualys_report_schedule` resource is possible today — record as gap in doc 08 | P3 |

---

## 11. Scan authentication (module: VMDR / Scan Authentication)

Doc section: Scan Authentication. Base `/api/2.0/fo/auth/`.

| Operation | Path | Actions | Status |
|---|---|---|---|
| List all records (summary) | `/api/2.0/fo/auth/` | `list` (`title`, `ids`, `id_min/max`) | Confirmed |
| Per-type CRUD | `/api/2.0/fo/auth/<type>/` | `list`/`create`/`update`/`delete`; `title` req on create; `ids` req on update/delete; targeting `ips`/`add_ips`/`remove_ips`, `network_id` | Confirmed |
| Vault CRUD | `/api/2.0/fo/vault/` | `list`/`create`/`update`/`delete` (`title`+`type` req) | Confirmed |

Confirmed type slugs: `windows`, `unix` (**also used for Cisco IOS & Checkpoint
Firewall — no separate slugs**), `network_ssh`, `oracle`, `oracle_listener`, `snmp`,
`ms_sql`, `mysql`, `postgresql`, `sybase`, `ibm_db2`, `informixdb`, `mariadb`,
`mongodb`, `neo4j`, `sapiq`, `sap_hana`, `vmware`, `vcenter`, `http`, `apache`,
`nginx`, `ms_iis`, `ibm_websphere`, `tomcat`, `oracle_weblogic`, `jboss`,
`kubernetes`, `docker`, `palo_alto_firewall` (+ Cisco APIC, SharePoint pages).

Vault types (Confirmed): CyberArk PIM Suite, CyberArk AIM, Thycotic Secret Server,
Quest, CA Access Control, Hitachi ID PAM, Lieberman ERPM, BeyondTrust PBPS, Wallix
AdminBastion, HashiCorp; Azure Key Vault at least for Palo Alto records (10.1).

- *Pagination:* fixed 1,000 records/request with `WARNING` continuation.
- *Secrets:* passwords **may be echoed** in list responses unless the caller lacks
  "View Password in Authentication Record" (VM-side per-type behaviour partially
  `Unverified`). **Decision (per task constraint):** treat all secrets as write-only
  arguments regardless; never store in workbook, plan output, or rely on read-back.
- *Windows:* domain type immutable after save (forces replacement).

| Classification | Terraform mapping | Import | Priority | Notes |
|---|---|---|---|---|
| resource (per family) + data source | **Both approaches** (per task question): `qualys_auth_record_<type>` resources with **write-only** credential arguments + vault reference support; `data.qualys_auth_records` for referencing pre-existing records | record ID | P1–P2 (Windows/Unix first — onboarding needs) | vault-backed records preferred to raw passwords; drift on credentials undetectable by design |
| Vaults | resource | `qualys_vault` | vault ID | P2 | per-type vault param schemas `Unverified` |

---

## 12. Users & distribution groups (module: VMDR / Users)

| Operation | Path | Notes | Status |
|---|---|---|---|
| Add user | `POST /msp/user.php?action=add` | V1; `user_role`, `business_unit`, `send_email`, contact fields; **no session auth**; `USER_OUTPUT`/`user_output.dtd` | Confirmed |
| Edit user | `POST /msp/user.php?action=edit` | can clear params; asset-group assignment | Confirmed |
| Activate/deactivate | `/msp/user.php?action=activate|deactivate` | role-scoped | Confirmed |
| List users | `POST /msp/user_list.php` | `user_list_output.dtd` | Confirmed |
| V2 user API | `/api/2.0/fo/user/` | third-party claim only | `Unverified` |
| Distribution groups CRUD | — | **no API found** (UI-only) | `Unverified` absence |

| Classification | Terraform mapping | Priority |
|---|---|---|
| legacy generation but the only surface → resource possible with caveats | `qualys_user` (deferred; V1 API, no session auth, password/activation flows) + `data.qualys_users` | deferred (P3) |

---

## 13. WAS (module: WAS, `/qps/rest/3.0/.../was/...`)

| Object | Operations (confirmed unless noted) | Terraform mapping | Priority |
|---|---|---|---|
| `webapp` | `create`, `search`, `count`, `delete` (single + bulk-by-filter); `get`/`update` per URL grammar (pattern-confirmed); tags via `TagList` element | resource `qualys_web_application` + data source | P2 |
| `optionprofile` | `create`, `update`, `get`, `search` confirmed; `delete` `Unverified` | resource `qualys_was_option_profile` | P2 |
| `webappauthrecord` | `create` confirmed (form/server/Selenium/OAuth2); `get` masks secrets without view-permissions; update/delete/search pattern-confirmed | resource `qualys_was_auth_record` (write-only secrets) | P2 |
| `wasscan` | `launch`, `search`, `status`, `download` confirmed; `cancel`/`delete` endpoints `Unverified` (cancelAfterNHours param confirmed) | imperative operation | P3 |
| `wasscanschedule` | `update` + object name confirmed; full CRUD + recurrence field model `Unverified` | resource (pending field verification) `qualys_was_scan_schedule` | P2–P3 |
| `report` / report templates | `create`, `get` (status), `download` confirmed; search template endpoint confirmed; delete pattern-confirmed | imperative operation + data source | P3 |
| report schedules | UI feature; **API `Unverified`** | gap (doc 08) | — |

Auth: Basic per request (baseline); JWT "token-based auth" documented in a technical
brief — per-endpoint WAS support `Unverified`. Encoding: XML default; JSON since
WAS 4.5 with both `Accept` and `Content-Type: application/json`. Pagination:
`hasMoreRecords`/`lastId` + Criteria `GREATER`.

---

## 14. AssetView / CSAM tagging (module: AM, `/qps/rest/2.0/.../am/...`)

| Capability | Operation | Status |
|---|---|---|
| Tag CRUD | `create/search/get/update/delete/count /am/tag` | Confirmed |
| Static tags | `ruleType` omitted/STATIC | Confirmed |
| Dynamic tags | `ruleType`: GROOVY, OS_REGEX, NETWORK_RANGE, NAME_CONTAINS, INSTALLED_SOFTWARE, OPEN_PORTS, VULN_EXIST, ASSET_SEARCH, GLOBAL_ASSET_VIEW + `ruleText` | Confirmed (`CLOUD_ASSET` `Unverified`) |
| Hierarchy | `parentTagId` on create; `children.set.TagSimple[]`; **never add and remove children in one call** | Confirmed |
| Assign tags to host assets | `POST /qps/rest/2.0/update/am/hostasset` with `tags.add.TagSimple{id}` / `tags.remove`; bulk via Criteria `id IN` | Confirmed |
| `update/am/asset` variant | static tags only — **dynamic tags rejected** on add/remove/set | Confirmed |
| List assets by tag | `search/am/hostasset` Criteria `tagName`/`tagId EQUALS` | Confirmed |
| Untagged-asset discovery | negative/NOT operator **not supported** on `search/am/hostasset` (community evidence); no official `NONE` operator found | `Unverified` — plan client-side set difference |
| WAS webapp tag assignment | `TagList` on `webapp` create/update | Confirmed |
| Cross-module tag identity | Strong official indications of a single shared tag tree (AssetView tag tree visible in all apps; WAS tag API references CSAM-managed tags) — **but no explicit statement that the same tag ID is usable across VMDR, AssetView and WAS. Status: `Unverified` (per task instruction, must remain so until official confirmation).** | `Unverified` |

| Classification | Terraform mapping | Import | Priority |
|---|---|---|---|
| resource + data source | `qualys_asset_tag` (static/dynamic, hierarchy); `qualys_host_tag_assignment`; `data.qualys_asset_tags`, `data.qualys_tagged_assets` | tag ID | **P0–P1** (tag-based scan/report targeting depends on it) |

---

## 15. TotalCloud / CloudView (module: CloudView — minimal retained scope)

| Capability | Endpoint | Status | Classification |
|---|---|---|---|
| GCP connector CRUD (existing provider) | `/cloudview-api/rest/v1/gcp/connectors` (POST/GET/PUT/DELETE; JSON; Basic; `pageNo`/`pageSize`) | Confirmed | resource + data source (existing `qualys_gcp_connector`) |
| Connector v3 replacement | `/qps/rest/3.0/create|run|delete/am/gcpassetdataconnector` (+ AWS/Azure) | Confirmed | **migration target** — CloudView connector APIs deprecated; plan migration of the existing resource (doc 07/09) |
| All other TC/CloudView operations | — | out of scope by explicit product decision | unsupported (out of scope) |
