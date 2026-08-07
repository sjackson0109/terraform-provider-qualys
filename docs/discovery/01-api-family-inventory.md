# 01 — API Family Inventory

Deliverable: the complete inventory of API source families in scope for this provider,
with their official documentation sources. Each family has its own base path, envelope,
encoding, pagination and authentication model — **no family may be used to infer the
behaviour of another**. In particular, the TotalCloud/CloudView Swagger endpoint is a
separate source and must not be used to infer VMDR endpoint behaviour.

## Family 1 — Qualys VM and PA APIs (VMDR XML API) — PRIMARY

- **Base path:** `https://qualysapi.<platform>/api/2.0/fo/...` (plus legacy V1 `/msp/*.php`)
- **Style:** action-parameter RPC over GET/POST form-encoded requests; XML responses with
  per-endpoint DTDs; `SIMPLE_RETURN` envelope for mutations.
- **Primary documentation (authoritative for VMDR capabilities):**
  - All-in-one online doc: <https://docs.qualys.com/en/vm/qweb-all-api/#t=get_started%2Fget_started.htm>
    (merged projects: `qapi-assets`, `qapi-scan`, `qapi-rep`, `qapi-auth`, `qapi-user`, `qapi-pc`)
  - Per-module tree: <https://docs.qualys.com/en/vm/api/index.htm>
  - Qualys API (VM and PA) User Guide **v10.39.1** (2026-07-10): <https://cdn2.qualys.com/docs/qualys-api-vmpc-user-guide.pdf>
  - Qualys API (VM, PA) XML/DTD Reference **v10.39.1** (2026-07-13): <https://cdn2.qualys.com/docs/qualys-api-vmpc-xml-dtd-reference.pdf>
  - Qualys API Quick Reference: <https://cdn2.qualys.com/docs/qualys-api-quick-reference.pdf>
  - API limits: <https://docs.qualys.com/en/vm/api/users/get_started/api_limits.htm> and
    <https://cdn2.qualys.com/docs/qualys-api-limits.pdf>
  - API lifecycle / deprecation notices: <https://notifications.qualys.com/api/> — note the
    Aug 2025 versioning standard (12-month migration window: 6 months End of Support + 6 months EOL):
    <https://notifications.qualys.com/api/2025/08/17/updates-on-api-versioning-standards-deprecation-timelines>
- **Sections inventoried** (detail in docs 02–03): Assets (hosts, IPs, asset groups,
  networks, domains, excluded IPs), Reports (reports, templates, schedules, scorecards,
  asset search), Scan Authentication (auth records, vaults), Scans, Scan Schedules,
  Scanner Appliances, Option Profiles, Users, API limits, lifecycle/deprecation.

## Family 2 — Qualys WAS APIs

- **Base path:** `https://qualysapi.<platform>/qps/rest/3.0/<operation>/was/<object>[/<id>]`
- **Style:** REST-ish portal API; `ServiceRequest`/`ServiceResponse` envelope; XML default,
  JSON supported (WAS 4.5+, requires both `Accept` and `Content-Type: application/json`);
  XSDs at `/qps/xsd/3.0/was/<object>.xsd`; pagination via `hasMoreRecords`/`lastId` +
  `Criteria field="id" operator="GREATER"`.
- **Documentation:**
  - Online: `https://docs.qualys.com/en/was/api/...` (get started, web apps, option
    profiles, auth records, scans, schedules, reports)
  - WAS API User Guide: <https://cdn2.qualys.com/docs/qualys-was-api-user-guide.pdf>
    (current evergreen version number `Unverified`; versioned copies exist, e.g. v3.12 under
    `cdn2.qualys.com/docs/version/10.21/`)
- **Sections inventoried:** web applications (`webapp`), WAS option profiles
  (`optionprofile` — WAS scan configurations), scans (`wasscan`), authentication records
  (`webappauthrecord`), scan schedules (**object name is `wasscanschedule`**, not
  `wasschedule`), reports (`report`) and report templates, tag associations (`TagList` on
  `webapp`). WAS report *schedules* exist in the UI; a WAS API for them is `Unverified`
  (none surfaced).

## Family 3 — Qualys AssetView / CSAM tagging APIs

- **Base path (tag management — current):**
  `https://qualysapi.<platform>/qps/rest/2.0/<create|search|get|update|delete|count>/am/<tag|hostasset|asset|...>`
- **Style:** same `ServiceRequest`/`ServiceResponse` portal envelope and pagination as WAS,
  but version `2.0` and module `am`.
