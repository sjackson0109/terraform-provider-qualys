# 06 — Candidate Terraform Resources, Data Sources and Imperative Operations

Deliverables 4, 5 and 6. Grounded in the per-operation classifications of doc 03.

## Deliverable 4 — Candidate Terraform resources

| Resource | API surface | CRUD notes | Priority |
|---|---|---|---|
| `qualys_asset_group` | `/api/2.0/fo/asset/group/` add/edit/delete | authoritative updates via `set_*`; exports group ID | **P0** |
| `qualys_asset_tag` | `qps/rest/2.0/.../am/tag` | static + dynamic (`ruleType`/`ruleText`), hierarchy via `parentTagId`; two-call child add/remove. **One tag object serves VMDR targeting, AssetView and WAS** (confirmed), so this is the single cross-family targeting primitive | **P0** |
| `qualys_vm_option_profile` | `/api/2.0/fo/subscription/option_profile/vm/` list/create/update/delete | field-level params; suppress non-round-trippable fields (brute-force lists, search-list titles) | **P0–P1** |
| `qualys_scan_schedule` | `/api/2.0/fo/schedule/scan/` create/update/delete/list | full recurrence/time/notify model; `active` toggle | **P0–P1** |
| `qualys_ip_registration` | `/api/2.0/fo/asset/ip/` add/update | **no delete API** — destroy is state-only (optional opt-in purge); CIDR expanded client-side | P1 |
| `qualys_network` | `/api/2.0/fo/network/` create/update | **no delete API** — destroy warns + removes from state; network declared explicitly (workbook rule) | P1 |
| `qualys_virtual_scanner` | `/api/2.0/fo/appliance/` create/update/delete + `assign_network_id` | exports `activation_code` (sensitive) for the infra provider deploying the VM; optional readiness wait | P1 |
| `qualys_host_tag_assignment` | `qps/rest/2.0/update/am/hostasset` tags.add/remove | assignment object (host ↔ tag set) | P1 |
| `qualys_auth_record_windows` / `qualys_auth_record_unix` (then further types) | `/api/2.0/fo/auth/<type>/` | **write-only** credential arguments; vault reference support; Windows domain type forces replacement | P1–P2 |
| `qualys_vault` | `/api/2.0/fo/vault/` | per-type config; write-only secrets | P2 |
| `qualys_domain` | `/api/2.0/fo/asset/domain/` | Basic-auth-only client path | P2 |
| `qualys_excluded_ips` | `/api/2.0/fo/asset/excluded_ip/` add/remove | supports `expiry_days` | P3 |
| `qualys_web_application` | `qps/rest/3.0/.../was/webapp` | JSON/XML ServiceRequest; TagList association. **`auth_record_ids`** added: a user-supplied example shows auth-record association as its own `authRecords.add`/`.remove` call, an incremental idiom distinct from tags' "set" | P2 |
| `qualys_was_option_profile` | `.../was/optionprofile` | full CRUD incl. delete. Also where WAS policy-compliance settings live — no separate policy object exists. **Wire wrapper key was wrong since PR #3** (`WasOptionProfile` guessed, `OptionProfile` confirmed by a user-supplied example); fixed | P2 |
| `qualys_was_auth_record` | `.../was/webauthrecord` | **Built.** Form (`STANDARD` flat username/password/loginUrl, or `CUSTOM` generic field list) and server records (flat username/password/domain); secrets masked on read → write-only. Endpoint path, sub-type vocabulary and the flat-element payload shape all confirmed or strongly corroborated by user-supplied excerpts of the official guide (Ch.3 p.102 and a Create/Update walkthrough) — verify against a tenant regardless. Selenium/OAuth2 record types not implemented | done |
| `qualys_was_scan_schedule` | `.../was/wasscanschedule` | **Built.** Single-web-app target, type, profile, `scheduling` sub-object (startDate/timeZone.code/occurrenceType/cancelAfterNHours), `DAILY`/`WEEKLY` recurrence, full `notification` sub-object, sendMail/sendOneMail/sendMailFromAddressOption, activate/deactivate as dedicated endpoints — all Confirmed against a user-supplied Create/Update walkthrough that corrected the top-level flat-field shape a prior pass guessed. **`MONTHLY` recurrence detail and tag-based multi-app targeting remain unmodelled** — no source has shown either. JSON wrapper key `WasScanSchedule` now Confirmed (seen verbatim in the walkthrough) | done |
| `qualys_gcp_connector` (existing) | **Connector v3** `/qps/rest/3.0/.../am/gcpassetdataconnector` | migrate off the deprecated CloudView v1 path (product decision: deprecated items are removed); state migration required for existing users | migrate |
| `data.qualys_was_findings` | `.../was/finding` | **Built.** Read-only: search/get. Filter and finding fields from the derived WAS reference (doc 11 §9), Corroborated non-official. Lifecycle actions (ignore/retest/severity override) not implemented | done |
| `qualys_report_template_scan` / `_patch` | `/api/2.0/fo/report/template/scan|patch/` | XML-body schemas; deferred until schema effort justified | deferred |
| ~~`qualys_user`~~ | `/msp/user.php` (V1) | **Out of scope** — legacy generation, and deprecated items are not being built on | dropped |
| `qualys_was_dns_override` | `/qps/rest/3.0/.../was/dnsoverriderecord` | staging-host resolution for WAS scans | P3 |
| `qualys_was_search_list` / `qualys_was_parameter_set` | `.../was/searchlist`, `.../was/parameter` | referenced by WAS option profiles | P3 |

