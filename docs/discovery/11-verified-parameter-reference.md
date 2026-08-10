# 11 — Verified Parameter Reference

Parameters below were read from the **Qualys API Quick Reference Guide** (official PDF,
supplied directly for this discovery pass). Everything in this document is therefore
`Confirmed` against official documentation unless a line says otherwise.

> Source note: the Quick Reference is Qualys copyright material. It is cited here and
> its parameter names are recorded for implementation, but the extracted document text
> is deliberately **not** committed to this repository.

This document supersedes any conflicting parameter detail in doc 03; doc 03 rows have
been updated to match.

## 1. VM option profiles — `/api/2.0/fo/subscription/option_profile/vm/`

Actions: `create` (POST), `update` (POST, `id` + *"For other parameters see Create VM
Option Profile"* — i.e. **every create parameter is updatable, including `default` and
`global`**), `list` (GET+POST), `delete` (GET+POST, `id`).

**Identity / scope:** `title`, `owner`, `default={0|1}`, `global={0|1}`,
`offline_scanner={0|1}`

**Scan — ports & handshake:** `scan_tcp_ports={none|full|standard|light}`,
`scan_tcp_ports_additional={port1,port2}`, `scan_udp_ports={none|full|standard|light}`,
`scan_udp_ports_additional`, `3_way_handshake={0|1}`, `authoritative_option={0|1}`

**Scan — duration & dead hosts:** `enable_max_scan_duration_per_asset={0|1}`,
`max_scan_duration_per_asset_minutes`, `scan_dead_hosts={0|1}`,
`close_vuln_on_dead_hosts={0|1}`, `not_found_alive_times`, `purge_host_data={0|1}`

**Scan — performance:** `external_scanners_use`, `scan_parallel_scaling={0|1}`,
`scan_overall_performance={high|normal|low|custom}`, `scan_external_scanners`,
`scan_scanner_appliances`, `scan_total_process`, `scan_http_process`,
`scan_packet_delay={minimum|short|medium|long|maximum}`,
`scan_intensity={normal|medium|low|minimum}`, `load_balancer={0|1}`

**Detection (the previously-missing block):**
- `vulnerability_detection={complete|custom|runtime}` — note the third value **`runtime`**
- `custom_search_list_ids={value1,value2}`, `custom_search_list_title={value1,value2}`,
  `exclude_search_list_ids={value1,value2}`
- `password_brute_forcing_system={minimal|limited|standard|exhaustive}`,
  `password_brute_forcing_custom={value1,value2}`
- `basic_host_information_checks={0|1}`, `oval_checks={0|1}`, `all_qrdi_checks={0|1}`
- `enable_additional_certificate_detection={0|1}`

**Authentication (not per-technology booleans — a list):**
- `authentication={value1,value2}` — the enabled authentication technologies
- `authentication_least_privilege=Unix`
- `test_authentication={0|1}`
- System auth: `include_system_auth={0|1}`, `use_system_auth_on_duplicate={0|1}`,
  `use_user_auth_on_duplicate={0|1}`

**Other scan behaviour:** `enable_dissolvable_agent={0|1}`,
`enable_windows_share_enumeration={0|1}`, `enable_lite_os_scan={0|1}`,
`enable_partial_ssl_tls_auditing={0|1}`, `custom_http_header`,
`custom_http_definition_key`, `custom_http_definition_header`, `host_alive_testing={0|1}`,
`not_overwrite_os={0|1}`, `ignore_firewall_generated_tcp_rst_packets={0|1}`,
`ignore_all_tcp_rst_packets={0|1}`,
`ignore_firewall_generated_tcp_syn_ack_packets={0|1}`,
`not_send_tcp_ack_or_syn_ack_packets_during_host_discovery={0|1}`

