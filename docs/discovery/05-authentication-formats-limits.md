# 05 — Authentication Discovery, Request/Response Formats, API Limits & Retries

Covers task sections 3 (authentication), 4 (request/response formats) and 6 (limits
and retries). Per-operation format details (method, path, action, params, response
root/DTD, pagination) live in the matrix (doc 03); this document records the
cross-cutting rules and the exceptions.

## 3.x Authentication discovery (VMDR endpoints)

| Property | Finding | Status |
|---|---|---|
| Basic authentication | Supported on all `/api/2.0/fo/` endpoints and V1 `/msp/` endpoints | Confirmed |
| Session authentication | `POST /api/2.0/fo/session/` `action=login` → `QualysSession` cookie (`Set-Cookie`); `action=logout` ends it. Supported across `/api/2.0/fo/` **except**: Domain V2 API (`/api/2.0/fo/asset/domain/`) and V1 user API (`/msp/user.php`) are **Basic-only** | Confirmed |
| Bearer / JWT | **`/api/2.0/fo/` endpoints DO accept JWT bearer tokens** — two mechanisms: Qualys OpenID Connect API authentication (external IdP) and Qualys-native token auth via `/auth/oauth`. Official wording: *"This authentication is supported by all Qualys APIs, /api/2.0/ and onward versions"*, with a worked example against `/api/2.0/fo/asset/ip/`. Qualifiers: **opt-in per subscription** (requires Qualys Support activation), Basic auth continues to work alongside it (either/or, not a replacement), token TTL **4 hours**. The gateway `/auth` endpoint (`username`/`password`/`token=true`) is a *separate*, always-on mechanism for CSAM/GAV/EDR/FIM/Container Security | **Confirmed** — this reverses the initial "gateway-only" reading |
| `X-Requested-With` header | **Required on every v2 API request**, both auth modes (CSRF protection) | Confirmed |
| Content type | Form POSTs: `application/x-www-form-urlencoded`; XML-body operations (option-profile import, report-template create/update): `text/xml` | Confirmed |
| Platform base URL | Per-platform API server (`qualysapi.qualys.com`, `qualysapi.qg2.apps.qualys.com`, `qualysapi.qualys.eu`, ...); authoritative directory <https://www.qualys.com/platform-identification/>; gateway URLs are separate (`gateway.qg1.apps.qualys.com`, `.qualys.eu`, `.qualys.in`, `.qualys.ca`, `.qualys.ae`, ...). Functional split: **gateway URLs** for Asset Inventory/CSAM, EDR, FIM and Container Security; **API server URLs** for everything else. **Design decision: the provider must expose `api_url` (and later `gateway_url`) as explicit configuration and must not ship a hardcoded POD map** — the full table could not be confirmed, and one source shows a `.co.uk` vs `.uk` conflict for UK1 | Confirmed (split + several gateway hostnames); full per-POD table `Unverified` |
| User permissions | Per-endpoint; recorded per row in doc 03 (e.g. option-profile export/import = Manager; purge = Manager or granted permission; appliance network assignment = Manager; scan summary = Manager) | Confirmed per noted rows; complete per-action matrices `Unverified` |
| API add-on gating | KnowledgeBase API requires subscription-level authorization from Qualys; Network Support is a subscription feature; Report Share required for report fetch; Consultant-only fields (`client_id`/`client_name`) | Confirmed |

**Rule adopted:** authentication support is determined **per endpoint**, never assumed
identical across API versions or endpoints (Domain V2 and `/msp/user.php` prove the
exceptions exist).

## 4.0 Request mechanics — rules the client must encode

From the compiled VM/PC API reference (derived secondary source; each is a
straightforward client rule, so low risk to adopt):

- **GET or POST** on most functions; POST preferred for large parameter sets because
  GET hits practical URL-length limits.
- **A repeated parameter keeps only its last instance** — the client must never emit a
  key twice (a real hazard when merging defaults with user config).
- **All parameters must be URL-encoded**, including `#` → `%23`; unencoded, the server
  treats the remainder as a URL fragment and silently drops it.
- **URL elements are case-sensitive**: `?action=list` works, `?Action=list` does not.
- Dates are **RFC 3339 / ISO 8601** UTC: `yyyy-mm-ddThh:mm:ssZ`.
- Responses are **UTF-8 XML** by default; some endpoints offer JSON/CSV via
  `output_format`.
- Passwords sent in a **POST body must be URL-encoded** (`+` → `%2B`); passwords sent
  via HTTP Basic do not.
- **Session timeout** defaults to 60 minutes (configurable per subscription). Jobs
  already launched in the background do not time out mid-processing.
- **Scan launch silently skips out-of-scope targets.** If a target IP is not in the
  user's scope or licence, it is dropped from the job rather than failing the request —
  so a "successful" launch may cover fewer assets than declared. Any scan-adjacent
  validation must compare requested vs actual targets rather than trusting success.

## 4.x Request/response format rules (VMDR)

- HTTP method, endpoint path, `action` parameter, form parameters, XML response root,
  referenced DTD, pagination and async behaviour: recorded per operation in doc 03.
- Success: HTTP 200 + expected XML root; mutations return `SIMPLE_RETURN` with
  `TEXT` (human message) + `ITEM_LIST/ITEM` `KEY`/`VALUE` pairs (created IDs).
- Errors: XML response with error code/text (per-endpoint DTDs); limit breaches use
  HTTP **409**; auth failures HTTP 401.