**Scan authentication approach (task question):** support **both** — resources that
create/manage records with write-only credential arguments (preferred with vault
references so no secret lives in config/state), *and* data sources for referencing
records created outside Terraform. No secrets in the XLSX workbook or Terraform
configuration; vault-backed records are the recommended pattern.

## Deliverable 5 — Candidate Terraform data sources

| Data source | Backing API | Purpose |
|---|---|---|
| `data.qualys_asset_groups` | asset group list | reference AG IDs for scans/schedules/reports |
| `data.qualys_asset_tags` | `search/am/tag` | reference tag IDs for targeting |
| `data.qualys_tagged_assets` | `search/am/hostasset` by tag; CSAM gateway QQL (`not tags.name:"X"`) for the untagged case | tag membership review; tag-compliance gaps |
| `data.qualys_host_assets` | host list | inventory, stale-asset review |
| `data.qualys_host_detections` | host detection list | stale-asset/purge subsystem input |
| `data.qualys_networks` | network list | explicit network selection (workbook rule) |
| `data.qualys_scanner_appliances` | appliance list | scanner selection, readiness checks |
| `data.qualys_option_profiles` | OP list (VM/PC) | profile IDs for scans/schedules |
| `data.qualys_scan_schedules` | schedule list | audit existing schedules |
| `data.qualys_report_templates` | `/msp/report_template_list.php` | template IDs (only list surface that exists) |
| `data.qualys_report_schedules` | schedule/report list | visibility of GUI-managed schedules (no resource possible) |
| `data.qualys_auth_records` | auth list (all types/per type) | reference existing records |
| `data.qualys_users` | `/msp/user_list.php` | owner/recipient validation |
| `data.qualys_scans` / `data.qualys_reports` | scan/report list | job visibility for imperative subsystems |
| `data.qualys_time_zone_codes` | `/msp/time_zone_code_list.php` | validate `time_zone_code` inputs |
| `data.qualys_gcp_connector` (existing) | CloudView | maintain |

## Deliverable 6 — Imperative operations (NOT normal Terraform resources)

Per the task rule, immediate scan execution is imperative unless a stable lifecycle is
demonstrated — none was.

| Operation | Why not a resource | Delivery vehicle |
|---|---|---|
| VM scan launch/cancel/pause/resume/fetch | job with transient REF; re-run ≠ update; destroy meaningless | runbook/CLI helper or CI job; schedules are the declarative alternative |
| Report launch/cancel/fetch/delete | async job artefacts with expiry | reporting subsystem |
| Scheduled report `launch_now` | trigger on GUI-owned object | CLI helper |
| Host purge (`asset/host/?action=purge`) | destructive, irreversible, bulk-selector | **separate stale-asset review & purge subsystem** with human confirmation; purge verification via host list re-read |
| Scorecard / asset-search report | one-shot generation | reporting subsystem |
| WAS scan launch/cancel | async job | CLI helper |
| Connector "run" (v3) | trigger | post-migration helper |
| Appliance readiness poll | verification, not state | `wait_for_online` option / validation helper |
| KnowledgeBase download | subscription-gated bulk read | out of provider scope (P3 helper at most) |

**Unsupported / legacy (recorded, not planned):** VM report-schedule create/update/delete
(GUI-only), distribution-group CRUD (no API), network delete (no API), appliance
replace (UI wizard), physical appliance create/delete (update-only API), scheduled maps
(V1-only, legacy), `/msp/scheduled_scans.php` (legacy except maps),
`/msp/asset_group_list.php` (legacy), compliance/remediation report templates (no API).

**Deferred pending new APIs:** WAS report schedules — no v3 API, but Qualys announced
V4 schedule/report endpoints in April 2026 covering scheduled reports; re-scope once
V4 reference documentation is available.
