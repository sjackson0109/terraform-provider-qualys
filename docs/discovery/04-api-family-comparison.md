# 04 — Comparison: VMDR XML APIs vs WAS APIs vs TotalCloud APIs

Deliverable 3. The three families (plus the AV/CSAM tagging family, which shares the
WAS transport style) differ on every axis that matters to a shared Go client. The
provider must implement **separate client layers per family**; nothing may be shared
except HTTP plumbing, credentials handling and retry policy hooks.

| Axis | VMDR XML API (VM/PA) | WAS API | AV/CSAM Tagging | TotalCloud / CloudView |
|---|---|---|---|---|
| Base path | `/api/2.0/fo/...` (+ V1 `/msp/*.php`) | `/qps/rest/3.0/<op>/was/<object>` | `/qps/rest/2.0/<op>/am/<object>` | `/cloudview-api/rest/v1/...` |
| Host | `qualysapi.<platform>` | `qualysapi.<platform>` | `qualysapi.<platform>` (CSAM inventory: `gateway.<pod>.apps.qualys.com`) | `qualysapi.<platform>` |
| Model | RPC via `action=` parameter | operation-in-URL REST-ish | operation-in-URL REST-ish | plain REST |
| Request encoding | form-encoded params; some ops take XML bodies (OP import, report templates) | XML `ServiceRequest` (JSON since WAS 4.5) | XML/JSON `ServiceRequest` | JSON |
| Response encoding | XML + per-endpoint DTD; `SIMPLE_RETURN` for mutations | `ServiceResponse` XML/JSON; XSD at `/qps/xsd/3.0/was/*.xsd` | `ServiceResponse` XML/JSON | JSON |
| Pagination | `truncation_limit` + `WARNING` (code 1980) with next-URL `id_min`; auth records fixed 1,000/page | `hasMoreRecords` + `lastId` + Criteria `id GREATER` | same as WAS | `pageNo` / `pageSize` |
| Auth | Basic **or** session (`QualysSession` cookie); **`X-Requested-With` mandatory**; exceptions: Domain V2 and `/msp/user.php` are Basic-only | Basic (JWT partially documented, `Unverified` per endpoint) | Basic (qps); CSAM gateway = JWT Bearer (from `POST /auth`, 4 h TTL) | Basic (gateway TC APIs: JWT) |
| Error model | XML `SIMPLE_RETURN`/response codes; HTTP 409 on limit breach | `responseCode` in envelope (e.g. `SUCCESS`, error codes) | same as WAS | HTTP status + JSON body |
| Async ops | scans & reports return REF/ID immediately; poll status | wasscan launch → poll `status/was/wasscan/<id>` | n/a | connector run is async |
| IDs | numeric per-object; scan REF `scan/<epoch>.<n>` | numeric object IDs | numeric tag/asset IDs | connector UUID |
| Deprecation channel | notifications.qualys.com/api + release notes | same | same | same — **CloudView connector APIs deprecated in favour of Connector v3 (`/qps/rest/3.0/.../am/*assetdataconnector`)** |

## Consequences for the provider design

1. **Three transport stacks** (form/XML action RPC; qps ServiceRequest; JSON REST) and
   **three pagination engines**. No shared response parsing.
2. **Secure XML parsing is a VMDR/WAS-family concern**: responses reference DTDs
   (`<!DOCTYPE ... SYSTEM>`); the parser must disable external entity resolution and
   never fetch remote DTDs (doc 05). JSON families are unaffected.
3. **Session-auth optimisation applies only to the VMDR family**, and not uniformly
   (Domain V2 and V1 `/msp/user.php` are Basic-only). The client must select auth mode
   per endpoint, not globally.
4. **Rate-limit handling** keys off VMDR `X-RateLimit-*`/`X-Concurrency-*` headers +
   HTTP 409; qps and CloudView families do not document the same headers
   (`Unverified`) — retry policy must be family-specific.
5. **The TotalCloud/CloudView family is quarantined**: nothing learned from its
   Swagger/JSON behaviour may inform VMDR client behaviour, and its only retained
   capability (GCP connector) is on an announced deprecation path toward Connector v3
   (`/qps/rest/3.0/create|run|delete/am/gcpassetdataconnector`).
6. **Tag identity across families** is operationally central (VMDR scan targeting by
   tag, WAS webapp tags, AV tag CRUD) but formally `Unverified` — see docs 03 §14 and 08.