**Map block:** `basic_information_gathering=[all|register|netblockonly|none]`,
`map_tcp_ports_standard_scan={0|1}`, `map_tcp_ports_additional`,
`map_udp_ports_standard_scan={0|1}`, `map_udp_ports_additional`,
`perform_live_host_sweep={0|1}`, `disable_dns_traffic={0|1}`,
`map_overall_performance={high|normal|low|custom}`, `map_external_scanners`,
`map_scanner_appliances`, `map_netblock_size={1024|4096|8192|16384|32768|65536 IPs}`,
`map_packet_delay={minimum|short|medium|long|maximum}`,
`map_authentication={VMware|vCenter}`

**Additional ports block:** `additional_tcp_ports={0|1}`,
`additional_tcp_ports_standard_scan={0|1}`, `additional_tcp_ports_additional`,
`additional_udp_ports={0|1}`, `additional_udp_ports_type={standard|custom}`,
`additional_udp_ports_custom`, `icmp={0|1}`, `blocked_resources={0|1}`,
`protected_ports={default|custom}`, `protected_ports_custom`,
`protected_ips={all|custom}`, `protected_ips_custom`

**Sibling profile families:** `/option_profile/pci/` (PCI, create/update/list/delete)
and `/option_profile/pc/` (compliance, with DB UDC restriction/limit parameters per
engine, `enable_auth_instance_discovery`, `auto_auth_types`, `ibm_was_discovery_mode`,
`oracle_template_id`/`_name`, instance data collection parameters).

**Export/import:** `/option_profile/?action=export` (`output_format=XML`,
`option_profile_id`, `option_profile_title`, `option_profile_type={user|compliance|pci}`)
and `?action=import` (XML body, `Content-Type: XML`).

## 2. Scanner appliances — corrections

- **Replace Appliance IS an API operation** (previously recorded as UI-only):
  `POST /api/2.0/fo/appliance/replace_iscanner/` with `action={replace}`,
  `old_scaner_name` *(sic — the Quick Reference spells it with this typo; verify the
  exact spelling against a tenant)*, `new_scanner_name`, `do_not_copy_settings={0|1}`,
  `do_not_remove_new_scanner_from_objects={0|1}`, `echo_request`.
- `polling_interval={60-360}`, default 180 — the Quick Reference range **conflicts**
  with a UI help page stating 60–3600. Treat 60–360 as authoritative for the API.
- Physical appliance update: `/api/2.0/fo/appliance/physical/` `action={update}`, `id`,
  `name`, `polling_interval`, `set_vlans`, `set_tags`/`add_tags`/`remove_tags`,
  `tag_set_by`, `comment`.
- Assign to network: `action={assign_network_id}`, `appliance_id`, `network_id`.

## 3. Host assets — `/api/2.0/fo/asset/ip/`

**Add IPs (`action=add`):** `ips={value}` **or POSTed CSV raw data**, `echo_request`.

**Host Update (`action=update`) — selector vs setter split (resolves the earlier
conflict):**
- *Selectors:* `ips`, `ids`, `ag_ids`, `ag_titles`, `network_id`, `network_name`,
  `tracking_method`, `host_dns`, `host_netbios`
- *Setters:* **`new_tracking_method`, `new_owner`, `new_ud1`, `new_ud2`, `new_ud3`,
  `new_comment`**

  The bare names are **selectors, not setters**. A client sending bare `owner`/`ud1`
  to change values is silently filtering rather than updating — this is exactly the
  trap the earlier third-party-source conflict pointed at.

**Purge (`action=purge`):** `ips`, `ids`, `ag_ids`, `ag_titles`, `no_vm_scan_since`,
`no_compliance_scan_since`, `data_scope={vm|pc|vm,pc}`, `compliance_enabled={0|1}`,
`os_pattern={PCRE regex}`, `network_ids`, `echo_request`.
Documented note: *"If `compliance_enabled=1` is specified in the same request as
`data_scope`, then vulnerability and compliance data will both be purged regardless of
the `data_scope` value."*

**Excluded hosts:** `/api/2.0/fo/asset/excluded_ip/` `action={add|remove|remove_all}`,
`ips`, `comment`, `expiry_days` (add), `dg_names` (add), `network_id`
(*"`ips` is invalid for `remove_all`"*). **Change history endpoint confirmed:**
`/api/2.0/fo/asset/excluded_ip/history/` `action={list}`, `ips`, `ids`, `id_min`,
`id_max`, `network_id`.

