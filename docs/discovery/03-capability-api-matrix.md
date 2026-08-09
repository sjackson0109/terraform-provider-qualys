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
| Add IPs / ranges | POST | `/api/2.0/fo/asset/ip/` | `add` | `ips` (single IPs + hyphenated ranges, comma-separated), `tracking_method`, `enable_vm`, `enable_pc`, **`enable_certview`**, **`enable_sca`**, `owner`, `ud1`, `ud2`, `ud3`, `comment`, `ag_title`, `echo_request` | SIMPLE_RETURN | Confirmed (core params); `enable_certview`/`enable_sca`/`ud*` on add **Corroborated (non-official)**. **CIDR input still `Unverified`** — every retrievable description says "a host IP range is specified with a hyphen"; expand CIDR client-side |
| Update host metadata | POST | `/api/2.0/fo/asset/ip/` | `update` | **Selectors:** `ips`, `ids`, `ag_ids`, `ag_titles`, `network_id`/`network_name` (scoping only — cannot move an IP between networks), `tracking_method`, `host_dns`, `host_netbios`. **Setters:** `new_tracking_method` (IP\|DNS\|NETBIOS only), `new_owner`, `new_ud1`, `new_ud2`, `new_ud3`, `new_comment`. Setters **overwrite all matched hosts** | SIMPLE_RETURN | Confirmed (full selector/setter split — see callout below) |
| Remove assets | — | — | — | **No delete API exists for subscription IPs** (official article 000003463); purge is the only API-side reduction | — | Confirmed absence |
| Purge host data | POST | `/api/2.0/fo/asset/host/` | `purge` | IP/AG mode (`use_tags=0`, default): `ids`, `ips`, `ag_ids`, `ag_titles`, `network_ids` (needs Network Support); tag mode (`use_tags=1`): `tag_set_include`, `tag_set_exclude`, `tag_include_selector`, `tag_exclude_selector`; filters: `data_scope` (`vm`, `pc`, or `vm,pc`), `compliance_enabled`, `no_vm_scan_since`, `no_compliance_scan_since`, `os_pattern`, `echo_request` | **`BATCH_RETURN`** (not SIMPLE_RETURN) — `BATCH_LIST/BATCH/TEXT` ("Hosts Queued for Purging") + `ID_SET`; DTD `/api/2.0/fo/asset/host/dtd/purge/output.dtd` | Endpoint/action/permissions Confirmed; parameter set **Corroborated (non-official)** — the `qualysdk` call schema lists exactly this set, matching the XSOAR pack independently; `tag_set_by` for purge `Unverified` |
| Host detection list | GET/POST | `/api/2.0/fo/asset/host/vm/detection/` | `list` | `status`, `include_ignored` (0/1, default 0), `include_disabled` (0/1, default 0), `detection_updated_since/before`, `detection_processed_after/before`, `detection_last_tested_since/before`(`_days`), `vm_scan_date_before/after`, `vm_auth_scan_date_before/after`, `vm_processed_before/after`, `max_days_since_last_vm_scan`, `max_days_since_detection_updated`, `truncation_limit`, `id_min`; dates `YYYY-MM-DD[THH:MM:SSZ]` | root **`HOST_LIST_VM_DETECTION_OUTPUT`**; DTD **`/api/2.0/fo/asset/host/vm/detection/dtd/output.dtd`** (no `list/` segment); fields `FIRST_FOUND_DATETIME`, `LAST_FOUND_DATETIME`, `LAST_SCAN_DATETIME` | Confirmed (root/DTD/`include_ignored`/`include_disabled`/date filters) |
| Excluded IPs list | GET/POST | `/api/2.0/fo/asset/excluded_ip/` | `list` | `ips` | excluded host list XML | Confirmed |
| Excluded IPs add | POST | `/api/2.0/fo/asset/excluded_ip/` | `add` | `ips`, `comment`, `expiry_days`, `dg_names`, `network_id` | SIMPLE_RETURN | Confirmed |
| Excluded IPs remove | POST | `/api/2.0/fo/asset/excluded_ip/` | `remove`, `remove_all` | `ips`, `comment`, `network_id` (*"`ips` is invalid for `remove_all`"*) | SIMPLE_RETURN | Confirmed |
| Excluded hosts history | GET/POST | `/api/2.0/fo/asset/excluded_ip/history/` | `list` | `ips`, `ids`, `id_min`, `id_max`, `network_id`, `echo_request` | XML | **Confirmed** (endpoint path now known) |