- Not-found behaviour: typically an empty list for `list` actions and an error
  `SIMPLE_RETURN` for mutations on missing IDs — **exact codes per endpoint
  `Unverified`; must be pinned during client implementation against a test tenant**
  (Terraform needs deterministic 404-equivalent detection for state removal).
- Truncation/pagination: `truncation_limit` + `WARNING` code 1980 with continuation
  URL (`id_min`) for host/detection lists; fixed 1,000-record pages for auth records;
  no truncation mechanism surfaced for scan/appliance/option-profile lists
  (`Unverified` absence).
- Async: scan launch → REF; report launch → ID; poll via `list` actions.

### Secure XML parsing (implementation requirement)

Responses reference DTDs via `<!DOCTYPE ... SYSTEM "https://qualysapi...dtd">`. The
implementation **must not retrieve remote DTDs** and must parse with external entity
and external DTD resolution disabled. For Go: use `encoding/xml` (which does not fetch
DTDs or resolve external entities), set `Decoder.Strict = true`, and reject any
`Directive`/`ENTITY` expansion beyond predefined entities; never route the DOCTYPE
SYSTEM URL through an HTTP fetch. DTD URLs are recorded in the matrix as schema
*documentation*, not as runtime resources.

## 6.x API limits and retries

| Item | Finding | Status |
|---|---|---|
| Concurrency limit | Default **2 concurrent calls per API endpoint per subscription** (tier-dependent) — checked before rate limit | Confirmed |
| Rate limit | Standard tier: **300 calls / 3600 s** sliding window; window looks back 1 hour (1 day for Express Lite/Consultant). Exact numbers for other tiers `Unverified` (in `qualys-api-limits.pdf`) | Confirmed / partial |
| Limit headers | `X-RateLimit-Limit`, `X-RateLimit-Window-Sec`, `X-RateLimit-Remaining`, `X-RateLimit-ToWait-Sec`, `X-Concurrency-Limit-Limit`, `X-Concurrency-Limit-Running` | Confirmed |
| Breach behaviour | **Two regimes.** QWEB/FO + QPS (`/api/2.0/fo/`, `/qps/rest/2.0/`, `/qps/rest/3.0/`): HTTP **409 Conflict** with the six headers above; `X-RateLimit-ToWait-Sec` = seconds to wait. **Gateway APIs** (`gateway.<pod>.apps.qualys.com`): HTTP **429 Too Many Requests**, only `X-RateLimit-Limit`/`X-RateLimit-Remaining`, **no concurrency headers**, and limits *"enforced uniformly across all subscriptions on a particular platform"* — no per-subscription tiering. CloudView's regime is `Unverified` | Confirmed (409 and 429 regimes); QPS-specifically-409 by inference |
| Concurrency vs rate ordering | Concurrency is evaluated **before** the rate limit; the concurrency error takes precedence | Confirmed |
| Exempt endpoints | Session login/logout (`/api/2.0/fo/session/`) exempt from limits | Confirmed |
| Per-API overrides | No separate rate/concurrency tier found for Host List Detection; its override is **pagination only** (`truncation_limit`, default 1,000, e.g. 10000) | Confirmed |
| Truncation limits | Host list & detection default page 1,000 (`truncation_limit` raisable, e.g. 10000); auth records fixed 1,000 | Confirmed |

### Operational details observed in a working third-party client

Read directly from the `qualysdk` source (`base/call_api.py`) — `Corroborated
(non-official)`, but these are behaviours a real client had to handle:

- **`X-RateLimit-ToWait-Sec` is not always present on the first limited response.**
  The client re-issues the request specifically to obtain the header, commenting that
  Qualys *"sometimes only includes this header when the rate limit is reached and
  retried"*. A retry loop that requires the header on the first 409/429 will stall.
- **429 occurs in practice beyond the gateway.** The client's error path excludes
  `429` (and `414`) from generic failure handling across *all* modules, including
  `vmdr`, `was`, `tagging` and `cloudview` — so the provider should treat 409 **and**
  429 as limit signals on every family rather than assuming one per family.
- **Limit breaches can arrive as a 409 whose meaning is only in the body** — the client
  special-cases a 409 whose `SIMPLE_RETURN/RESPONSE/TEXT` contains *"This API cannot be
  run again for another…"*. Reinforces the parse-`CODE`/`TEXT` rule above.
- **Some modules return no rate-limit headers at all** (Patch Management, as of the
  client's 12-2024 note), so header-driven pacing needs a no-header fallback.
- Errors surface consistently at `SIMPLE_RETURN/RESPONSE/TEXT` for XML modules; HTML
  error bodies also occur and must not be fed to the XML path.

### Retry policy the provider must implement (endpoint-safe)

1. On 409 with `X-RateLimit-ToWait-Sec`: wait that value (+ jitter), then retry —
   **only for idempotent reads** and for mutations proven un-executed.
2. On 409 concurrency (`X-Concurrency-Limit-Running` ≥ limit): bounded exponential
   backoff.
3. **Never blind-retry destructive or non-idempotent operations** (purge, delete,
   scan/report launch, appliance create — which consumes licenses): on ambiguous
   failure (timeout after send), re-read state (`list`) to determine whether the
   operation applied before any retry.
4. Respect `Retry-After`-equivalent (`X-RateLimit-ToWait-Sec`) rather than fixed
   backoff when present.
5. Serialise per-endpoint calls to stay under the concurrency default of 2; make the
   ceiling configurable per subscription tier.