## 4. Asset groups — `/api/2.0/fo/asset/group/`

**Add (`action=add`, bare names):** `title`, `network_id`, `comments`, `division`,
`location`, `function`, `business_impact={critical|high|medium|low|none}`, `ips`,
`appliance_ids`, `default_appliance_id`, `domains`, `dns_names`, `netbios_names`,
`cvss_enviro_cdp={high|medium-high|low-medium|low|none}`,
`cvss_enviro_td={high|medium|low|none}`, `cvss_enviro_cr|ir|ar={high|medium|low}`

**Edit (`action=edit`, `id` + edit-only names):** `set_title`, `set_comments`,
`set_division`, `set_location`, `set_function`, `set_business_impact`,
`add|remove|set_ips`, `add|remove|set_appliance_ids`, `set_default_appliance_id`,
`add|remove|set_domains`, `add|remove|set_dns_names`, `add|remove|set_netbios_names`,
`set_cvss_enviro_cdp|td|cr|ir|ar`

**Resolved:** `business_impact` enum tail is **`none`**, not "Minor". There is **no
`set_network_id` and no `set_owner`** — network and owner are create-time only.

**List:** `ids`, `id_min`, `id_max`, `truncation_limit`, `network_ids`, `unit_id`,
`user_id`, `show_attributes={None|All|comma-separated list of TITLE, OWNER,
OWNER_USER_NAME, NETWORK_IDS, LAST_UPDATE, IP_SET, APPLIANCE_LIST, DOMAIN_LIST,
DNS_LIST, NETBIOS_LIST, EC2_ID_LIST, HOST_IDS, USER_IDS, UNIT_IDS, BUSINESS_IMPACT,
CVSS, COMMENTS}`

## 5. Networks — `/api/2.0/fo/network/`

`Network List (GET+POST)`: `action={list}`, `echo_request`, `ids`.
`Network (POST)`: **`action={create|update}`**, `name`, `echo_request`.
**No `delete` action exists** — now confirmed from the official action enumeration.

## 6. Scan schedules — `/api/2.0/fo/schedule/scan/`

**List:** `action={list}`, `id`, **`active={0|1}`**, `show_notifications={0|1}`,
`client_id`, `client_name`, `echo_request`.

**Create:** `scan_title`, `active={0|1}`, `option_title`/`option_id`, targets
(`target_from={assets|tags}`, `ip`, `asset_groups`, `asset_group_ids`,
`exclude_ip_per_scan`, `tag_include_selector`, `tag_exclude_selector`, `tag_set_by`,
`tag_set_include`, `tag_set_exclude`, `use_ip_nt_range_tags[_include|_exclude]`,
`ip_network_id`), scanners (`iscanner_id`, `iscanner_name`, `default_scanner`,
`scanners_in_ag`, `scanners_in_tagset`, `scanners_in_network`), EC2
(`connector_name`, `connector_uuid`, `ec2_endpoint`, `ec2_only_classic`),
`runtime_http_header`, `fqdn`, **`client_id`/`client_name` (confirmed on create)**.

*Recurrence:* `occurrence={daily|weekly|monthly}`, `frequency_days` (1–365),
`frequency_weeks` (1–52), **`weekdays={sunday|monday|tuesday|wednesday|thursday|friday|
saturday}`** — **named days, not numeric** — `frequency_months` (1–12), `day_of_month`
(1–31), `day_of_week` (**0–6, 0 = Sunday** — numeric here, unlike `weekdays`),
`week_of_month={first|second|third|fourth|last}`, `recurrence={value}`.

*Time:* `start_date`, `start_hour` (0–23), `start_minute` (0–59), `time_zone_code`,
`observe_dst={yes|no}`.

*Duration:* `end_after` (0–119) + `end_after_mins` (0–59); `pause_after_hours` (1–119)
+ `pause_after_mins` (0–59); `resume_in_days` (1–9) + `resume_in_hours` (0–23).
Required-together rules are documented explicitly.