> ✅ **RESOLVED — `asset/ip/?action=update` selector vs setter split.** The Quick
> Reference lists both groups on the same action: the **bare** names (`tracking_method`,
> `host_dns`, `host_netbios`, `ips`, `ids`, `ag_ids`, `ag_titles`, `network_id`,
> `network_name`) are **selectors** that choose which hosts to act on, while the
> **`new_*`** names (`new_tracking_method`, `new_owner`, `new_ud1`, `new_ud2`,
> `new_ud3`, `new_comment`) are the **setters** that carry the new values. The earlier
> conflict came from a third-party SDK sending bare `owner`/`ud1`/`comment` as if they
> were setters — those are silently treated as filters, so such a call filters rather
> than updates. **The provider must use the `new_*` spelling for all writes.**

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
| Purge host data | imperative operation | NOT a resource; expose via separate stale-asset review/purge subsystem | n/a | **destructive** — removes vuln/compliance data, tickets, removes asset from AssetView; **asynchronous/queued** (BATCH_RETURN returns hosts *queued*, so a follow-up read may still see the host — verification must poll, not assume); never blind-retry | P2 | `tag_set_by` for purge |
| Host detections | data source | `qualys_host_detections` (feeds stale-asset subsystem) | n/a | none | P2 | `include_ignored` |
| Excluded IPs | resource | `qualys_excluded_ips` | `ips` | removing exclusions re-enables scanning | P3 | remove params |

---

## 2. Asset groups (module: VMDR / Assets)

Doc section: Assets → Asset Groups. Path `/api/2.0/fo/asset/group/`.

| Operation | Method | Action | Key params | Response / DTD | Status |
|---|---|---|---|---|---|
| List | GET/POST | `list` | `ids`, filters | `ASSET_GROUP_LIST_OUTPUT` / `.../asset/group/asset_group_list_output.dtd` | Confirmed |
| Create | POST | `add` | **bare names** (no `set_` prefix): `title` (req), `network_id`, `ips`, `domains`, `dns_names`, `netbios_names`, `appliance_ids`, `default_appliance_id`, `comments`, `division`, `function`, `location`, `business_impact`, `cvss_enviro_cdp/td/cr/ir/ar` | SIMPLE_RETURN with new ID | Confirmed (official params page); full bare-name list **Corroborated (non-official)** |
| Update | POST | `edit` | `id` (req). **Member lists use `add_*`/`remove_*`/`set_*` triads**: `add_ips`/`remove_ips`/`set_ips`, `add_domains`/`remove_domains`/`set_domains`, `add_dns_names`/`remove_dns_names`/`set_dns_names`, `add_netbios_names`/`remove_netbios_names`/`set_netbios_names`, `add_appliance_ids`/`remove_appliance_ids`/`set_appliance_ids`. Scalars are **set-only**: `set_title`, `set_comments`, `set_division`, `set_function`, `set_location`, `set_business_impact`, `set_default_appliance_id`, `set_cvss_enviro_cdp/td/cr/ir/ar` | SIMPLE_RETURN | `set_ips`/`add_ips`/`remove_ips` + set/remove semantics Confirmed; remaining triads and scalar `set_*` names **Corroborated (non-official)** — the `qualysdk` `manage_ag` call schema (read directly from source) lists this exact union of bare + `set_*` names, independently matching the XSOAR pack. Note `network_id` is **absent** from the edit set, reinforcing that network is create-time only |
| Delete | POST | `delete` | `id` | SIMPLE_RETURN | Confirmed |
| Legacy V1 list | GET | `/msp/asset_group_list.php` | — | V1 XML | legacy — superseded by 2.0; formal deprecation status `Unverified` |

- **Answer to the discovery question:** asset group update is *parameter-selectable* —
  `set_*` = authoritative overwrite (can empty the group), `add_*`/`remove_*` = additive/
  subtractive. **Terraform must use `set_*` (authoritative) to converge on declared
  state.**
- **Create/update parameter asymmetry (important for the client):** `action=add` takes
  bare names (`ips`, `domains`, `title`, ...); `action=edit` takes prefixed names
  (`set_ips`, `set_domains`, `set_title`, ...). The two calls are **not** the same
  parameter set with a different action — the client needs separate encoders.
- *Owner and network:* no `set_owner_user_id` / `set_network_id` parameter was found on
  `edit`; `network_id` appears only on `add`, while `OWNER_USER_ID`/`NETWORK_IDS` exist
  as *output* attributes. **Treat owner and network as create-time (ForceNew) until
  disproved** (`Unverified`).
- *Enum caveat:* `business_impact` — the Qualys UI doc lists Critical/High/Medium/Low/
  **Minor**; one wrapper models the tail as **none**. The exact enum tail is
  `Unverified`; validate against the tenant rather than hard-coding.
- *Permissions:* dedicated perms page; Unit Managers restricted to their business unit;
  exact API matrix `Unverified`.
- *Stable identifier:* numeric group ID (returned on create).

