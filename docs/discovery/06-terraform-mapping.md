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
| `qualys_host_tag_assignment` | `qps/rest/2.0/update/am/hostasset` tags.add/remove | assignment object (host ↔ tag set) | done |
| `qualys_auth_record_windows` / `qualys_auth_record_unix` / 28 further types | `/api/2.0/fo/auth/<type>/` | **write-only** credential arguments; vault reference support; Windows domain type forces replacement. **All 30 Confirmed type slugs from doc 03 §11 are now built** — the 28 beyond Windows/Unix registered directly against the existing generic `resourceAuthRecord` factory, no new client code needed | done |
| `qualys_vault` | `/api/2.0/fo/vault/` | `title`/`type` managed (Confirmed); per-vault-type connection parameters are a generic passthrough map, not a fabricated per-type schema (still `Unverified` — see doc 08) | done |
| `qualys_domain` | `/api/2.0/fo/asset/domain/` | Basic-auth-only client path; write parameters remain `Unverified` (doc 08) — only `data.qualys_domains` (read) is built | deferred (write side blocked) |
| `qualys_excluded_ips` | `/api/2.0/fo/asset/excluded_ip/` add/remove | supports `expiry_days`. **Built**, but write-only with respect to drift: the list response's element names are unconfirmed, so Read does not decode remote state (see the resource's doc) | done |
| `qualys_web_application` | `qps/rest/3.0/.../was/webapp` | JSON/XML ServiceRequest; TagList association. `auth_record_ids` uses its own `authRecords.add`/`.remove` call, an incremental idiom distinct from tags' "set". A "Gap Review" document later confirmed and closed several more gaps: `attributes` (map, "set" idiom), `config.cancelScansAt`/`cancelScansAfterNHours`/`defaultDnsOverride.id`, `dnsOverrides` (separate "set" collection from the default), `swaggerFile`/`postmanCollection` (base64 file uploads, mutually exclusive), `malwareMonitoring`/`malwareNotification` flags, and read-only `crawlingScripts`. `malwareScheduling` recurrence is deliberately NOT modelled — the same document says its structure should come from a dedicated example, not be inferred. A genuine landmine is reproduced faithfully: `config` sent without a cancellation element clears any existing default-cancellation setting | P2 |
| `qualys_was_option_profile` | `.../was/optionprofile` | full CRUD incl. delete. Also where WAS policy-compliance settings live — no separate policy object exists. **Wire wrapper key was wrong since PR #3** (`WasOptionProfile` guessed, `OptionProfile` confirmed by a user-supplied example); fixed | P2 |
| `qualys_was_auth_record` | `.../was/webauthrecord` | **Built.** Form (`STANDARD`/`CUSTOM`, both via the generic `fields`/`set`/`WebAppAuthFormRecordField` list; `SELENIUM` via `seleniumScript`/`seleniumCreds`), server (flat username/password/domain), and OAuth2 (`oauth2Record`, flat grant-specific fields, NOT the generic list) records; secrets masked on read → write-only. **A "Gap Review" document corrected a prior pass's STANDARD shape**: an earlier walkthrough showed STANDARD credentials as flat `username`/`password` elements, but the official create example — quoted directly this time — uses the same fields list as CUSTOM. Fixed; see doc 08's twelfth research pass. Endpoint path and sub-type vocabulary remain confirmed via a primary-source excerpt (Ch.3 p.102) | done |
| `qualys_was_scan_schedule` | `.../was/wasscanschedule` | **Built.** Single-web-app target, type, profile, `scheduling` sub-object (startDate/timeZone.code/occurrenceType/cancelAfterNHours), `DAILY`/`WEEKLY` recurrence, full `notification` sub-object, sendMail/sendOneMail/sendMailFromAddressOption, activate/deactivate as dedicated endpoints — Confirmed against a user-supplied Create/Update walkthrough. **`MONTHLY` is deliberately NOT exposed**: an earlier pass modelled it by analogy from `qualys_was_report_schedule`'s confirmed MONTHLY shape, and a "Gap Review" document explicitly corrected this — the payload is confirmed for report schedules, not scan schedules, and the client now rejects a MONTHLY request rather than send an unconfirmed one. Tag-based multi-app targeting also remains unmodelled — WAS Multi-Scan launches confirm the shape exists, but not that `WasScanSchedule` accepts it. JSON wrapper key `WasScanSchedule` Confirmed (seen verbatim in the walkthrough) | done |
| `qualys_was_report_schedule` | `.../was/reportschedule` | **Built.** New object, not previously modelled at all. Report template, single-web-app target, output format, email recipients, and the same `schedule` sub-object/recurrence shape as `wasscanschedule` (this pairing is the source of scan-schedule's `MONTHLY` support) — Confirmed against a user-supplied Create/Update walkthrough, wrapper key `ReportSchedule` seen verbatim. Activate/deactivate as dedicated endpoints | done |
| `data.qualys_was_report_templates` | `.../was/reporttemplate` | **Built.** Read-only: count/search/get only, confirmed absent create/update/delete — templates are UI-managed. Only `id`/`name`/`type` modelled; richer template configuration (display, graph/grouping, filters, search lists) is prose-only in the source, no field names given. Distinct from the existing `data.qualys_report_templates`, which covers the legacy VM report template API | done |
| `qualys_gcp_connector` (existing) | **Connector v3** `/qps/rest/3.0/.../am/gcpassetdataconnector` | migrate off the deprecated CloudView v1 path (product decision: deprecated items are removed); state migration required for existing users | migrate |
| `data.qualys_was_findings` | `.../was/finding` | **Built, brought to parity with `data.qualys_vm_findings`.** Read-only: search/get, now with multi-web-app (`web_app_ids`), direct-by-ID (`finding_ids`), multi-QID/status/type/finding_type (single value server-side, 2+ client-side), a severity range, and Confirmed/Corroborated server-side date-range filters (`GREATER`/`LESSER` on `firstDetectedDate`/`lastDetectedDate` — Corroborated, since the field/operator combination itself isn't independently verified). Cross-search dedup by finding ID with conflict diagnostics, mirroring VM. Optional batched `enrich_with_knowledgebase` reuses `vmdr.GetKnowledgeBaseEntries` (the same subscription-wide KB, keyed by QID across VM/PC/WAS). **Breaking**: `severity` (string→int), `status`/`type`/`finding_type` (string→set) changed type | done |
| `qualys_was_finding_ignore` | `.../was/finding` (`ignore`/`reopen` actions) | **Built.** Confirmed against a user-supplied "Findings Lifecycle Actions" walkthrough: `POST ignore\|reopen/was/finding/<id>` with a `data.Finding.comment` body. `fix` is confirmed too (client method `FixWASFinding`) but deliberately not wrapped in a resource — the source itself recommends against using lifecycle actions as a rescan substitute. `retest`/`retestStatus` are now Confirmed and implemented as client methods (`RetestWASFinding`, `GetWASFindingRetestStatus`) — also not a resource, for the same async-job reasoning as scan/report launches (see Deliverable 6). Severity override's existence is confirmed but its exact endpoint/payload is not — still open | done |
| `qualys_report_template_scan` / `_patch` | `/api/2.0/fo/report/template/scan|patch/` | XML-body schemas; deferred until schema effort justified | deferred |
| ~~`qualys_user`~~ | `/msp/user.php` (V1) | **Out of scope** — legacy generation, and deprecated items are not being built on | dropped |
| `qualys_was_dns_override` | `/qps/rest/3.0/.../was/dnsoverride` | **Built.** Path corrected from an earlier `dnsoverriderecord` guess. Confirmed against a user-supplied walkthrough with inline docs.qualys.com citations; update/delete take no ID in the URL path (a real deviation from every other WAS object) — ID travels in the body for update, as a filter criterion for delete | done |
| `qualys_was_search_list` / `qualys_was_parameter_set` | `.../was/searchlist`, `.../was/parameter` | referenced by WAS option profiles. **Deliberately not built**: doc 11 §9 confirms the endpoints and CRUD verbs exist but names no request/response fields at all beyond "same CRUD pattern" — building against that would mean fabricating a schema, which this project's evidence discipline refuses to do | blocked (no field-name evidence) |

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
| `data.qualys_tagged_assets` | `search/am/hostasset` by tag; CSAM gateway QQL (`not tags.name:"X"`) for the untagged case | **Built** (tag lookup only; the untagged-case QQL query is not implemented). tag membership review; tag-compliance gaps |
| `data.qualys_host_assets` | host list | inventory, stale-asset review |
| `data.qualys_host_detections` | host detection list | stale-asset/purge subsystem input |
| `data.qualys_vm_findings` | host detection list (`DETECTION_LIST`, same endpoint as `qualys_host_detections`) | **Built.** Individual VM vulnerability findings (one host + one QID + one detection instance, never aggregated), for a downstream remediation-reporting workflow. Kept as a wholly separate data model from `qualys_host_detections`, which stays host-summary-only — see doc 08's research-pass entry for the evidence tier (Corroborated, not Confirmed) and the deliberate design choices (host_ids/ips/status sent server-side; asset_group_ids resolved to member IPs via the already-Confirmed `GetAssetGroup`; qids/severity-range/date-range filtered client-side rather than guessing unconfirmed query parameters). Optional batched KnowledgeBase enrichment (`data.qualys_vm_findings.enrich_with_knowledgebase`) |
| `data.qualys_networks` | network list | explicit network selection (workbook rule) |
| `data.qualys_scanner_appliances` | appliance list | scanner selection, readiness checks |
| `data.qualys_option_profiles` | OP list (VM/PC) | profile IDs for scans/schedules |
| `data.qualys_scan_schedules` | schedule list | audit existing schedules |
| `data.qualys_report_templates` | `/msp/report_template_list.php` | template IDs (only list surface that exists) |
| `data.qualys_report_schedules` | schedule/report list | **Built.** visibility of GUI-managed schedules (no resource possible) |
| `data.qualys_auth_records` | auth list (all types/per type) | **Built** (one type per read, matching the per-type API). reference existing records |
| `data.qualys_users` | `/msp/user_list.php` | owner/recipient validation. **Built** (eighth pass) — a user-supplied primary-source excerpt (DTD name, full parameter table, complete sample response) made this Confirmed. Surfaced the first real evidence for MSP-style access scoping: a sample Manager-role user carries no `ASSIGNED_ASSET_GROUPS` at all, while a sample Scanner-role user does — asset group assignment, not `BUSINESS_UNIT` membership, looks like the actual visibility mechanism, though this is an inference from one document, not yet corroborated by a Business Unit API reference. Write-side (`user.php` add/edit) and all of Business Unit management (`business_unit.php`) remain completely unresearched — see doc 08 |
| `data.qualys_scans` / `data.qualys_reports` | scan/report list | **Built.** job visibility for imperative subsystems |
| `data.qualys_time_zone_codes` | `/msp/time_zone_code_list.php` | validate `time_zone_code` inputs. **Still blocked**: doc 08 records three research passes finding no field-name evidence for this endpoint's output |
| `data.qualys_gcp_connector` (existing) | CloudView | maintain |
| `data.qualys_tenant_capabilities` | `GET /qps/rest/portal/version` | **Built.** Doc 08 §7's recommended tenant capability probe; decoded as a generic flattened map since no field schema for the response was found by this project |

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
| WAS finding retest (`RetestWASFinding`, confirmed) | async job against accumulating findings state, not an object with an update/delete lifecycle | client method built; poll `GetWASFindingRetestStatus` |
| WAS Burp report import (`ImportWASBurp`, confirmed) | one-shot import into accumulating findings state | client method built (`POST import/was/burp`, flat body — not the usual `data.<Wrapper>` nesting) |
| Connector "run" (v3) | trigger | post-migration helper |
| Appliance readiness poll | verification, not state | `wait_for_online` option / validation helper |
| KnowledgeBase bulk download (a standalone "export the whole KB" operation) | subscription-gated bulk read | out of provider scope (P3 helper at most). **Narrower, batched KB lookups by QID are now built** — see `vmdr.GetKnowledgeBaseEntries`, used only to enrich `data.qualys_vm_findings` — that is a bounded join, not a bulk export, so it did not need this classification |

**Unsupported / legacy (recorded, not planned):** VM report-schedule create/update/delete
(GUI-only), distribution-group CRUD (no API), network delete (no API), appliance
replace (UI wizard), physical appliance create/delete (update-only API), scheduled maps
(V1-only, legacy), `/msp/scheduled_scans.php` (legacy except maps),
`/msp/asset_group_list.php` (legacy), compliance/remediation report templates (no API).

~~**Deferred pending new APIs:** WAS report schedules — no v3 API...~~ **Reversed.**
This discovery's original passes found no evidence of a v3 report-schedule API and
assumed the announced V4 schedule/report endpoints (April 2026) were the only path.
A user-supplied walkthrough later confirmed a v3 `/qps/rest/3.0/.../was/reportschedule`
API does exist (`qualys_was_report_schedule`, built — see the table above). Whether V4
adds something this v3 surface lacks is still open.