- **Documentation:**
  - Asset Management & Tagging API v2 User Guide:
    <https://cdn2.qualys.com/docs/qualys-asset-management-tagging-api-v2-user-guide.pdf>
  - CSAM/GAV API (inventory, gateway JWT): <https://docs.qualys.com/en/csam/api/> and
    <https://cdn2.qualys.com/docs/qualys-gav-csam-api-v2-user-guide.pdf>
- **Relationship:** tag CRUD remains current on `qps/rest/2.0/.../am/tag` (CSAM release
  2.16, July 2023, extended these same endpoints). The CSAM/GAV **gateway** API
  (`POST https://gateway.<pod>.apps.qualys.com/rest/2.0/search/am/asset`, JWT Bearer from
  `POST /auth`) is the current *inventory read* surface — it is **not** a tag-CRUD
  replacement.
- **Sections inventoried:** tag creation, static and dynamic tags (ruleTypes), tag
  hierarchy (`parentTagId`, `children.set`), host-asset tag assignment
  (`update/am/hostasset` with `tags.add`/`tags.remove`), tag removal, untagged-asset
  discovery (limited — see doc 08), WAS web-application tag assignment.

## Family 4 — Qualys TotalCloud / CloudView APIs — RETAINED, MINIMAL SCOPE

- **Base path:** `https://qualysapi.<platform>/cloudview-api/rest/v1/...`
- **Style:** JSON REST, Basic auth, `pageNo`/`pageSize` pagination (a third pagination
  regime, distinct from both families above). Swagger UI published on the platform portal
  (`.../cloudview-api/swagger-ui.html`).
- **Documentation:**
  - CloudView/TotalCloud API guide: <https://www.qualys.com/docs/qualys-cloudview-api-user-guide.pdf>
    (versioned v1.23 at `cdn2.qualys.com/docs/version/10.20/`)
  - TotalCloud doc tree: `https://docs.qualys.com/en/tc/api/...`
- **Scope retained for this product:** only the **GCP connector** capabilities already
  shipped by this provider (`/cloudview-api/rest/v1/gcp/connectors` — create/get/list/
  modify/delete; see `cloudview/gcp/gcp-connector.go:12`). Unrelated AWS/Azure/GCP/OCI
  connector operations are **not** inventoried.
- **Deprecation alert (affects the existing resource):** CloudView connector APIs are
  being deprecated in favour of centralized **Connector v3** APIs in Asset Management —
  `POST /qps/rest/3.0/create/am/gcpassetdataconnector` (and `run`, `delete`, AWS/Azure
  equivalents). New GCP connectors can only be created from the Connectors application.
  Sources: <https://docs.qualys.com/en/conn/api/aws_3/connector_v3_apis.htm>,
  <https://docs.qualys.com/en/conn/api/gcp_3/delete_gcp_connector_3.0.htm>,
  <https://cdn2.qualys.com/docs/qualys-connectors-api-v3-user-guide.pdf>,
  <https://notifications.qualys.com/api/2026/03/04/qualys-enterprise-trurisk-platform-totalcloud-2-22-0-and-connector-2-15-0-api-notification>.
- **Hard rule:** this family must never be used to infer VMDR endpoint behaviour —
  different base path, envelope, encoding, pagination and error model.

## Cross-family summary

| Family | Base path | Encoding | Envelope | Pagination | Auth |
|---|---|---|---|---|---|
| VM/PA (VMDR) | `/api/2.0/fo/` + legacy `/msp/` | form-encoded in, XML out (DTD) | action RPC, `SIMPLE_RETURN` | `truncation_limit` + WARNING/`id_min` next-URL | Basic, session (`QualysSession`), or JWT bearer (subscription opt-in); `X-Requested-With` required |
| WAS | `/qps/rest/3.0/.../was/...` | XML (default) or JSON | `ServiceRequest`/`ServiceResponse` | `hasMoreRecords`/`lastId` + Criteria GREATER | Basic (JWT partially documented — `Unverified` per endpoint) |
| AV/CSAM tagging | `/qps/rest/2.0/.../am/...` | XML or JSON | `ServiceRequest`/`ServiceResponse` | `hasMoreRecords`/`lastId` + Criteria GREATER | Basic; CSAM gateway APIs use JWT Bearer (429 limit regime) |
| TotalCloud/CloudView | `/cloudview-api/rest/v1/` | JSON | plain REST | `pageNo`/`pageSize` | Basic (gateway TC APIs: JWT) |