| Classification | Terraform mapping | Import | Destructive effects | Priority | Unresolved |
|---|---|---|---|---|---|
| resource + data source | `qualys_asset_group` (use `set_*` on update, bare names on create); `data.qualys_asset_groups` | group ID | delete removes group (not member hosts); `set_*` can empty member lists | **P0** (dependency of scans/schedules) | owner/network mutability; `business_impact` enum tail |

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
| Delete network | — | **No API delete exists.** The official action enumeration for `/api/2.0/fo/network/` is `list` and `create|update` only. A UI-only delete exists (conflict report + ~1h cleanup) | **Confirmed absence** |

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
| List | GET/POST | `list` | `output_mode=brief|full`, `type=physical|virtual|offline`, `network_id`, `busy`, `scan_detail`, `include_license_info`, `show_tags` | `APPLIANCE_LIST_OUTPUT` / `appliance_list_output.dtd`. `output_mode=full` APPLIANCE elements: `ID`, `UUID`, `NAME`, `NETWORK_ID`, `SOFTWARE_VERSION`, `RUNNING_SLICES_COUNT`, `RUNNING_SCAN_COUNT`, `STATUS`, `CMD_ONLY_START`, `MODEL_NUMBER`, `SERIAL_NUMBER`, **`ACTIVATION_CODE`**, `INTERFACE_SETTINGS`, `PROXY_SETTINGS`, `IS_CLOUD_DEPLOYED`, `CLOUD_INFO`, `VLANS` (children `SETTING`, `VLAN`), `STATIC_ROUTES`, `ML_LATEST`/`ML_VERSION`, `VULNSIGS_LATEST`/`VULNSIGS_VERSION`, `ASSET_GROUP_COUNT`, `ASSET_GROUP_LIST`, `ASSET_TAGS_LIST`, `LAST_UPDATED_DATE`, `POLLING_INTERVAL`, `USER_LOGIN`, **`HEARTBEATS_MISSED`**, `SS_CONNECTION`, `SS_LAST_CONNECTED`, `FDCC_ENABLED`, `USER_LIST`, `UPDATED`, `COMMENTS`, `RUNNING_SCANS`, `MAX_CAPACITY_UNITS` | Confirmed |
| Read details | GET | scanner details page | returns `IP_SCANNERS_LIST_OUTPUT` | — | Confirmed existence; params `Unverified` |
| Create virtual scanner | POST | `create` | `name`, `polling_interval` (60–360, default 180), `asset_group_id` (required for UM/Scanner; Managers must omit) | `APPLIANCE_CREATE_OUTPUT` — **includes ACTIVATION CODE (the personalisation code / "perscode") and REMAINING_QVSA_LICENSES** | Confirmed |
| Update virtual | POST | `update` | `id`, `name`, `comment`, `polling_interval` (60–3600, default 180), `set_vlans`, `set_routes`, `set_tags`/`add_tags`/`remove_tags`, `enable_ipv6`. **`set_vlans`/`set_routes` are authoritative replace** — the value passed becomes the entire list and `""` deletes all records; there are **no** `add_vlans`/`remove_vlans` parameters. Formats: VLAN `<VLAN_ID>|<IPv4>|<NETMASK>|<VLAN_NAME>|ipv6_static\|ipv6_auto|<IPv6>` (skipped IPv4 attributes need an empty space placeholder), route `<IP>|<NETMASK>|<GATEWAY>|<NAME>`; comma-separated for multiple; up to 4094 each. Requires the "VLANs and static routes" subscription feature | SIMPLE_RETURN | Confirmed (replace semantics now confirmed) |
| Update physical | POST | `/api/2.0/fo/appliance/physical/` `update` | `id`, `name`, `polling_interval`, `comment`, `set_vlans`, tags | SIMPLE_RETURN | Confirmed |
| Delete virtual | POST | `delete` | `id`; virtual only; fails while scans run; side effects: removed from asset groups, dependent schedules deactivated | SIMPLE_RETURN | Confirmed |
| Replace appliance | POST | `/api/2.0/fo/appliance/replace_iscanner/` `action=replace` | `old_scaner_name` *(sic — Quick Reference typo, verify spelling)*, `new_scanner_name`, `do_not_copy_settings={0|1}`, `do_not_remove_new_scanner_from_objects={0|1}`, `echo_request` | SIMPLE_RETURN | **Confirmed — reverses the earlier "UI-only" finding.** A replace API does exist |
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
4. *Readiness verification* — poll `action=list` until `STATUS`=`Online`, using
   `HEARTBEATS_MISSED`, `LAST_UPDATED_DATE`, `SS_CONNECTION`/`SS_LAST_CONNECTED` and
   the `ML_VERSION` vs `ML_LATEST` / `VULNSIGS_VERSION` vs `VULNSIGS_LATEST` pairs
   (up-to-date check without an extra call). There is **no "check readiness now"
   action** — polling `action=list` is the mechanism, and the platform heartbeat runs
   only **every 4 hours** (offline threshold configurable 1–5 missed checks), so a
   short-timeout wait loop inside Create would be wrong. → provider helper /
   validation operation, not a resource. (`SS_OK`/`SS_CONNECTED` remain unconfirmed as
   API *values*; `SS_CONNECTION` is the real element name.)

