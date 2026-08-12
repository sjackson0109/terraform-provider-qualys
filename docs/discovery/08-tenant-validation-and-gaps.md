# 08 — Endpoints Requiring Tenant Validation, and Documentation Gaps

Deliverables 8 and 9.

## Deliverable 8 — Endpoints requiring tenant validation

These endpoints depend on subscription features, add-ons, roles or platform releases
and must be **probed at provider configure-time (or first use) per tenant**, with
clear diagnostics rather than raw API errors:

| Endpoint / capability | Tenant condition to validate |
|---|---|
| `/api/2.0/fo/network/` (all) | **Network Support** feature enabled on the subscription; caller is Manager |
| `/api/2.0/fo/knowledge_base/vuln/` | KnowledgeBase API authorization granted by Qualys (add-on) |
| `/api/2.0/fo/report/?action=fetch` | **Report Share** enabled |
| `/api/2.0/fo/scan/summary/`, `/scan/vm/summary/` | Manager role |
| `/api/2.0/fo/subscription/option_profile/?action=export|import` | Manager role |
| `/api/2.0/fo/asset/host/?action=purge` | Manager or granted "Purge host information/history" |
| `/api/2.0/fo/appliance/?action=create` | Manager, or UM/Scanner with "Manage virtual scanner appliances" (+ mandatory `asset_group_id` for non-Managers); QVSA licenses remaining |
| `/api/2.0/fo/appliance/?action=assign_network_id` | Manager + Network Support |
| `/api/2.0/fo/asset/domain/` | Basic-auth path; bulk delete requires platform ≥10.25 |
| `client_id`/`client_name` params (scans/schedules) | Consultant-type subscription only |
| EC2 scan params (`connector_name`, `ec2_endpoint`) | EC2 connector configured; platform ≥10.16 behaviours |
| `/qps/rest/3.0/.../was/...` | WAS module licensed |
| `/qps/rest/2.0/.../am/...` | AssetView/CSAM available (standard on VMDR subscriptions) |
| `/cloudview-api/rest/v1/...` | CloudView/TotalCloud module; **deprecation state of connector APIs on the tenant** |
| Auth-record secrets read-back | "View Password in Authentication Record" permission (provider must not depend on it) |
| Rate/concurrency ceilings | subscription tier (headers observed at runtime are authoritative; limits are customisable per subscription by Qualys Support, so **never hardcode tier numbers**) |
| **JWT bearer auth on `/api/2.0/fo/`** | **opt-in per subscription** (OpenID Connect API authentication or native `/auth/oauth` — both require Qualys Support activation). Probe by attempting a token and falling back to Basic; never assume tokens work |
| WAS V4 schedule/report APIs | announced Apr 2026 — availability on the tenant must be probed before use |
| **Qualys EULA acceptance** | **An API-only account (no GUI access) that has not accepted the EULA stays inactive and *every* API call fails.** Acceptance is a first-login step (`acceptEULA.php`). This is the most likely cause of a brand-new service account failing every request, so the provider must detect it and say so explicitly rather than surfacing a generic auth error |
| Per-endpoint API version / EOS date | versions differ per capability and are actively migrating (doc 07); the EOS date is carried per endpoint, so the client should record it and warn when running on a deprecated version |

Recommended pattern: a `provider helper` "tenant capability probe". Start with
**`GET /qps/rest/portal/version`**, which returns installed Portal and per-module
versions (WAS, VM, CA, FIM…) — it feature-gates without touching customer data and is
cheaper than probing each family. Fall back to minimal list calls
(`network?action=list`, `appliance?action=list`, `search/am/tag` limit 1) for the
FO-family features that the version endpoint does not cover, cache the result, and add a
`validation operation` mode for the onboarding workbook (validate network IDs, scanner
names, option-profile IDs, tag IDs, template IDs before apply).

## Deliverable 9 — Capabilities for which official API documentation could not be found

Confirmed-absent (documented nowhere; community evidence corroborates absence):

1. **Report schedule create/update/delete/enable/disable** — GUI-only; API has only
   `list` + `launch_now`.
2. **Distribution group CRUD** (report/scan notification recipients) — UI-only.
3. **Network delete** at API level (UI delete exists).
4. **Host asset/IP delete** (purge is the only API-side removal; official article
   000003463).
5. ~~**Scanner appliance replace** — UI wizard only.~~ **Withdrawn** — a replace API
   exists (`/api/2.0/fo/appliance/replace_iscanner/`); see "Reversed by the third pass".
6. **Physical appliance create/delete** — only `physical/?action=update` exists.
7. **"Launch now" for scan schedules** — no such action (only report schedules have it).
8. **Remediation / compliance-policy report template CRUD** — template APIs cover
   Scan, PCI Scan, Patch, Map only.
9. **Vulnerability scorecards other than vulnerability type via API** — compliance and
   WAS scorecards not launchable by API.

### Closed by the second research pass

The following were `Unverified` in the first pass and are now resolved (details in
doc 03; `Corroborated (non-official)` where noted):

- Asset-group `set_*`/`add_*`/`remove_*` triads for domains, DNS names, NetBIOS names
  and appliance IDs, plus the scalar `set_*` names — and the **create-vs-edit
  parameter asymmetry** (bare names on `add`, prefixed on `edit`).
  *(Corroborated (non-official).)*
- Host-purge parameter set, `data_scope` values (`vm`/`pc`/`vm,pc`), and that purge
  returns **`BATCH_RETURN`** with hosts *queued* — i.e. asynchronous.
  *(Corroborated (non-official).)*
- Host-detection root element `HOST_LIST_VM_DETECTION_OUTPUT` and DTD
  `/api/2.0/fo/asset/host/vm/detection/dtd/output.dtd` (**no `list/` segment**);
  `include_ignored` and `include_disabled` both exist.
- Appliance `set_vlans`/`set_routes` are **authoritative replace** (`""` clears);
  full `output_mode=full` element list; readiness signals (`HEARTBEATS_MISSED`,
  `SS_CONNECTION`, `LAST_UPDATED_DATE`, `*_LATEST` pairs) and the 4-hour platform
  heartbeat cadence.
- **`ACTIVATION_CODE` is re-retrievable via `action=list`, but only until the appliance
  activates** — drives the Computed+Sensitive never-overwrite-with-empty rule.
