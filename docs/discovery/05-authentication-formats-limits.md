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
| Bearer / JWT | Documented for **gateway** APIs (CSAM/GAV, PM, etc.) via `POST https://gateway.<pod>.apps.qualys.com/auth` (`username`, `password`, `token=true`, form-encoded; TTL 4 h) and in the Token-Based Authentication technical brief. **No documentation shows `/api/2.0/fo/` accepting JWT** — treat as unsupported for VMDR (explicit negative `Unverified`) | Partially confirmed |
| `X-Requested-With` header | **Required on every v2 API request**, both auth modes (CSRF protection) | Confirmed |
| Content type | Form POSTs: `application/x-www-form-urlencoded`; XML-body operations (option-profile import, report-template create/update): `text/xml` | Confirmed |
| Platform base URL | Per-platform API server (`qualysapi.qualys.com`, `qualysapi.qg2.apps.qualys.com`, `qualysapi.qualys.eu`, ...); authoritative directory <https://www.qualys.com/platform-identification/>; gateway URLs are separate (`gateway.<pod>.apps.qualys.com`). Provider config must take the API server URL explicitly (existing `base_url` provider attribute) | Confirmed (full per-POD table `Unverified`) |
| User permissions | Per-endpoint; recorded per row in doc 03 (e.g. option-profile export/import = Manager; purge = Manager or granted permission; appliance network assignment = Manager; scan summary = Manager) | Confirmed per noted rows; complete per-action matrices `Unverified` |
| API add-on gating | KnowledgeBase API requires subscription-level authorization from Qualys; Network Support is a subscription feature; Report Share required for report fetch; Consultant-only fields (`client_id`/`client_name`) | Confirmed |

**Rule adopted:** authentication support is determined **per endpoint**, never assumed
identical across API versions or endpoints (Domain V2 and `/msp/user.php` prove the
exceptions exist).

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
| Breach behaviour | HTTP **409 Conflict** (not 429) with the headers above; `X-RateLimit-ToWait-Sec` = seconds to wait before the next call | Confirmed |
| Exempt endpoints | Session login/logout (`/api/2.0/fo/session/`) exempt from limits | Confirmed |
| Truncation limits | Host list & detection default page 1,000 (`truncation_limit` raisable, e.g. 10000); auth records fixed 1,000 | Confirmed |

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