**Activation code retrieval (state-design critical):** `ACTIVATION_CODE` appears in
`APPLIANCE_LIST_OUTPUT` as well as `APPLIANCE_CREATE_OUTPUT`, **but only while the
appliance is not yet activated** — once activated it is no longer returned. A Read that
blindly overwrites the attribute will therefore erase it from state on the first apply
after activation. The resource must treat `activation_code` as Computed + Sensitive and
**only update it when the API returns a non-empty value**.

| Operation | Classification | Terraform mapping | Import | Destructive | Priority | Unresolved |
|---|---|---|---|---|---|---|
| Create/update/delete virtual | resource | `qualys_virtual_scanner` — exports `activation_code` (Computed+Sensitive, never overwritten with empty), `status`; VLANs/routes as whole-list attributes written on every update | appliance ID | delete deactivates schedules, alters asset groups | **P1** | — |
| List/details | data source | `data.qualys_scanner_appliances` | n/a | none | P1 | details params |
| Network assignment | part of resource (separate call) | attribute `network_id` on `qualys_virtual_scanner` | — | — | P1 | un-assign mechanics |
| Readiness wait | validation operation | optional `wait_for_online` timeout on resource / standalone helper | — | — | P2 | polling interval etiquette |
| Physical appliance mgmt | deferred | update-only; no create/delete API | — | — | deferred | — |
| Replace appliance | imperative operation | not a resource (it is a migration action, not desired state) — expose as a helper; Terraform models the *replacement* appliance declaratively | — | transfers config, removes old appliance from objects unless suppressed | P3 | — |

---

## 6. Option profiles (module: VMDR / Scans)

Doc section: Scans → Option Profiles. **Two parallel API surfaces:**

| Operation | Method | Path | Action | Notes | Status |
|---|---|---|---|---|---|
| Export (all/one) | GET/POST | `/api/2.0/fo/subscription/option_profile/` | `export` | `option_profile_id`/`option_profile_title`; whole-XML `OPTION_PROFILES` / `option_profile_info.dtd`; **Manager only** | Confirmed |
| Import | POST | `/api/2.0/fo/subscription/option_profile/` | `import` | body = `OPTION_PROFILES` XML (`text/xml`); returns new ID; **Manager only**; since ~8.10 | Confirmed |
| List (VM) | GET/POST | `/api/2.0/fo/subscription/option_profile/vm/` | `list` | full XML incl. `BASIC_INFO/ID`, `IS_DEFAULT`, `IS_GLOBAL`, `USER_ID`, scan sections | Confirmed |
| Create (VM) | POST | `/api/2.0/fo/subscription/option_profile/vm/` | `create` | **field-level form params** (`title`, `global`, `scan_tcp_ports=full`, `vulnerability_detection=complete`, ...) per `vm_op_params.htm` | Confirmed |
| Update (VM) | POST | `/api/2.0/fo/subscription/option_profile/vm/` | `update` | `id` (req) + `title` + create-params; `vm_op_params.htm` is the shared parameter reference for create and update | Confirmed (endpoint/action/`id`); a separate `update_vm_op.htm` page `Unverified` |
| Delete (VM) | POST | `/api/2.0/fo/subscription/option_profile/vm/` | `delete` | `id` | Confirmed (quick reference) |
| PC variants | — | `/subscription/option_profile/pc/` | `list/create/update/delete` | parallel | Confirmed |
| **Newer major versions** | — | `/api/4.0/fo/subscription/option_profile/vm/` (Release 10.36, Nov 2025); `POST /api/5.0/fo/subscription/option_profile/vm` and `GET,POST /api/7.0/fo/subscription/option_profile/` (Release 10.39.1, Jul 2026 — adds `allow_on_host_script_execution`) | — | **There is no `/api/3.0/` option-profile endpoint** — the version ladder is 2.0 → 4.0 → 5.0 → 7.0. (The "3.0" reference in `release_10_30_api_versioned.htm` is about the *Host Detection List* API, not option profiles.) | Existence Confirmed; 2.0→4.0/5.0 field/encoding delta `Unverified` |

**Confirmed field-level parameter names** (correcting several plausible-but-wrong guesses):

| Group | Parameters |
|---|---|
| Identity/scope | `title` (req on create), `id` (req on update), `owner`, `global` (0/1), **`default`** (0/1 — the IS_DEFAULT setter), `offline_scanner` |
| Ports | `scan_tcp_ports` (`none|full|standard|light`), `scan_tcp_ports_additional`, `scan_udp_ports`, `scan_udp_ports_additional`, `3_way_handshake`, `authoritative_option` |
| Dead hosts | `scan_dead_hosts`, **`close_vuln_on_dead_hosts`** (not `close_vulnerabilities`), `not_found_alive_times`, **`purge_host_data`** (not `purge`) |
| Performance | **`scan_overall_performance`** (`high|normal|low|custom`, not `overall_performance`), `scan_parallel_scaling`, `scan_external_scanners`, `scan_scanner_appliances`, **`scan_total_process`** (not `processes_to_run`), `scan_http_process`, **`scan_packet_delay`** (not `packet_delay`), `external_scanners_use` |
| Behaviour | **`load_balancer`** (not `load_balancer_detection`), **`enable_dissolvable_agent`** (not `dissolvable_agent`), `test_authentication`, `allow_on_host_script_execution` (10.39.1+) |
| Map settings | `perform_live_host_sweep`, `disable_dns_traffic`, `map_overall_performance`, `map_external_scanners`, `map_scanner_appliances`, `map_netblock_size`, `basic_information_gathering` (**a Map setting, not a scan setting**) |