- Option-profile field-level parameter names (several earlier guesses were wrong);
  `default={0|1}` sets IS_DEFAULT at create and **forces the profile global**;
  `title` is mutable; **no `/api/3.0/` option-profile endpoint exists** (ladder is
  2.0 → 4.0 → 5.0 → 7.0).
- Scan schedule `start_date` (MM/DD/YYYY), the `set_start_time=1` gate requiring all
  five time fields together, `recurrence` as occurrence count, confirmation that
  `end_after`/`end_after_mins` are a **duration cap and not an end date**, and that
  **`ACTIVE` is a four-valued enum** (0/1/2/3).
- Report `report_refs` is required for **Scan Based Findings** templates and unused for
  **Host Based Findings** — determined by the template, not the report type.
- A v3 report endpoint exists for `action=fetch`.
- **Cross-module tag identity is Confirmed** — one subscription-wide tag tree spanning
  VMDR scan/report targeting, AssetView/CSAM and WAS (see doc 03 §14 for the exact
  wording and the AM&T `modules`-list caveat). This was the single most consequential
  open question and it resolves in favour of a single shared `qualys_asset_tag`.
- **Untagged-asset discovery is partly solved**: the CSAM gateway accepts QQL
  `not tags.name:"X"`, so "assets lacking tag X" needs no client-side set difference.
- WAS `cancel/was/wasscan` (with `cancelWithResults`), bulk `delete/was/wasscan`,
  `delete/was/optionprofile`, and full `wasscanschedule` CRUD (9 operations) all exist.
- **`/api/2.0/fo/` does accept JWT bearer tokens** (subscription opt-in) — this reverses
  the first pass's "gateway-only" reading.
- Limit regimes are split: FO/qps return **409**, the CSAM gateway returns **429** with
  no concurrency headers and no per-subscription tiering.

### Newly surfaced (not known in the first pass)

- **WAS V4 schedule/report APIs** announced 13 Apr 2026, explicitly covering scheduled
  reports, with V3 unchanged — this may eventually make a WAS report-schedule resource
  possible. Paths and schemas `Unverified`.
- **Option-profile API versions 4.0 / 5.0 / 7.0** exist (10.36 and 10.39.1); V2 is not
  the only generation.

### Closed by the third pass (official Quick Reference PDF supplied directly)

See **[doc 11](11-verified-parameter-reference.md)** for the full parameter lists.

- **Every missing VM option-profile parameter name** — `vulnerability_detection`
  (`complete|custom|runtime`), the custom/exclude search-list IDs,
  `password_brute_forcing_system|_custom`, `authentication={value1,value2}` (a **list**,
  not per-technology booleans), `enable_additional_certificate_detection`,
  `scan_intensity`, and the system-auth trio. **`update` accepts every create
  parameter**, so `default`/`global` are updatable and nothing is ForceNew.
- **Asset-group parameters are now officially Confirmed** (previously *Corroborated*),
  including that `business_impact`'s tail value is **`none`** (not "Minor"), and that
  no `set_network_id`/`set_owner` exists — network and owner are create-time only.
- **The `asset/ip/?action=update` conflict is resolved**: bare names are *selectors*,
  `new_*` names are *setters*. The provider must write via `new_*`.
- **Purge parameters officially Confirmed**, plus the documented rule that
  `compliance_enabled=1` alongside `data_scope` purges both data types regardless.
- **Excluded-hosts change history endpoint found**:
  `/api/2.0/fo/asset/excluded_ip/history/`; and `action={add|remove|remove_all}` with
  `dg_names`/`network_id` on the excluded-IP endpoint.
- **Network has no delete** — confirmed from the official action enumeration
  (`list`, `create|update` only).
- **Scan schedule gaps all closed**: `start_date`, `recurrence`,
  **`before_notify_message`**, `client_id`/`client_name` on create, and the
  **`active={0|1}` list filter**. ⚠️ Correction: `weekdays` takes **day names**
  (`sunday`…`saturday`), while `day_of_week` takes **0–6** — two encodings in one API.
- **WAS `wasscanschedule` element model confirmed** (`startDate`, `timeZone`,
  `occurrenceType={ONCE|DAILY|WEEKLY|MONTHLY}`, target/profile/scanner elements),
  unblocking that resource apart from the per-occurrence frequency sub-elements.
- **Tagging**: `lastVulnScan` filter confirmed; `evaluate/am/tag` and
  `activate/am/hostasset?module=QWEB_VM|QWEB_PC` newly surfaced; a **`NONE` operator
  exists** in the qps filter language.

### Reversed by the third pass

- **Appliance replacement IS an API operation** — `/api/2.0/fo/appliance/replace_iscanner/`
  with `action=replace`, `old_scaner_name` *(sic)*, `new_scanner_name`,
  `do_not_copy_settings`, `do_not_remove_new_scanner_from_objects`. Previously recorded
  as UI-only.
- `polling_interval` range is **60–360** per the Quick Reference, conflicting with a UI
  help page's 60–3600. Treat 60–360 as authoritative for the API; verify on a tenant.

### Still `Unverified` (bolded items gate design decisions)

- **`asset/ip/?action=update` setter names: `new_owner`/`new_ud1…` (official docs) vs
  bare `owner`/`ud1…` (working third-party SDK).** A direct conflict between two
  otherwise reliable sources — gates `qualys_ip_registration` updates entirely. Settle
  empirically: send one spelling, then read back via `asset/host/?action=list` and
  confirm the value actually changed. See the callout in doc 03 §1.
- **CIDR notation acceptance on `asset/ip/?action=add`** — no Qualys source affirms or
  denies it; only the Restricted IPs endpoint documents CIDR. The earlier
  counter-signal (a third-party type hint accepting `IPv4Network`) is now **withdrawn**:
  reading that library's source shows it converts *responses* — Qualys returns
  `IP`/`IP_RANGE` with hyphenated ranges, which the client turns into network objects
  via `summarize_address_range`. That is output convenience, not input support, and its
  own docstrings say "a host IP range is specified with a hyphen". Evidence now points
  to **hyphenated ranges only on input**. Mitigation is version-proof either way: accept
  CIDR in config, normalise to `first-last` on the wire, normalise returned ranges back
  for diff suppression.
