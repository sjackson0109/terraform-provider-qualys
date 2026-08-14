# Qualys API Capability Discovery — Expanded (Phase 0)

**Status:** Discovery complete; implementation underway. See
[Implementation status](#implementation-status) below for what has been built
against these findings.
**Branch:** `claude/qualys-api-discovery-expansion-l9a7y4`
**Date:** 2026-08-06

This directory is the expanded capability-discovery output for the Qualys Terraform
provider. It treats the **Qualys VM and PA API documentation**
(<https://docs.qualys.com/en/vm/qweb-all-api/#t=get_started%2Fget_started.htm>) as the
primary source for VMDR capabilities. The TotalCloud/CloudView Swagger endpoint is kept
as a **separate API-family source** and is never used to infer VMDR endpoint behaviour.

## Document map

| # | Document | Deliverable |
|---|----------|-------------|
| 01 | [API family inventory](01-api-family-inventory.md) | Inventory of every API family in scope (VM/PA, WAS, AssetView/CSAM tagging, TotalCloud/CloudView) with documentation sources |
| 02 | [VM/PA documentation inventory](02-vmdr-documentation-inventory.md) | Navigable inventory of the relevant VM and PA API documentation structure |
| 03 | [Capability-to-API matrix](03-capability-api-matrix.md) | Expanded capability-to-API matrix (every discovered operation, classified) |
| 04 | [API family comparison](04-api-family-comparison.md) | Comparison of VMDR XML APIs, WAS APIs and TotalCloud APIs |
| 05 | [Authentication, formats, limits](05-authentication-formats-limits.md) | Authentication discovery, request/response formats, API limits and retry policy |
| 06 | [Terraform mapping](06-terraform-mapping.md) | Candidate resources, candidate data sources, imperative operations |
| 07 | [API version selection](07-api-version-selection.md) | Per-capability API version recommendation |
| 08 | [Tenant validation & gaps](08-tenant-validation-and-gaps.md) | Endpoints requiring tenant validation; capabilities with no official documentation found |
| 09 | [Revised MVP sequence](09-mvp-sequence.md) | Revised MVP sequence based on actual supported API dependencies |
| 10 | [Onboarding workbook schema](10-onboarding-workbook-schema.md) | Companion: XLSX workbook columns implied by the confirmed API surface (explicit network declaration; no secrets) |
| 11 | [Verified parameter reference](11-verified-parameter-reference.md) | Authoritative parameter lists read from the official Qualys API Quick Reference — supersedes conflicting detail elsewhere |

## Method and evidence policy

Discovery was grounded in the **official Qualys documentation** (docs.qualys.com,
cdn2.qualys.com/docs PDF guides, notifications.qualys.com API notifications,
success.qualys.com knowledge base). Every operation in the matrix carries:

- the documentation section and URL it was sourced from; and
- a **verification status**:
  - `Confirmed` — the fact was directly confirmed against official documentation
    (page title/snippet/URL evidence captured during discovery);
  - `Corroborated (non-official)` — the fact is not visible in retrievable official
    documentation, but two or more **independent** third-party implementations that
    call the API agree on it verbatim. Principal sources: the Cortex XSOAR
    `demisto/content` Qualys v2 pack; the `qualysdk` Python library
    (<https://github.com/0x41424142/qualysdk>, docs <https://qualysdk.jakelindsay.uk>),
    whose call schemas were read directly from source at commit `e8c9529`; and
    PowerShell wrappers such as `MSAdministrator/POSH-Guard`. Strong enough to design a
    schema against; **must still be probed against a tenant before release**. Where
    these sources *disagree* with official documentation the conflict is recorded
    explicitly rather than silently resolved — see the `asset/ip/?action=update`
    callout in doc 03 §1;
  - `Unverified` — the fact is believed correct from the documented API pattern or
    adjacent official material, but the specific page/field could not be retrieved
    during this discovery pass. **Nothing marked `Unverified` may be implemented
    without first confirming it against the live documentation or a test tenant.**

### Source materials supplied directly

The **Qualys API Quick Reference Guide** (official PDF) was supplied for this discovery
and is the authority behind doc 11. Qualys copyright material is **cited, not
committed** — the extracted text is not stored in this repository.

Two documents would still materially improve the output and could not be obtained:

1. **Qualys API (VM and PA) User Guide** (`qualys-api-vmpc-user-guide.pdf`) — its
   appendices hold the v2 error/response code table (needed for not-found detection)
   and the per-service-level API limits table. The Quick Reference contains neither.
2. **WAS API User Guide** (`qualys-was-api-user-guide.pdf`) — for the per-occurrence
   recurrence sub-elements of `WasScanSchedule`.

### Research constraint (must be re-run before implementation)

The discovery environment's network policy **blocked direct HTTPS fetches** to
`docs.qualys.com`, `www.qualys.com`, `cdn2.qualys.com` and `web.archive.org`
(proxy CONNECT 403). Web search against the official documentation *was* available and
is what grounds the `Confirmed` statuses (official page titles, URLs and content
snippets). Consequences:

1. Full page-by-page crawling of the MadCap TOC at
   `docs.qualys.com/en/vm/qweb-all-api/` was not possible; the navigable inventory in
   doc 02 was reconstructed from confirmed page URLs plus the published user-guide
   structure.
2. Items that a direct crawl would trivially settle (exact parameter lists on a few
   pages, DTD contents) are marked `Unverified` and listed in doc 08.
3. Before implementation starts, doc 08's `Unverified` list must be resolved either
   from an environment that can reach docs.qualys.com or against a test subscription.

## Scope decisions carried into this discovery

- **VM/PA (VMDR) XML API** (`/api/2.0/fo/...`) is the primary family: assets, asset
  groups, networks, domains, scanner appliances, option profiles, scans, scan
  schedules, reports, report schedules, scan authentication, users, host detections,
  purge, API limits, lifecycle/deprecation notices.
- **WAS** (`/qps/rest/3.0/...`): web applications, WAS option profiles, scans,
  authentication records, scan schedules, reports/report schedules, tag associations.
- **AssetView / CSAM tagging** (`/qps/rest/2.0/.../am/...`): tag CRUD, static/dynamic
  tags, hierarchy, host-asset tag assignment/removal, untagged-asset discovery.
- **TotalCloud / CloudView** (`/cloudview-api/rest/v1/...`): retained **only** for the
  connector capabilities the existing provider already ships (GCP connector resource +
  data source). Unrelated AWS/Azure/GCP/OCI connector operations are not inventoried,
  except to record the officially announced deprecation path that affects the existing
  resource (see docs 01 and 04).
- Secrets (scan authentication passwords, vault credentials) are **never** placed in
  the XLSX onboarding workbook or in Terraform state where avoidable; see doc 06.
- The onboarding workbook must declare the intended Qualys **network explicitly**;
  network selection is never derived from whether an address is public or private
  (see the Networks sections of docs 03 and 06).
- XML parsing in the eventual implementation must be secure: **no remote DTD
  retrieval; external entity and external DTD resolution disabled** (see doc 05).

## Implementation status

Phases 0–3 of the sequence in [doc 09](09-mvp-sequence.md) are implemented, plus
the well-grounded part of Phase 4.

| Discovery finding | Where it landed |
|---|---|
| Secure XML parsing; never fetch DTDs | `vmdr/xml.go` |
| Parse `CODE`/`TEXT`, not HTTP status | `vmdr/errors.go` |
| EULA failure is its own diagnostic | `vmdr/errors.go` (`ErrEULANotAccepted`) |
| 409/429 split; honour `X-RateLimit-ToWait-Sec` | `vmdr/client.go`, `qps/client.go` |
| Never repeat creates or destructive calls | `request.nonIdempotent` |
| Per-capability API versions + EOS warnings | `vmdr/version.go` |
| Truncation continuation, never to a foreign host | `vmdr/pagination.go` |
| `OBJECT_NOT_FOUND` also means out-of-scope | `qps/client.go`, surfaced as a warning |
| Asset group create/edit parameter asymmetry | `vmdr/assetgroup.go` |
| `new_*` setters vs bare selectors | `vmdr/ip.go` |
| Purge is async and needs a host selector | `vmdr/ip.go` |
| Activation code disappears after activation | `provider/resource_virtual_scanner.go` |
| `ACTIVE` is a four-valued enum | `vmdr/scanschedule.go` |
| `set_start_time` gates the whole time group | `vmdr/scanschedule.go` |
| `weekdays` names vs `day_of_week` numbers | `provider/resource_scan_schedule.go` |
| Custom detection silently falls back | rejected at plan time |
| Default option profile is also global | rejected at plan time |
| Credentials are write-only; vaults preferred | `provider/resource_auth_record.go` |
| One tag tree across VM, AssetView and WAS | single `qualys_asset_tag` |
| CIDR accepted, hyphenated ranges on the wire | `vmdr/ipset.go` + set hashing |
| WAS web application + option profile CRUD | `qps/webapp.go`, `qps/wasoptionprofile.go`, `provider/resource_web_application.go`, `provider/resource_was_option_profile.go` |
| Scan schedule / VM option profile lookup for existing objects | `provider/data_source_scan_schedules.go`, `provider/data_source_option_profiles.go` |
| Stale-asset review input (host last-scan time) | `vmdr/hostdetection.go`, `provider/data_source_host_detections.go` |
| Report template lookup (legacy V1 list, deliberately isolated) | `vmdr/reporttemplate.go` (`request.rawEndpoint`/`legacyEndpoint`), `provider/data_source_report_templates.go` |
| Domain + netblock lookup (read-only; write parameters unconfirmed) | `vmdr/domain.go`, `provider/data_source_domains.go` |

Still not implemented, and why:

- **`qualys_was_auth_record`** — `webauthrecord` create/get/update/delete/search
  are confirmed to exist and secrets are confirmed masked on read, but no source
  obtained during discovery surfaces the actual field names for the
  form/server/Selenium/OAuth2 credential payload. Per this discovery's evidence
  policy, a credential-bearing resource is not built on a guessed wire schema;
  see doc 08's `Unverified` list.
- **`qualys_was_scan_schedule`** — the element model is otherwise confirmed
  (doc 11 §8), but the per-occurrence frequency sub-elements (every N
  days/weeks/months) remain `Unverified`, and a schedule resource cannot omit
  its own recurrence.
- **`data.qualys_time_zone_codes`** — the endpoint (`time_zone_code_list.php`)
  and its DTD name are confirmed, but three research passes found no source
  naming its XML output fields (unlike the sibling `report_template_list.php`,
  where a fresh search did). `time_zone_code` stays an unvalidated free-text
  field on `qualys_vm_scan_schedule`; see doc 08.
- **`qualys_domain`** (write side) — the Domain V2 API's existence, auth
  requirements and list output are confirmed, and `data.qualys_domains`
  covers the read side, but no source names the add/update/delete request
  parameters; see doc 08.
- **Report resources** — report *schedules* have no create/update/delete API at
  all; report launches are jobs, not configuration.
- **CloudView → Connector v3 migration** — the existing `qualys_gcp_connector`
  still uses the deprecated CloudView path.
- **Policy Compliance** — out of scope pending a decision; the `/pc/` endpoints
  are parallel variants the client already handles.

The `Unverified` items in [doc 08](08-tenant-validation-and-gaps.md) that still
gate work are the newer-generation schemas (scan v3.0, schedule v5.0, option
profile v4.0–v6.0, detection v5.0), the VM/PA numeric error-code table, and
whether `ips` accepts CIDR. None blocks what is built: the client normalises
CIDR either way, and not-found detection uses an empty-list read rather than a
code match.