*Notifications:* `before_notify={0|1}`, `before_notify_unit={days|hours|minutes}`,
`before_notify_time`, **`before_notify_message`** (confirmed to exist),
`after_notify={0|1}`, `after_notify_message`, `recipient_group_ids`.

**Update:** `action={update}`, `id`, `set_start_time={0|1}`, `client_id`, `client_name`
— with the documented rule that start-time changes require `set_start_time=1` plus
`start_date`, `start_hour`, `start_minute`, `time_zone_code`, `observe_dst` together.
**Delete:** `action={delete}`, `id`.

## 7. Tagging — `/qps/rest/2.0/.../am/...`

Tag CRUD confirmed: `get|create|update|search|count|delete /am/tag` (with and without
`/<id>`), plus **`evaluate/am/tag/<id>`** (re-evaluate a dynamic tag — newly surfaced).

Tag search filters: `id`, `name`, `parent`, `ruleType` (STATIC, GROOVY, OS_REGEX,
NETWORK_RANGE, NAME_CONTAINS, INSTALLED_SOFTWARE, OPEN_PORTS, VULN_EXIST,
ASSET_SEARCH), `color` (`#FFFFFF` format).

Host assets: `get|create|update|search|count|delete /am/hostasset`, plus
**`activate/am/hostasset[/<id>]?module=QWEB_VM|QWEB_PC`** (newly surfaced — this is how
an asset is activated into the VM or PC module).

Host-asset search filters include `qwebHostId`, **`lastVulnScan` (date)** (previously
`Unverified`), `lastComplianceScan`, `informationGatheredUpdated`, `os`, `dnsHostName`,
`netbiosName`, `netbiosNetworkID`, `trackingMethod`, `port`, `installedSoftware`.

## 8. WAS scan schedules — `/qps/rest/3.0/.../was/wasscanschedule`

Operations: `count`, `search`, `get/<id>`, `create` (single web app **and** multiple),
`update/<id>`, `activate/<filters>`, deactivate (via update), `delete/<id>` or
`delete/<filters>`, `download/<id>|<filters>` (iCalendar).

**Element model (this was the last blocking gap):**
- *Required:* `name`, `target.webApp.id` (single) or `target.webApps.id` /
  `target.tags.id` (multiple) with `target.tags.included.option={ALL|ANY}` and
  `target.tags.included.tagList.Tag.id`, `type={DISCOVERY|VULNERABILITY}`, `profile.id`
  (*required unless the target has a default option profile*), **`startDate`**,
  **`timeZone`**, **`occurrenceType={ONCE|DAILY|WEEKLY|MONTHLY}`**,
  `notification` (Boolean), `reschedule` (Boolean)
- *Optional:* `target.authRecordOption`, `target.profileOption`, `target.scannerOption`,
  `target.randomizeScan`, `target.scannerAppliance.type={EXTERNAL|INTERNAL|scannerTags}`,
  `target.scannerAppliance.friendlyName`, `target.webAppAuthRecord.id` /
  `target.webAppAuthRecord.isDefault`, `proxy.id`, `dnsOverride.id`,
  `cancelOption={DEFAULT|SPECIFIC}`, `sendMail` (Boolean)

Per-occurrence frequency sub-elements (every N days/weeks/months) are **not** enumerated
in the Quick Reference — this section's summary of `occurrenceType`, `startDate` and
`timeZone` as flat top-level elements also turned out to be incomplete in a more
consequential way: a later user-supplied "Create and Update Scan Schedule" walkthrough
showed all three actually nest under a `scheduling` sub-object (`timeZone` itself
wrapping a `.code`), and `cancelOption` nests under `target`, not top-level. The Quick
Reference's element list named the right elements, just not their true payload
structure — the same class of gap the WAS auth record's fields-list-vs-flat-elements
correction hit. `qualys_was_scan_schedule` was rebuilt against the walkthrough: it now
covers `DAILY`/`WEEKLY` recurrence (`everyNDays`; `everyNWeeks`/`onDays`/
`occurrenceCount`), `cancelAfterNHours`, a full `notification` sub-object, `sendMail`/
`sendOneMail`/`sendMailFromAddressOption`, and confirms activate/deactivate as dedicated
endpoints. `MONTHLY` recurrence detail remains open — no source has shown a
`monthlyOccurrence` example for this object specifically. (A later pass briefly
modelled it anyway, by analogy with `qualys_was_report_schedule`'s confirmed MONTHLY
shape; a user-supplied "Gap Review" document explicitly rejected that inference —
"do not derive the WasScanSchedule MONTHLY payload solely from report scheduling" —
and it was reverted. `qualys_was_scan_schedule` does not accept MONTHLY.) Tag-based
multi-web-app targeting is still not modelled — the same document confirms the shape
exists for one-off WAS Multi-Scan launches, but not that this object accepts it. The
JSON wrapper key `WasScanSchedule` is now Confirmed, appearing verbatim in the
walkthrough.