- **Numeric error code for "object not found"** on `edit`/`delete`, and the full v2
  error-code table — **still open for the VMDR FO family only.** The qps families
  (WAS, AM/tagging) are now solved: `ServiceResponse/responseCode` returns
  **`OBJECT_NOT_FOUND`**, with the caveat that an out-of-scope object returns the same
  code as a deleted one, so a permissions fault can look like a deletion. Only `WARNING` code 1980 (truncation) is grounded; 1901 (invalid
  parameter) is single-source non-official; 1905 exists but its meaning is not grounded.
  **Note also that some VM/PA calls return HTTP 200 on error** — the provider must parse
  `CODE`/`TEXT` rather than branch on HTTP status. Until this closes, implement
  not-found detection as a list-by-id returning an empty set, not a code match.
- `NOT_EQUALS` against `tagName` on the classic `/qps/rest/2.0/search/am/hostasset`
  endpoint (confirmed for `tags.name` on the gateway surface only); and any predicate
  for "assets with no tags at all", which still needs client-side computation.
- Scan-schedule list pagination model (no `truncation_limit`/`id_min` surfaced).
- `tag_set_by` on purge; a third-party reference to `/api/5.0/fo/asset/host/?action=purge`
  that could not be corroborated.
- **Option-profile 2.0 → 4.0/5.0/6.0 and report 2.0 → 3.0 deltas — now blocking, not
  deferred.** Doc 07 records an EOS/EOL migration programme covering Scan (v3.0),
  Schedule Scan (**v5.0**), Option Profiles (v4.0–v6.0), Asset Host / VM Detection
  (**v5.0**) and Report Template Scan (v7.0). Pinning V2 therefore has a known expiry,
  and the newer schemas are undocumented in everything obtained so far. This is the
  single most valuable remaining research target.
- WAS V4 schedule/report paths and schemas.
- Report types beyond vulnerability/compliance accepting `use_tags`; a report-specific
  concurrency cap (likely none distinct from the default-2 per-API limit).
- Full per-service-level API limits table (only Standard 300/3600s + concurrency 2 are
  grounded); CloudView's rate-limit regime; the complete platform hostname table
  (UK1/AU1/KSA1/EU3 unconfirmed, and a `.co.uk` vs `.uk` conflict for UK1) — mitigated
  by requiring explicit URL configuration rather than a hardcoded POD map.
- Literal endpoint paths for `delete/was/optionprofile` and `delete/was/wasscan`
  (existence confirmed via guide TOC; verify at runtime before shipping).
- Current version of the evergreen WAS API guide PDF (latest indexed versioned copy is
  3.12 / 2022-06-03, but WAS 3.18 release notes from 2024 prove it is newer).
- Vault per-type parameter schemas; `/api/2.0/fo/vault/index.php` path form; VM-side
  per-type password echo behaviour.
- WAS: `delete/was/optionprofile`, `cancel|delete/was/wasscan`, full `wasscanschedule`
  CRUD + recurrence fields, WAS report-schedule API, per-endpoint JWT support, current
  guide version numbers.
- ~~**`webappauthrecord` field names.**~~ **Closed (Corroborated, non-official) —
  `qualys_was_auth_record` is now built.** A fourth research pass reached
  `raw.githubusercontent.com` (docs.qualys.com and cdn2.qualys.com remain
  egress-blocked in this environment) and pulled the data-class source of an
  independent open-source Qualys API client (`0x41424142/qualysdk`), which agrees
  with this discovery's earlier derived-reference summary: a `formRecord` or
  `serverRecord` sub-object with `type`, `sslOnly`, `authVault`, and a `fields` list
  of `{name, value, secured}` entries (class name `WebAppAuthFormRecordField`,
  wrapped in the same `"set"` idiom already used for tag associations); server
  credentials ride the same `fields` list under the names `username`/`password`/
  `domain`. Two independent sources agreeing is the same evidence bar this
  discovery already accepted for the asset-group `set_*`/`add_*`/`remove_*` triads,
  so it is applied consistently here. **Still not built:** Selenium script and
  OAuth2 record sub-types (`grantType`, `accessTokenUrl`, `clientId`,
  `clientSecret`, `seleniumScript`, …) — same two sources name these fields too,
  but they were left out of this pass to keep the shipped surface verifiable in one
  sitting. Verify the whole resource against a live tenant before depending on it.
