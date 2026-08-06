# Qualys API Capability Discovery — Expanded (Phase 0)

**Status:** Discovery only — no provider code has been changed in this phase.
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

## Method and evidence policy

Discovery was grounded in the **official Qualys documentation** (docs.qualys.com,
cdn2.qualys.com/docs PDF guides, notifications.qualys.com API notifications,
success.qualys.com knowledge base). Every operation in the matrix carries:

- the documentation section and URL it was sourced from; and
- a **verification status**:
  - `Confirmed` — the fact was directly confirmed against official documentation
    (page title/snippet/URL evidence captured during discovery);
  - `Unverified` — the fact is believed correct from the documented API pattern or
    adjacent official material, but the specific page/field could not be retrieved
    during this discovery pass. **Nothing marked `Unverified` may be implemented
    without first confirming it against the live documentation or a test tenant.**

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
