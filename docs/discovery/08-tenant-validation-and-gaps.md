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
| Rate/concurrency ceilings | subscription tier (headers observed at runtime are authoritative) |

Recommended pattern: a `provider helper` "tenant capability probe" that runs cheap
list calls (`network?action=list`, `appliance?action=list`, WAS `count/was/webapp`,
`search/am/tag` with limit 1) and caches which families/features are live, plus a
`validation operation` mode for the onboarding workbook (validate network IDs,
scanner names, option-profile IDs, tag IDs, template IDs before apply).

## Deliverable 9 — Capabilities for which official API documentation could not be found

Confirmed-absent (documented nowhere; community evidence corroborates absence):

1. **Report schedule create/update/delete/enable/disable** — GUI-only; API has only
   `list` + `launch_now`.
2. **Distribution group CRUD** (report/scan notification recipients) — UI-only.
3. **Network delete** at API level (UI delete exists).
4. **Host asset/IP delete** (purge is the only API-side removal; official article
   000003463).
5. **Scanner appliance replace** — UI wizard only.
6. **Physical appliance create/delete** — only `physical/?action=update` exists.
7. **"Launch now" for scan schedules** — no such action (only report schedules have it).
8. **Remediation / compliance-policy report template CRUD** — template APIs cover
   Scan, PCI Scan, Patch, Map only.
9. **Vulnerability scorecards other than vulnerability type via API** — compliance and
   WAS scorecards not launchable by API.

`Unverified` items to resolve before implementation (from all matrix rows; the ones
that gate design decisions are bolded):

- **CIDR notation acceptance on `asset/ip/?action=add`** (assume ranges-only; expand
  CIDR client-side).
- **Exact purge parameter names** (`data_scope=vm|pc`, selector params, `echo_request`).
- **Asset-group `set_*` parameter names beyond `set_ips`** (domains, dns/netbios
  names, appliance IDs) — gates `qualys_asset_group` schema.
- **Cross-module tag identity (same tag ID across VMDR / AssetView / WAS)** — strong
  official indications, no explicit statement; must stay `Unverified` until confirmed.
- **Untagged-asset discovery**: no NOT/NONE operator on `search/am/hostasset`
  (community says unsupported) → plan client-side set difference; confirm.
- `include_ignored` and first/last-found input filters on host detection list.
- Host-detection output root element + DTD path.
- Appliance `set_vlans`/`set_routes` replace-vs-merge semantics; full `output_mode=full`
  element list; readiness values (`SS_*` unconfirmed as API values).
- VM option profile `update` doc page; `IS_DEFAULT` writability; fields forcing
  replacement; `/api/3.0/` option-profile endpoint delta.
- Scan-schedule `start_date`/end-date params; `ACTIVE` output values 2/3;
  `client_id`/`client_name` as schedule inputs; `before_notify_message`.
- Report types (beyond vuln/compliance) accepting `use_tags`; `report_refs`
  required/optional matrix; max concurrent running reports; `/api/3.0/fo/report/`
  variant details.
- Excluded-hosts change-history endpoint path; excluded-IP `remove` params.
- Vault per-type parameter schemas; `/api/2.0/fo/vault/index.php` path form; VM-side
  per-type password echo behaviour.
- WAS: `delete/was/optionprofile`, `cancel|delete/was/wasscan`, full `wasscanschedule`
  CRUD + recurrence fields, WAS report-schedule API, per-endpoint JWT support, current
  guide version numbers.
- Tagging: `CLOUD_ASSET` ruleType; `lastVulnScan` hostasset filter.
- V2 user API existence (`/api/2.0/fo/user/` — third-party claim only).
- Formal deprecation status of `/msp/scheduled_scans.php`, `/msp/user.php`,
  `/msp/asset_group_list.php`.
- Scan-list/appliance-list/OP-list truncation mechanisms (none surfaced — absence
  unproven).
- Not-found error codes per endpoint (needed for Terraform state removal semantics).
- Exact per-tier rate/concurrency numbers; complete per-POD platform URL table.
- qps (WAS/AM) and CloudView rate-limit header behaviour.

> Resolution path: re-run targeted checks from an environment with direct access to
> docs.qualys.com (the discovery environment's network policy blocked it — see README),
> then confirm empirically against a test subscription before each affected schema is
> frozen.