- **Fifth pass — a user-supplied primary-source excerpt, direct from the official WAS
  API User Guide (Chapter 3 "Authentication API", p.102, "Current authentication
  record count").** This is the first Confirmed (not merely Corroborated) evidence
  for this object, and it corrected a real mistake: the endpoint path is
  `/qps/rest/3.0/count/was/webauthrecord` — **not** `webappauthrecord`, the name this
  discovery had assumed since the first research pass and which `qualys_was_auth_record`
  was built against. Fixed across the client, tests and docs. The same excerpt confirms
  the count/search filter fields (`id`, `updateDate`, `name`, `lastScan`,
  `lastScanStatus={NOT_USED|SUCCESSFUL|FAILED}`, `tags.id`, `tags.name`, `credentials`,
  `createdDate`) and, via the `credentials` filter, the record sub-type vocabulary:
  `FORM_STANDARD`, `FORM_CERT`, `FORM_SELFINITIAL`, `SERVER_BASIC`, `SERVER_DIGEST`.
  Added to the client as the `WASAuthForm*`/`WASAuthServer*` constants, documented as
  likely-but-not-directly-confirmed for the create-time `type` field (a filter enum
  confirms the vocabulary exists, not that the create payload's field spells it the
  same way). The create/update payload shape itself remains unconfirmed by this
  excerpt — only the count endpoint's filters were in the pasted page. **Next-most-useful
  ask if more of the guide becomes available:** the "Create Authentication Record" and
  "Update Authentication Record" pages (expected to follow this one in Chapter 3), and
  the WAS Scan Schedule chapter's recurrence sub-elements.
- **Sixth pass — a user-supplied "Create and Update Authentication Records" walkthrough
  covering exactly the two pages the fifth pass asked for.** Treated as strong but not
  absolute evidence: it reads as a transcription (clean Markdown/fenced code, unlike the
  raw, messy text the fifth pass's genuine PDF copy/paste produced), not a verbatim quote.
  It corrects a real design mistake rather than just filling a gap: a `STANDARD` form
  record and a server record both use flat `username`/`password` elements (plus
  `login_url` for form records) directly under `formRecord`/`serverRecord` — **not** the
  generic `fields` list of `{name, value, secured}` entries the fourth pass's two sources
  described. `qualys_was_auth_record` sent every credential through that generic list;
  now only a `CUSTOM` form record does (kept, since supported-types documentation
  separately names CUSTOM and the generic list is still the two original sources'
  evidence for it — just not for STANDARD). Also newly confirmed: a flat `comments`
  string element (not a list, as the fourth pass's summarised source implied), and that
  associating an auth record with a web application is a **separate** call —
  `update/was/webapp/<id>` with `data.WebApp.authRecords.add`/`.remove`, each a list of
  `{id}` refs, an incremental idiom distinct from tags' authoritative "set". Added as
  `qualys_web_application.auth_record_ids`, diffed against prior state on every apply.
  The GET response shape for `authRecords` was not shown in either excerpt; this
  provider's decode is inferred by analogy with how `tags` round-trips, disclosed as
  such in code. **Still open:** whether OAuth2/Selenium records follow the same
  flat-element pattern or the generic list.
- ~~**WAS scan schedule recurrence sub-elements.**~~ **Closed for `DAILY`/`WEEKLY`
  (Confirmed) — `MONTHLY` still open.** A user-supplied "Create and Update Scan
  Schedule" walkthrough (transcribed, not a verbatim quote — strong but not absolute
  evidence) supplied exactly the recurrence detail this discovery had been asking for
  since the third pass, and along the way corrected a bigger structural mistake: the
  first `qualys_was_scan_schedule` was built with `startDate`/`timeZone`/
  `occurrenceType` as flat top-level fields, per the official Quick Reference's element
  list, and `cancelOption` at the top level too. The walkthrough shows all of these
  nested under a `scheduling` sub-object (`timeZone` itself wrapping a `.code`), and
  `cancelOption` living under `target` instead. This is the second time in this
  discovery a summarised element list (Quick Reference or a derived reference) named
  the right elements but not their true nesting — the first was the WAS auth record's
  fields-list-vs-flat-elements mistake, closed by the prior pass. **Pattern for future
  passes: treat a flat element list as leads to verify, not a payload shape, until a
  worked example confirms the nesting.** Newly confirmed alongside recurrence:
  `cancelAfterNHours`, the full `notification` sub-object (`active`, `reschedule`,
  `delay.nb`/`.scale`, `recipients.set.EmailAddress`, `message`), `sendMail`,
  `sendOneMail`, `sendMailFromAddressOption`, and that activate/deactivate are dedicated
  endpoints (resolving the "Conflicts with the Quick Reference" item doc 11 §9 recorded
  in favour of the derived reference). `qualys_was_scan_schedule` rebuilt accordingly.
  **Still open:** `MONTHLY` recurrence — no `monthlyOccurrence` example was supplied;
  tag-based (multi-web-app) targeting, still not modelled at all.
- **Eighth pass — WAS option profile wrapper key was wrong since this resource's first
  version, and every mock-based test passed anyway.** A user-supplied "Policy
  Compliance" walkthrough clarifies that WAS has no separate policy-compliance object —
  it's configured through the option profile — and along the way supplied a genuine
  create example: `<OptionProfile>`, not `<WasOptionProfile>`. This project's
  naming-convention pattern (`WasOptionProfile` mirroring `was/optionprofile`, by
  analogy with `WasScanSchedule` for `was/wasscanschedule`) held for the schedule object
  but not this one — there is no reliable rule here, each wrapper key needs its own
  evidence. **Consequence worth internalising:** this bug shipped in PR #3 and survived
  every subsequent review and test run, because this package's unit tests mock the HTTP
  layer — the mock server echoes back whatever key the code under test sends, so a
  wrong wrapper key and its "verification" test both agree with each other and neither
  is checked against reality. Unit tests here prove internal consistency, not wire
  correctness; only a live-tenant probe (still not done for anything in this provider)
  or a primary/worked-example source closes that gap. Fixed: wrapper key corrected to
  `OptionProfile` in both the create/get/search decode path and the update encoder
  (which built its payload as a raw map, so needed a separate fix). Also added:
  `comments`, confirmed as a flat string.
- ~~**WAS report schedules — deferred, assumed no v3 API.**~~ **Reversed; built as
  `qualys_was_report_schedule`.** A ninth research pass — a user-supplied "Create and
  Update Report Schedule" walkthrough, paired with a "Report Templates" walkthrough —
  confirmed a v3 `/qps/rest/3.0/.../was/reportschedule` API exists after all,
  contradicting this discovery's earlier passes, which found nothing and assumed only
  the announced V4 schedule/report endpoints would cover this. Wrapper key
  `ReportSchedule` confirmed verbatim. The report-schedule walkthrough also supplied the
  one piece `qualys_was_scan_schedule` was still missing: a `monthlyOccurrence` example
  (`dayOfMonth`, `everyNMonths`) — applied to scan schedule by analogy, since `DAILY`/
  `WEEKLY` matched exactly between the two schedule types in the evidence obtained.
  Report schedule and scan schedule now share their occurrence-encoding logic in
  `qps/wasscanschedule.go`/`wasreportschedule.go` rather than duplicating it.
  Separately, the report-templates walkthrough confirmed the WAS report template API
  (`/qps/rest/3.0/.../was/reporttemplate`) is genuinely read-only (count/search/get,
  explicitly no create/update/delete) — built as `data.qualys_was_report_templates`,
  distinct from the existing `data.qualys_report_templates` (the legacy VM API). Its
  wrapper key (`ReportTemplate`) was **not** shown in a worked example this time —
  chosen because 3 of 4 confirmed wrapper keys in this codebase lack a `Was` prefix,
  disclosed as a guess, not a repeat of the option-profile mistake.
- ~~**Finding lifecycle actions (ignore/retest/severity override) — not implemented.**~~
  **Partially closed.** A tenth research pass — a user-supplied "Findings Lifecycle
  Actions" walkthrough — confirmed `ignore`/`reopen`/`fix` as `POST
  {action}/was/finding/<id>` with a `data.Finding.comment` body, reusing the `Finding`
  wrapper key already confirmed for get/search. `qualys_was_finding_ignore` built for
  `ignore`/`reopen` (create=ignore, delete=reopen — a genuine declarative state: an
  admin-accepted risk or confirmed false positive). `fix` is implemented at the client
  level (`FixWASFinding`) but deliberately not wrapped in a resource: the same
  walkthrough explicitly recommends against using lifecycle actions as a rescan
  substitute, since "fixed" is meant to be scan-verified and can be reverted by a later
  scan — administrator-declared "this is fixed" doesn't fit this provider's declarative
  model the way `ignore` does. **`retest` and severity overrides (updateSeverity/
  restoreSeverity) remain unimplemented** — no source has covered them yet.
- ~~**WAS DNS override records — field names never obtained, entirely gated.**~~
  **Closed.** An eleventh research pass — a user-supplied walkthrough, this one
  carrying inline citations to specific docs.qualys.com pages (count/search/get/
  create/update/delete_dns.htm) rather than reading as a plain transcription — closed
  this gate and corrected the endpoint path this discovery had recorded:
  `was/dnsoverride`, not the `dnsoverriderecord` guess doc 06 carried since the first
  pass. Wrapper key `DnsOverride` confirmed verbatim. `mappings` (hostName/ipAddress
  pairs) and `comments` (a list, not the flat string confirmed for option
  profile/auth record — DNS override's comments field is evidenced differently, kept
  as shown) use the same set/add/remove idiom already confirmed for tag associations;
  this provider always sends `set`. **Two genuine API deviations, unique to this
  object among everything built so far:** update (`POST update/was/dnsoverride`, no
  path suffix) carries the ID inside the request body instead of the URL, and delete
  (`POST delete/was/dnsoverride`, no path suffix) carries the ID as a filter criterion
  instead. Built as `qualys_was_dns_override`, referenced by
  `qualys_was_scan_schedule.dns_override_id` (already-built field, now backed by a
  real resource instead of only an external ID). Web-application-level default DNS
  override association (mentioned in the walkthrough's overview but with no create/
  update example shown) is not built — same reasoning as the still-unconfirmed
  `qualys_web_application` config fields (`defaultAuthRecord`, `cancelScansAt`, etc.).
- ~~**WAS auth record STANDARD shape (built on lower-confidence evidence);
  `qualys_was_scan_schedule` MONTHLY (built by analogy); Selenium/OAuth2 auth
  records, findings retest, Burp import, and several `qualys_web_application`
  fields — all unimplemented.**~~ **Twelfth research pass — two corrections
  to already-shipped code, plus several new closures.** A user-supplied "Gap
  Review" document, quoting official Qualys reference pages directly rather
  than transcribing a walkthrough, both confirmed several previously open
  gaps and **explicitly contradicted two things this provider had already
  shipped**:
  - **`qualys_was_auth_record` STANDARD shape, corrected again.** The ninth
    research pass's transcribed walkthrough had shown `STANDARD` form
    credentials as flat `username`/`password` elements, and this provider's
    schema followed suit. The Gap Review document quotes the official
    Qualys `STANDARD` create example directly: it uses the same
    `fields`/`set`/`WebAppAuthFormRecordField` list as `CUSTOM`. Fixed in
    `qps/wasauthrecord.go` — `username`/`password` are now encoded into
    that list (as fields named `"username"`/`"password"`) for both
    sub-types; `login_url` remains a flat element, since the correction
    only concerned the credential fields. This is the second time a
    transcribed walkthrough got a nesting detail wrong that a direct quote
    of the official page corrected — see the standing caution below.
  - **`qualys_was_scan_schedule` MONTHLY, reverted.** An earlier pass added
    `day_of_month`/`every_n_months` to `qualys_was_scan_schedule` by
    analogy with `qualys_was_report_schedule`'s confirmed MONTHLY shape,
    reasoning that DAILY/WEEKLY matched exactly between the two schedule
    types. The Gap Review document explicitly rejects that inference:
    "Do not derive the WasScanSchedule MONTHLY payload solely from report
    scheduling... keep this gap open until either the wasscanschedule.xsd
    confirms the structure, or an official WasScanSchedule MONTHLY request
    example is obtained." Reverted: `day_of_month`/`every_n_months` are
    removed from the `qualys_was_scan_schedule` schema, `MONTHLY` is
    removed from its `occurrence_type` validator, and
    `CreateWASScanSchedule`/`UpdateWASScanSchedule` now reject a MONTHLY
    request outright rather than send an unconfirmed payload.
    `qualys_was_report_schedule` keeps MONTHLY — its payload shape is
    independently confirmed, not borrowed.
  - **Newly closed, all Confirmed by the same document:** Selenium form
    auth records (`seleniumScript`/`seleniumCreds`) and a dedicated OAuth2
    auth record (`oauth2Record`, flat grant-specific fields, explicitly
    *not* the generic field list) — both added to `qualys_was_auth_record`.
    Findings `retest`/`retestStatus` (`RetestWASFinding`,
    `GetWASFindingRetestStatus` client methods; not a resource, same
    async-job reasoning as scan/report launch). Burp report import
    (`ImportWASBurp`, `POST import/was/burp` with a flat body — the one
    WAS write call seen so far that does *not* nest under
    `data.<WrapperKey>`). `qualys_web_application` gained `attributes`,
    `config.cancelScansAt`/`cancelScansAfterNHours`/
    `defaultDnsOverride.id`, a separate `dnsOverrides` collection,
    `swaggerFile`/`postmanCollection` (mutually exclusive base64 uploads),
    `malwareMonitoring`/`malwareNotification`, and read-only
    `crawlingScripts`.
  - **Explicitly still open, per the same document:** the exact
    `WasScanSchedule` MONTHLY XML/JSON structure; whether
    `WasScanSchedule` accepts the same tag-based multi-web-app targeting
    confirmed for one-off WAS Multi-Scans (`tags.included`/`excluded`,
    `ALL`/`ANY`); `progressiveScanning`'s full request/response shape;
    `malwareScheduling`'s recurrence structure; the exact `authRecords`
    collection wrapper on a WebApp GET (existence and content-type are now
    confirmed, serialisation as count/list vs. something else is not);
    Edit/Restore Finding Severity's endpoint and payload (existence
    confirmed); the Catalog CRUD/search/promote workflow; Search List/
    Parameter Set schemas; the exact bearer-token endpoint and whether
    `/auth/oidc`/`/auth/oauth` differ; and whether the newly-confirmed rich
    report-configuration fields (`display.contents`/`graphs`/`groups`,
    `filters.searchlists`/`status`/`url`/`remediation`, `showPatched`,
    report format enum) apply to report-template *retrieval* or only to
    report *creation* — deliberately not bolted onto
    `data.qualys_was_report_templates` until that's resolved, since
    building it onto the wrong object would repeat the STANDARD/MONTHLY
    mistake a third time.

  **Standing caution reinforced by this pass:** a transcribed walkthrough
  (however detailed) is not the same evidence tier as a direct quote of an
  official reference page, and a shape "seen only in a companion object's
  example" is not the same as a shape confirmed for the object in question
  — both caused a shipped correction this session, in the same auth-record
  file, twice. Prefer verifying every remaining "by analogy" or
  "transcribed" item against a tenant, the object's own XSD, or a direct
  quote before treating it as load-bearing.
- ~~**VM per-vulnerability detection fields (`DETECTION_LIST`) — deliberately unread by `qualys_host_detections` since the endpoint's own schema was unconfirmed; Qualys KnowledgeBase field names never sourced at all.**~~
  **Thirteenth research pass — closed via a new, separate data source rather
  than by changing `qualys_host_detections`.** A user-supplied requirements
  document asked for the VM equivalent of `data.qualys_was_findings`:
  individual vulnerability findings (one host + one QID + one detection
  instance, never aggregated), for a customer remediation-reporting
  workflow. `hostdetection.go`'s own doc comment had flagged
  `DETECTION_LIST`'s field names as unconfirmed since this provider's
  earliest VM work and deliberately left them undecoded; this pass resolves
  that gap with a **Corroborated, not Confirmed** schema — `vmdr/vmfinding.go`
  decodes `QID`, `TYPE`, `SEVERITY`, `PORT`, `PROTOCOL`, `SSL`, `RESULTS`,
  `STATUS`, `FIRST_FOUND_DATETIME`, `LAST_FOUND_DATETIME`,
  `LAST_TEST_DATETIME`, `LAST_UPDATE_DATETIME`, `LAST_FIXED_DATETIME`,
  `TIMES_FOUND`, `IS_IGNORED`, `IS_DISABLED`, reflecting Qualys's
  long-published Host List Detection API schema from general knowledge
  rather than a fetched or user-supplied source this session (docs.qualys.com
  remains blocked in this sandbox). `FIRST_FOUND_DATETIME`/
  `LAST_FOUND_DATETIME` specifically carry stronger corroboration: they are
  two of the three field names this same doc already records as Confirmed
  for this exact endpoint (the third, `LAST_SCAN_DATETIME`, is the
  host-level field `qualys_host_detections` already used) — evidence this
  provider had without ever connecting it to the per-detection schema
  before. Two fields this session found no confident evidence for
  (a per-detection `FQDN`, a `SERVICE` element) were deliberately left out
  rather than guessed, continuing this project's standing rule.

  A new `vmdr/knowledgebase.go` closes the previously entirely-unsourced
  KnowledgeBase API gap (doc 08 had only ever recorded its endpoint path and
  subscription-gated status, never a field name) the same way: `QID`,
  `TITLE`, `CATEGORY`, `CVSS>BASE`, `CVSS_V3>BASE`, `PCI_FLAG`, `PATCHABLE`,
  Corroborated on the same basis. It is used only for a bounded, QID-list
  join (`GetKnowledgeBaseEntries`, one batched request for however many
  unique QIDs a `qualys_vm_findings` read produces) — the standalone bulk-KB
  download this doc already classified as out of scope is untouched by this
  pass.

  **Deliberately not sent as query parameters, and applied client-side
  instead:** `qids`, the severity range, and all six date-range filters
  (`first_found_after/before`, `last_found_after/before`,
  `last_test_after/before`). This endpoint's Confirmed parameter list (this
  doc, above) has never included `qids`/`severities`/a detection-date-range
  filter for this specific endpoint, even though conventions with those
  names exist elsewhere in the Qualys VM API — asserting one here without
  confirmation risks a filter that silently does nothing if the name is
  wrong, exactly the class of mistake the WAS auth-record/scan-schedule
  corrections (this doc's twelfth pass) warn against repeating. `host_ids`,
  `ips` and `status` remain genuine server-side parameters, all three
  already Confirmed/used by `qualys_host_detections` against this same
  endpoint. `asset_group_ids` is resolved to member IPs via
  `GetAssetGroup` (already Confirmed) rather than guessing an `ag_ids`
  parameter for this endpoint specifically — a composition of two
  already-evidenced facts instead of a new one.

  Built as `data.qualys_vm_findings`, explicitly **not** a change to
  `qualys_host_detections` — the user's requirement was to keep host
  inventory and vulnerability findings as separate data models, and this
  provider's existing WAS precedent (`qualys_host_detections` vs.
  `qualys_was_findings`) already established that split.
- ~~**`data.qualys_was_findings` — thin single-value filtering, no dedup,
  no KnowledgeBase enrichment, no VM-findings-style multi-value/range
  support.**~~ **Fourteenth research pass — brought to parity with the
  thirteenth pass's `data.qualys_vm_findings`, following the same
  requirements document's explicit request to mirror that structure while
  respecting WAS-specific identity and fields.** `qps/wasfinding.go` gained:
  - `UniqueID` on `WASFinding` — Confirmed, not new evidence but a
    connection this provider had not made before: `GetWASFindingRetestStatus`
    (added in the twelfth pass) already decodes a `Finding.uniqueId` from
    the same object family this file's get/search calls decode.
  - `ID`/`QID` EQUALS filters and four `GREATER`/`LESSER` date-range
    filters (`firstDetectedDate`/`lastDetectedDate`) — doc 11 §9 already
    listed `id`, `qid`, `firstDetectedDate` and `lastDetectedDate` as
    Confirmed Finding filter fields, and documents `GREATER`/`LESSER` as
    generally valid qps operators; `qps/client.go`'s own pagination cursor
    already depends on `id GREATER lastId` working, corroborating the
    operator further. The specific field/operator combination for the date
    bounds is still Corroborated, not independently Confirmed, since "not
    every operator is valid on every field" per the same doc.
  - `SearchWASFindings` now dedupes by finding ID and returns
    `[]WASFindingConflict` for genuine disagreements, mirroring
    `vmdr.ListVMFindings` exactly. This changed the method's signature
    (`(findings, err)` → `(findings, conflicts, err)`) — a contained,
    deliberate breaking change to an internal package method with exactly
    two callers in this repo (both updated).

  `provider/data_source_was_findings.go` was rewritten rather than
  incrementally patched, because the requirements document specified exact
  types for `severity`/`status`/`type`/`finding_type` that differ from what
  was shipped (`severity` string→int, the other three string→set) — a
  **deliberate breaking change**, called out prominently in the resource
  description and doc rather than hidden. New filters follow the same
  single-value-server/multi-value-client-side split established in the
  thirteenth pass for VM findings, for the same reason: this session found
  no Confirmed multi-value or range query parameter for `qids`, `status`,
  `type`, `finding_type`, or the severity range on this endpoint, so
  asserting one risks a silently-inert filter. `web_app_ids` resolves to N
  searches (one per ID, each using the Confirmed single-value `webApp.id`
  filter) merged and deduplicated, rather than guessing an `IN`-style
  multi-value parameter. `finding_ids` bypasses search entirely via direct
  `GetWASFinding` calls (already Confirmed), then applies every other
  filter to that fixed set client-side.

  KnowledgeBase enrichment reuses `vmdr.Client.GetKnowledgeBaseEntries`
  directly rather than duplicating a WAS-specific KB client: Qualys
  maintains one subscription-wide KnowledgeBase keyed by QID across VM, PC
  and WAS, so the thirteenth pass's KB client already serves this data
  source too. `vmdr.KBEntry` gained `Diagnosis`/`Consequence`/`Solution`
  (same Corroborated tier as the rest of that type) since this document
  explicitly asked for them under WAS enrichment; `data.qualys_vm_findings`
  picked up the same three fields for free, a pure non-breaking addition.

  **Deliberately not implemented, per the same document's own preference
  for "finding metadata... rather than raw HTTP request/response
  evidence":** `category`, `description`, `impact`, `solution` (as direct
  Finding fields — `category`/`solution`-equivalent content is available
  via enrichment instead), `payload`, `request`, `response`, `parameter`,
  `authentication`, `ssl`, `access_path`, `external_reference`,
  `owasp_category`, `wasc_category`, `cwe_id`, and a CVSS v2 score on the
  Finding object itself. None have a confirmed representation on the
  Finding object in this session's evidence; inventing wire shapes for them
  would repeat the exact mistake this doc's twelfth pass corrected twice
  already in the same auth-record file.
- **`time_zone_code_list.php` output field names.** The endpoint's existence and its
  DTD name (`time_zone_code_list.dtd`) are confirmed (doc 03 §8), and it is the source
  of the `time_zone_code` values `qualys_scan_schedule` accepts. Three research passes
  (this discovery's original pass, plus two fresh web searches specifically for this
  gap) found no page or sample output naming its XML fields — unlike the sibling
  `report_template_list.php`, whose fields a fresh search *did* surface (doc 11-style
  evidence: official page + sample XML). **Gates `data.qualys_time_zone_codes`** —
  `time_zone_code` stays a free-text string on `qualys_scan_schedule` (unvalidated
  client-side, exactly like `bruteforce_option` on `qualys_was_option_profile`) rather
  than being checked against a guessed enum.
- **Domain V2 (`/api/2.0/fo/asset/domain/`) add/update/delete parameter names.**
  The family's existence, Basic-auth-only/Manager-role requirements, and the
  `list` output shape are confirmed (a fresh search surfaced the official "List
  Domain" page's sample XML: `DOMAIN_LIST`/`DOMAIN` with `DOMAIN_NAME`,
  `DOMAIN_ID`, `NETWORK`, `NETBLOCK`/`RANGE`/`START`/`END`). Searches for the
  write-side pages ("Update Domain", "Manage Your Domains") surfaced only that
  `domain` and `netblock` parameters exist, not a complete reference — direct
  fetches of docs.qualys.com are blocked in this environment (see the research
  constraint in this document's header), so the pages themselves could not be
  read. **Gates the write side of `qualys_domain`** — `data.qualys_domains`
  (read-only) is built; a resource is not, until this closes.
- Tagging: `CLOUD_ASSET` ruleType; `lastVulnScan` hostasset filter.
- V2 user API existence (`/api/2.0/fo/user/` — third-party claim only).
- **`user.php` `add`/`edit` (user create/update) parameters, and all of the
  Business Unit API (`business_unit.php` — list/add/edit, or whether it is
  even API-accessible rather than UI/Manager-only).** The eighth pass closed
  the read side of users (`user_list.php`, now Confirmed — see below) but
  found nothing on either of these. This is the current blocker on
  Terraform-managed MSP-style tenant onboarding (customer users + business
  units + asset scoping in one apply): a `qualys_user` resource cannot be
  built without confirmed `add`/`edit` parameters (does `business_unit_id`
  really exist as a settable field? what does `asset_group_ids` look like
  on write, versus the read-side `ASSIGNED_ASSET_GROUPS` title list?), and
  a `qualys_business_unit` resource cannot be built at all without first
  confirming the endpoint exists and what it accepts.
- **Whether Business Unit membership itself gates asset/scan/report
  visibility, or whether it is purely organisational and asset group
  assignment does the actual scoping.** The eighth pass's `user_list.php`
  sample is suggestive (see the Confirmed entry below) but not conclusive —
  needs either an official access-control/permissions page or empirical
  confirmation against a tenant with two business units and cross-checking
  what a Unit Manager can list via the API.
- Formal deprecation status of `/msp/scheduled_scans.php`, `/msp/user.php`,
  `/msp/asset_group_list.php`.
- Scan-list/appliance-list/OP-list truncation mechanisms (none surfaced — absence
  unproven).
- Not-found error codes per endpoint (needed for Terraform state removal semantics).
- Exact per-tier rate/concurrency numbers; complete per-POD platform URL table.
- qps (WAS/AM) and CloudView rate-limit header behaviour.
- **WAS Catalog, search list and parameter set response/request field names.**
  Doc 11 §9 (a derived secondary source) confirms the endpoints
  (`/qps/rest/3.0/<op>/was/catalog[/<id>]`, `.../was/searchlist`,
  `.../was/parameter`) and their CRUD verbs exist, but names no fields at
  all beyond "same CRUD pattern" for the latter two, and no create verb at
  all for Catalog (only count/search/get/update/delete — consistent with
  catalog entries being scan-discovered, not user-created). **Deliberately
  not built this pass**, unlike the seventh pass's Corroborated-tier
  builds below: those had at least a DTD filename or an internally
  consistent request/response naming convention to build against; this has
  neither.
- **`/qps/rest/portal/version` response schema.** Doc 11 names this as the
  recommended tenant capability probe but gives no field list, DTD, or
  class name. `qps.GetPortalVersion` (seventh pass, below) decodes it
  generically rather than assert field names.
- **Vault update/delete selector parameter name (`id` vs `ids`).** Doc 03
  §11 confirms vault CRUD exists on `/api/2.0/fo/vault/` but does not print
  the update/delete selector field, unlike scan and report schedules on
  the same API family, which do confirm `id` (singular) for the equivalent
  call. `vmdr.UpdateVault`/`DeleteVault` (seventh pass) use `id` by analogy
  with those two, not by direct confirmation for this endpoint.
- **Excluded-IP list response element names.** Doc 03 §1 confirms the
  `list` action and its `ips` request parameter exist, but no source found
  by this project names the response's XML elements — not even a DTD
  filename, which is why this is listed as a harder gap than the
  Corroborated-tier VM scan/report/report-schedule lists below.
  `qualys_excluded_ips` (seventh pass) does not attempt to decode it; see
  its resource doc for the consequence.

### Seventh pass — building out the remaining gap-list items identified from doc 06's Deliverable 4/5/6 tables

Closed a batch of items previously listed only as candidates, at whatever
evidence tier each one's own research trail actually supports — most at
Confirmed (reusing doc 03 §11's already-Confirmed authentication record type
slugs and vault CRUD fields, and doc 03 §1/§14's confirmed excluded-IP and
host-tagging parameters), a few at Corroborated (VM scan/report/report-schedule
list response field names, inferring from this codebase's own
consistently-held convention that a Qualys response field name matches its
request filter's spelling — the same reasoning basis, not the same claim,
as the DETECTION_LIST precedent in `vmfinding.go`).

Built: 28 additional `qualys_auth_record_*` resources (every remaining
Confirmed type slug from doc 03 §11); `qualys_vault` (create/update/delete,
with per-vault-type parameters left as a generic passthrough map rather than
a fabricated schema — see doc comment on `vmdr.VaultInput`);
`qualys_excluded_ips`; `qualys_host_tag_assignment` and
`data.qualys_tagged_assets`; `data.qualys_auth_records`; `data.qualys_scans`;
`data.qualys_reports`; `data.qualys_report_schedules`; and
`data.qualys_tenant_capabilities` (the doc 08 §7-recommended
`GET /qps/rest/portal/version` probe, decoded as a generic flattened map
since this session found no field schema for it at all).

Explicitly not built in this pass, and not silently dropped either: WAS
Catalog/search lists/parameter sets (no field-name evidence — see the
`Unverified` entry above), the WAS finding rich fields and severity override
(already-known unresolved from earlier passes), Domain V2 write parameters
(already-known unresolved), `time_zone_code_list.php` fields (already-known
unresolved), and VM/PA user/distribution-group write APIs (deliberately
deferred, doc 06 P3, not a research gap — the legacy V1 API's different auth
model makes it a design decision to postpone, not a missing fact). Imperative
operations (scan/report launch and lifecycle actions) remain intentionally
un-modelled as resources per doc 06's classification, which this pass does
not revisit.

### Eighth pass — user-supplied primary-source excerpt for "List Users" (`user_list.php`), prompted by an MSP business-unit/user-visibility question

The user asked how to set up business units and users so that a customer
logging in sees only their own assets, scans and results, and how an MSP
parent account sees across all business units — a capability area this
project had never researched (only the endpoint's existence was known,
via doc 03's `/msp/user_list.php` reference, deferred P3 in doc 06). Rather
than build against general knowledge of Qualys's MSP model — which this
project has no confirmed evidence for and would have meant fabricating a
schema, exactly what this discovery process exists to prevent — the user
was asked for targeted search queries and supplied a primary-source excerpt
from the official "List Users" API page in response.

**Confirmed** by that excerpt: the DTD name (`user_list_output.dtd`), the
full request parameter table (`external_id_contains`, `external_id_assigned`
— mutually exclusive — and `show_access_permissions`), and a complete
sample response covering two users (a Manager and a Scanner) with every
field this project's `vmdr.User`/`data.qualys_users` now expose. Built
directly against this excerpt with no gaps filled by inference on the
field-name level.

**Not Confirmed, but a genuine and useful clue**: comparing the two sample
users shows the Manager carries no `ASSIGNED_ASSET_GROUPS` element at all,
while the Scanner does, and both users' `BUSINESS_UNIT` value is literally
"Unassigned". That is consistent with asset-group assignment being the real
access-scoping mechanism and Business Unit being closer to an organisational
label — which would mean an MSP-style "customer only sees their own assets"
setup is really about assigning the right asset groups per user, not just
attaching them to a business unit. This is a working hypothesis from one
document's sample data, not a stated fact in the excerpt, and it does not
yet answer the MSP parent-account cross-business-unit visibility half of the
original question at all. Recorded as `Unverified` above pending either a
Business Unit API document or empirical confirmation against a tenant with
more than one business unit.

**Still open, and the actual blocker on building anything write-side for
this capability area**: `user.php` create/edit parameters, and the entire
Business Unit API (existence, endpoint, parameters, response shape). See
the `Unverified` entries above for the specific search queries that would
close each.

> Resolution path: re-run targeted checks from an environment with direct access to
> docs.qualys.com (the discovery environment's network policy blocked it — see README),
> then confirm empirically against a test subscription before each affected schema is
> frozen.