**All previously-missing parameter names are now Confirmed** from the official Quick
Reference — full list in **[doc 11](11-verified-parameter-reference.md) §1**. The gap
closers:

- `vulnerability_detection={complete|custom|runtime}` (note the third value `runtime`)
  with `custom_search_list_ids`, `custom_search_list_title`, `exclude_search_list_ids`
- `password_brute_forcing_system={minimal|limited|standard|exhaustive}` and
  `password_brute_forcing_custom`
- **Authentication is a list, not per-technology booleans:** `authentication={value1,value2}`,
  plus `authentication_least_privilege=Unix`, `test_authentication`, and the system-auth
  trio (`include_system_auth`, `use_system_auth_on_duplicate`, `use_user_auth_on_duplicate`)
- `enable_additional_certificate_detection={0|1}`, `basic_host_information_checks`,
  `oval_checks`, `all_qrdi_checks`, `scan_intensity={normal|medium|low|minimum}`
- **`action=update` accepts every create parameter** (*"For other parameters see Create
  VM Option Profile"*) — so `default` and `global` are updatable and nothing is
  documented as immutable
- A third profile family exists alongside `/vm/` and `/pc/`: `/option_profile/pci/`

- **Managed lifecycle is viable — option profiles need not be read-only.** The viable
  Terraform schema is the *field-level* create/update/delete surface (form params),
  using `list` for read/drift. The XML import/export surface suits backup/migration
  (provider helper), not resource CRUD.
- **Known non-round-trippable fields (Confirmed, 8.10 release notes):** Password Brute
  Force Lists always import empty; search lists are matched by title with silent
  fallback of Vulnerability Detection to "Complete" → these fields must be
  `ignore_changes`-style suppressed or write-only in the schema.
- *Stable identifier:* numeric profile ID (in `BASIC_INFO/ID`).
- *Default designation:* settable via `default={0|1}` on **both create and update**.
  **Coupling hazard remains:** marking a profile default also forces it global, so
  `default` and `global` are not independent booleans — modelling them independently
  will produce perpetual drift.
- *Mutability:* `title` is mutable, and `update` accepts the full create parameter set,
  so **no option-profile field is ForceNew** on current evidence.

| Classification | Terraform mapping | Import | Destructive | Priority | Unresolved |
|---|---|---|---|---|---|
| resource (field-level API) + data source; XML import/export = provider helper | `qualys_vm_option_profile`; `data.qualys_option_profiles` | profile ID | delete breaks referencing scans/schedules | **P0–P1** (dependency of scans/schedules) | vulnerability-detection / auth-toggle / brute-force parameter names; `default`+`global` on update; 2.0→4.0/5.0 delta |

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
| List | GET/POST | `list` | `id`, **`active={0|1}`**, `show_notifications`, `client_id`, `client_name`, `echo_request` (also `scan_type`, `show_cloud_details`) | Confirmed — `SCHEDULE_SCAN_LIST_OUTPUT` / `schedule_scan_list_output.dtd`. The `active` filter is now confirmed; pagination model still `Unverified` |
| Create | POST | `create` | `scan_title`, `option_id|option_title`, `active=0|1`; targets/scanners as scan launch (`ip`, `asset_group_ids`, `target_from=tags` + tag params, `iscanner_name`, `priority`); recurrence: `occurrence=daily|weekly|monthly`, `frequency_days` (1–365), `frequency_weeks` (1–52) + **`weekdays={sunday|monday|…|saturday}` — NAMED days**, `frequency_months` (1–12) + (`day_of_month` 1–31 XOR `day_of_week` **0–6 numeric, 0=Sunday** + `week_of_month={first|second|third|fourth|last}`), **`recurrence`** = number of occurrences after which the schedule deactivates itself; time: **`start_date`**, `start_hour` 0–23, `start_minute` 0–59, `time_zone_code`, `observe_dst=yes|no`; duration cap (**not** an end date): `end_after` (0–119)+`end_after_mins` (0–59), `pause_after_hours` (1–119)+`pause_after_mins` (0–59), `resume_in_days` (1–9)+`resume_in_hours` (0–23); notifications: `before_notify`+`before_notify_unit` (`days|hours|minutes`)+`before_notify_time`+**`before_notify_message`**, `after_notify`+`after_notify_message`, `recipient_group_ids`; also `fqdn`, `runtime_http_header`, EC2 (`connector_name`, `connector_uuid`, `ec2_endpoint`, `ec2_only_classic`), **`client_id`/`client_name`** | Confirmed. **There is no `end_date` parameter.** ⚠️ **`weekdays` takes day *names* while `day_of_week` takes 0–6 — two different encodings in one API** |
| Update | POST | `update` | `id`, `echo_request`, `client_id`, `client_name`, + fields. **Time changes are gated:** any change to the start time requires `set_start_time=1` **plus all five** of `start_date`, `start_hour`, `start_minute`, `time_zone_code`, `observe_dst` sent together — update is not a field-by-field PATCH for this group | Confirmed |
| Delete | POST | `delete` | `id` | Confirmed |
| Enable/disable | POST | `update` with `active=0|1` | — | Confirmed |
| Launch schedule now | — | **does not exist** (no `action=launch` on schedule/scan; asymmetric with report schedules) | — | Confirmed absence |
| Legacy V1 | GET/POST | `/msp/scheduled_scans.php` | `add_task`/`drop_task`/`save_task` | legacy; **only** surface for scheduled *maps*; no formal deprecation notice found (`Unverified`) |

- *Timezones:* `time_zone_code` values from V1 helper `/msp/time_zone_code_list.php`
  (DTD `time_zone_code_list.dtd`); DST via `observe_dst`.
- *Recipients:* distribution groups only — **no free-form email addresses**; no
  distribution-group CRUD API found (doc 08).
- **`ACTIVE` is a four-valued enum, not a boolean:** `0` deactivated, `1` active,
  `2` active and not paused (*continuous schedules only*), `3` paused (*continuous
  schedules only*). Values 2/3 are newer than the 10.15 DTD. The provider must not map
  this to a Go `bool`; use an int, or normalise `{1,2}→true` / `{0,3}→false` and accept
  that pause state becomes invisible.
- **Server-side drift hazard:** a schedule using `recurrence` (occurrence count) sets
  itself `ACTIVE=0` once the count is exhausted. If `active` is a managed field, such
  schedules will show perpetual diffs — it needs drift suppression or a documented
  caveat.

| Classification | Terraform mapping | Import | Priority | Unresolved |
|---|---|---|---|---|
| resource + data source; legacy `/msp/scheduled_scans.php` = legacy (not used); timezone list = provider helper/validation | `qualys_scan_schedule` (start-time fields as one always-written block); `data.qualys_scan_schedules` | schedule ID | **P0–P1** (core of recurring-scan onboarding) | `before_notify_message`; `client_id/client_name` on create; list pagination/`active` filter |

---

## 9. Reports (module: VMDR / Reports)

Doc section: Reports. Path `/api/2.0/fo/report/`.

| Operation | Method | Action | Key params | Status |
|---|---|---|---|---|
| List | GET/POST | `list` | `id`, `state` (Running/Finished/Submitted/Canceled/Errors), `EXPIRATION_DATETIME` in output | Confirmed — `REPORT_LIST_OUTPUT` / `report_list_output.dtd`. A **v3 report endpoint exists** (`/api/3.0/fo/report/?action=fetch&id=` for downloading saved reports); v2 remains current and was updated as recently as Release 10.32. The 2.0→3.0 delta and whether v3 covers `launch`/`list` are `Unverified` |
| Launch | POST | `launch` | `template_id`, `report_title`, `output_format`, `report_type`, `pdf_password`, `recipient_group` (distribution groups, max 50). **Targeting depends on the template's findings mode, not the report type:** *Scan Based Findings* → `report_refs` is **required**; *Host Based Findings* → use `ips`, `ips_network_id`, `asset_group_ids` (or tag params) and not `report_refs` | Confirmed |
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
| `optionprofile` | `create`, `update`, `get`, `search` confirmed; `delete` confirmed via guide TOC | resource `qualys_was_option_profile` | P2 |
| `webauthrecord` | `create` confirmed (form/server/Selenium/OAuth2); `get` masks secrets without view-permissions; update/delete/search pattern-confirmed | resource `qualys_was_auth_record` (write-only secrets) | P2 |
| `wasscan` | `launch`, `search`, `status`, `download` confirmed. **`cancel/was/wasscan/<id>` confirmed** — takes optional boolean `cancelWithResults` to retain partial results; supported for single and *child* scans only (not parent scans), and recommended only after ~20 minutes in Running status. **`delete/was/wasscan` confirmed** as a *filtered bulk* delete (`<filters><Criteria field="id" operator="IN">` with comma-separated IDs) | imperative operation | P3 |
| `wasscanschedule` | **Full CRUD confirmed** (count, search, get, create single + multiple, update, activate, delete, download-iCalendar) **and the element model is now confirmed** — `name`, `target.webApp.id` / `target.webApps.id` / `target.tags.id` (+`target.tags.included.option={ALL\|ANY}`), `type={DISCOVERY\|VULNERABILITY}`, `profile.id`, `startDate`, `timeZone`, **`occurrenceType={ONCE\|DAILY\|WEEKLY\|MONTHLY}`**, `notification`, `reschedule`, plus optional scanner/auth/proxy/cancel options. See [doc 11](11-verified-parameter-reference.md) §8. Only the per-occurrence frequency sub-elements (every N days/weeks) remain `Unverified` | resource `qualys_was_scan_schedule` | P2–P3 |
| `report` / report templates | `create`, `get` (status), `download` confirmed; search template endpoint confirmed; delete pattern-confirmed | imperative operation + data source | P3 |
| report schedules | **No v3 API — UI-only.** However Qualys announced (13 Apr 2026) **new V4 API endpoints for schedule and report management, explicitly covering scheduled reports**, with V3 endpoints unchanged. V4 paths/schemas `Unverified` (`/qps/rest/4.0/...` inferred, not confirmed) | deferred — re-scope once V4 reference docs are available | deferred |
| `optionprofile` delete | `delete/was/optionprofile/<id>` present in the WAS API guide TOC ("Delete an Option Profile"); literal path not printed on a retrieved page | — | — |

Auth: Basic per request (baseline); JWT "token-based auth" documented in a technical
brief — per-endpoint WAS support `Unverified`. Encoding: XML default; JSON since
WAS 4.5 with both `Accept` and `Content-Type: application/json`. Pagination:
`hasMoreRecords`/`lastId` + Criteria `GREATER`.

---

## 14. AssetView / CSAM tagging (module: AM, `/qps/rest/2.0/.../am/...`)

| Capability | Operation | Status |
|---|---|---|
| Tag CRUD | `create/search/get/update/delete/count /am/tag` (with and without `/<id>`), plus **`evaluate/am/tag/<id>`** to re-evaluate a dynamic tag | Confirmed |
| Host-asset activation | **`activate/am/hostasset[/<id>]?module=QWEB_VM` or `?module=QWEB_PC`** — how an asset is activated into the VM or PC module; `delete/am/hostasset` also exists | Confirmed (newly surfaced) |
| Static tags | `ruleType` omitted/STATIC | Confirmed |
| Dynamic tags | `ruleType`: GROOVY, OS_REGEX, NETWORK_RANGE, NAME_CONTAINS, INSTALLED_SOFTWARE, OPEN_PORTS, VULN_EXIST, ASSET_SEARCH, GLOBAL_ASSET_VIEW + `ruleText` | Confirmed (`CLOUD_ASSET` `Unverified`) |
| Hierarchy | `parentTagId` on create; `children.set.TagSimple[]`; **never add and remove children in one call** | Confirmed |
| Assign tags to host assets | `POST /qps/rest/2.0/update/am/hostasset` with `tags.add.TagSimple{id}` / `tags.remove`; bulk via Criteria `id IN` | Confirmed |
| `update/am/asset` variant | static tags only — **dynamic tags rejected** on add/remove/set | Confirmed |
| List assets by tag | `search/am/hostasset` Criteria `tagName`/`tagId EQUALS`. Full documented filter set: `qwebHostId`, `id`, `name`, `created`, `updated`, `tagName`, `tagId`, **`lastVulnScan`**, `lastComplianceScan`, `informationGatheredUpdated`, `os`, `dnsHostName`, `netbiosName`, `netbiosNetworkID`, `trackingMethod`, `port`, `installedSoftware` | Confirmed (`lastVulnScan` now confirmed) |
| Untagged-asset discovery | **Negation is supported on the gateway surface**: `POST gateway.<pod>.apps.qualys.com/rest/2.0/search/am/asset` accepts QQL with `not tags.name:"X"`. `NOT_EQUALS` is documented as an operator for `tags.name`. Additionally the qps filter language **does have a `NONE` operator** — the Quick Reference documents WAS filters `webApp.tags` *with `operator="NONE"`* and `lastScan` *with `operation="NONE"`*, i.e. "has no tags at all" is expressible there. **Client-side set difference is therefore not required for "assets lacking tag X"** | Confirmed (gateway QQL negation; `NONE` operator exists in qps filters). Whether `NONE`/`NOT_EQUALS` applies to `tagName` on the *classic* `/qps/rest/2.0/search/am/hostasset` endpoint specifically is still `Unverified` |
| WAS webapp tag assignment | `TagList` on `webapp` create/update; WAS search supports `webApp.tags.id` / `webapp.tags.name` | Confirmed |
| **Cross-module tag identity** | **Now Confirmed — one subscription-wide tag tree.** (a) *VMDR*: tags are chosen from the same tree that AssetView/CSAM manages — *"a benefit of the tag tree is that you can assign any tag in the tree to a scan or report"*, and *"All users can see all tags in the subscription and can choose tags for their scans and reports"*, so `tag_set_include`/`tag_set_by=id` on scan/report launch resolve against tags created by `create/am/tag`. (b) *WAS*: *"Tags are defined through the CyberSecurity Asset Management (CSAM) application"* — WAS does not have a private tag namespace, and its `TagList` carries the same numeric tag `id`. **Caveat to carry forward:** the AM&T API v2 guide lists its own supported modules as VM, PC, SCA, CERTVIEW, CLOUDVIEW (WAS absent) — this reads as the scope of the *AM API surface* rather than a namespace split, since the WAS-side docs independently source their tags from CSAM, but no single official sentence asserts the full round trip in one breath | **Confirmed** for both (a) and (b), with the `modules`-list caveat noted |

| Classification | Terraform mapping | Import | Priority |
|---|---|---|---|
| resource + data source | `qualys_asset_tag` (static/dynamic, hierarchy); `qualys_host_tag_assignment`; `data.qualys_asset_tags`, `data.qualys_tagged_assets` | tag ID | **P0–P1** (tag-based scan/report targeting depends on it) |

---

## 14a. Capability areas surfaced late — not yet inventoried in detail

Source: compiled reference of the VM/PC API guide (derived secondary source). These are
recorded so the inventory is not silently incomplete; none has been worked through to
parameter level, and each needs its own pass before implementation.

| Area | Endpoint / notes | Classification (provisional) |
|---|---|---|
| **VM remediation tickets** | `/api/2.0/fo/ticket/` — list, edit, delete, view deleted, get detail, and *set vulnerabilities to ignore on hosts* | data source + imperative; the ignore operation is relevant to the stale-asset subsystem |
| Scan schedule run history | `/api/2.0/fo/scan/schedules/runhistory/?action=list` — up to 500 schedule IDs, 1–50 executions each; **not available for map/discovery schedules** | data source — useful for verifying schedules actually fire |
| Static / dynamic search lists | `/api/2.0/fo/qid/search_list/{static\|dynamic}/` — full CRUD. **Direct dependency of option profiles** (`custom_search_list_ids`) | **resource** — promote to P1, since option profiles reference these by ID |
| Restricted IPs | list/manage — IPs blocked from scanning regardless of targeting (a safety guardrail). Documented as accepting **CIDR** (`action=replace&ips=10.0.0.0/8`) | resource |
| Virtual hosts | `/api/2.0/fo/asset/vhost/` — multiple hostnames on one IP, scoping scans per hostname | resource |
| IPv6 mapping records | `/api/2.0/fo/asset/ip/v4_v6/` — list/add; IPv6 addresses are not stable scan targets the way IPv4 are | data source + resource |
| Patch list | per-host patches relevant to detections (`host_id`, `output_format=xml`) | data source |
| Cloud perimeter scan jobs | `/api/2.0/fo/scan/cloud/perimeter/job/` (v2.0–v4.0) — create/update/reset | deferred |
| Cloud internal scan jobs | `/api/2.0/fo/scan/cloud/internal/job/` — Azure and GCP internal scans | deferred |
| SCAP scans | `/api/2.0/fo/scan/scap/?action=list` — list only | deferred |
| Containerized scanner appliances | create/list/update/delete — scanners as containers (Kubernetes/ephemeral) | **candidate resource** — may fit this product better than virtual appliances |
| Activity log | `/api/2.0/fo/activity_log` — auditable record of every API call; filter by process state (Queued, Running, Expired, Finished, Blocked) | data source |
| Vendor IDs / references, editing vulnerabilities | QID→vendor advisory mapping; local QID metadata overrides | deferred |
| **Policy Compliance (whole module)** | controls, policies, export/import/merge, framework reports, posture (`/pcrs/` streaming), exceptions, control criticality, SCAP reports, and the PCAS policy-authoring and library trees | **out of scope for this product** unless PC is explicitly in remit — recorded, not planned |

⚠️ **Conflict on the user API.** This source gives a base path family of
`/api/2.0/fo/user/`, while the earlier pass found only V1 `/msp/user.php` documented
(the V2 path was a third-party claim). `Unverified` — low priority, since `qualys_user`
is out of scope.

## 15. TotalCloud / CloudView (module: CloudView — minimal retained scope)

| Capability | Endpoint | Status | Classification |
|---|---|---|---|
| GCP connector CRUD (existing provider) | `/cloudview-api/rest/v1/gcp/connectors` (POST/GET/PUT/DELETE; JSON; Basic; `pageNo`/`pageSize`) | Confirmed | resource + data source (existing `qualys_gcp_connector`) |
| Connector v3 replacement | `/qps/rest/3.0/create|run|delete/am/gcpassetdataconnector` (+ AWS/Azure) | Confirmed | **migration target** — CloudView connector APIs deprecated; plan migration of the existing resource (doc 07/09) |
| All other TC/CloudView operations | — | out of scope by explicit product decision | unsupported (out of scope) |