**Notable for untagged discovery:** WAS schedule/scan search filters document
`webApp.tags` **with `operator="NONE"`**, and `lastScan` with `operation="NONE"` —
so a `NONE` operator does exist in the qps filter language.

Also confirmed: `cancel/was/wasscan/<id>`, `download/was/wasscan/<id>` (3.0 and a 2.0
variant), `status/was/wasscan/<id>`.

## 9. WAS API — wider surface

Source: a compiled reference derived from the WAS API documentation tree
(`docs.qualys.com/en/was/api/...`), supplied for this discovery. It is a **derived
secondary source**, not the official guide, so items below are
`Corroborated (non-official)` unless the Quick Reference independently confirms them —
and where the two disagree, the Quick Reference wins (see the conflicts at the end).

### Objects beyond those already inventoried

| Object | Base path | Operations |
|---|---|---|
| Catalog (discovered/candidate assets not yet promoted to WebApps) | `/qps/rest/3.0/<op>/was/catalog[/<id>]` | count, search, get, update (single + bulk by filter), delete |
| Finding | `/qps/rest/3.0/<op>/was/finding[/<id>]` | count, search, get, **ignore**, **activate**, **updateSeverity**, **restoreSeverity**, **retest**, retest status |
| DNS override record | `/qps/rest/3.0/<op>/was/dnsoverriderecord[/<id>]` | count, search, get, create, update, delete |
| Search lists / parameter sets | `/qps/rest/3.0/<op>/was/searchlist`, `.../was/parameter` | same CRUD pattern |
| Report template | `/qps/rest/3.0/<op>/was/report/template[/<id>]` | count, search, get |
| Burp import | `/qps/rest/3.0/import/was/burp/<webapp_id>` | multipart upload of a Burp XML export; imported issues surface with `findingType=BURP` |

WebApp also exposes **`purge/was/webapp/<id>`** and a `removeFromSubscription` flag on
delete, plus `swaggerFile` (OpenAPI upload), `progressiveScanning`, malware
monitoring/notification/scheduling, `config.defaultAuthRecord`, `config.cancelScansAt`,
`crawlingScripts` (Selenium), and custom `attributes` (key/value, filterable via
`field="attributes"` + `name="..."`).

WAS option profile fields: `maxCrawlRequests`, performance (`LOW|MEDIUM|HIGH`),
`bruteforceOption`, `timeoutErrorThreshold`, `unexpectedErrorThreshold`,
`sensitiveContent.creditCardNumber`, `sensitiveContent.socialSecurityNumber`.

Finding filters: `id`, `qid`, `webApp.id`/`webApp.tags`, `url`, `severity`,
`status={NEW|ACTIVE|REOPENED|FIXED|PROTECTED}`,
`type={VULNERABILITY|SENSITIVE_CONTENT|INFORMATION_GATHERED}`,
`findingType={QUALYS|MANUAL|BURP}`, `firstDetectedDate`, `lastDetectedDate`,
`isIgnored`, `cvssV3.base`, `cwe`.

Scan status values: `SUBMITTED`, `RUNNING`, `FINISHED`, `TIME_LIMIT_EXCEEDED`,
`SCAN_NOT_LAUNCHED`, `SCANNER_NOT_AVAILABLE`, `ERROR`, `CANCELED`.
Scan `type`: `VULNERABILITY`, `DISCOVERY`, `SCA`.

### qps error model — closes not-found detection for this family

`ServiceResponse/responseCode` values, with
`responseErrorDetails/errorMessage` + `errorResolution` on failure:

| `responseCode` | Meaning |
|---|---|
| `SUCCESS` | completed normally |
| `INVALID_REQUEST` | malformed payload, bad field, or feature not enabled for the subscription |
| `UNAUTHORIZED` / `INSUFFICIENT_PERMISSION` | bad credentials, expired token, or missing WAS asset permission |
| **`OBJECT_NOT_FOUND`** | the `<id>` does not exist **or is not in the caller's scope** |
| `LIMIT_EXCEEDED` | report storage limit, or API rate/concurrency limit |

**This is the not-found signal for the qps families (WAS and AM/tagging)** — the
provider can map `OBJECT_NOT_FOUND` to "remove from state". Note the scope caveat: an
object that exists but is outside the caller's scope returns the same code, so a
permissions problem can masquerade as a deletion. The equivalent for the VMDR FO family
is still missing (see below).

### Pagination detail

Beyond `hasMoreRecords`/`lastId`, search calls accept a `<preferences>` block:
`startFromOffset` (default 1), `startFromId`, `limitResults` (default 100, **max 1000**).
Filter operators: `EQUALS`, `NOT EQUALS`, `CONTAINS`, `GREATER`, `LESSER`, `IN`, `NONE`
— not every operator is valid on every field.

### Useful for the tenant capability probe

- **`GET /qps/rest/portal/version`** returns installed Portal and per-module versions
  (WAS, VM, CA, FIM…). This is a better probe primitive than the cheap-list calls
  proposed in doc 08 — it feature-gates without touching customer data.
- `X-Powered-By` response header (`Qualys:<POD_ID>:<SUB_UUID>:<USER_UUID>`) can be
  enabled by Qualys Support for per-subscription call tracing.

### ⚠️ Conflicts with the Quick Reference — Quick Reference treated as authoritative

1. **"Scan again"** — the derived reference gives `POST /launch/was/wasscan/<id>`; the
   Quick Reference gives **`/qps/rest/3.0/scanagain/was/scan/{scanId}`**, and `qualysdk`
   independently implements `scanagain/was/scan/{scanId}`. Two independent sources beat
   one; use `scanagain`, but verify on a tenant.
2. **Token auth endpoint** — the derived reference gives `/auth/oidc` with
   `client_id`/`client_secret`/`grant_type=client_credentials`; earlier official
   material gives `/auth/oauth` and the gateway `/auth`. These may be different flows
   (OIDC client-credentials vs Qualys-native) rather than a contradiction. `Unverified`
   — resolve before implementing token auth for WAS.
3. **Schedule deactivate** — the derived reference gives a dedicated
   `/deactivate/was/wasscanschedule/<id>`; the Quick Reference shows activate via
   `update`/`activate` and lists a "Deactivate an existing schedule" heading. Both
   plausibly exist. `Unverified`.

### Still open for WAS

The **per-occurrence recurrence sub-elements** remain unresolved. The derived reference
describes only "an `occurrence` block supporting daily/weekly/monthly patterns (mirrors
the `malwareScheduling` occurrence structure)" — that names the block but not the
every-N-days/weeks/months fields inside it. `occurrenceType={ONCE|DAILY|WEEKLY|MONTHLY}`,
`startDate` and `timeZone` are confirmed (§8); the frequency fields still need the
official WAS guide or a tenant probe.

## Still not answered by the Quick Reference

The Quick Reference contains **no error/response code table and no API-limit numbers** —
those live in the *Qualys API (VM and PA) User Guide* appendices. So these remain open:

- the numeric "object not found" code and the full v2 error-code table;
- the per-service-level rate/concurrency limits table;
- whether `ips` accepts CIDR (the Quick Reference writes `{ip,range…}` throughout,
  consistent with hyphenated ranges only, but never states the negative).
