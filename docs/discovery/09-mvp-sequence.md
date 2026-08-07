# 09 — Revised MVP Sequence (based on actual supported API dependencies)

Deliverable 10. The sequence is driven by the dependency graph the discovery
confirmed, not by feature wish-lists. Arrows = "must exist first".

```
tags ──────────────┐
asset groups ──────┼─→ scan schedules ─→ (recurring scanning live)
option profiles ───┤
networks ──→ virtual scanner ──(activation code)──→ [infra provider deploys VM]
                   └─→ readiness verified ─→ schedules may reference scanner
auth records / vaults ─→ authenticated scan schedules
report templates (data source) ─→ report launches (imperative)
host detections (data source) ─→ stale-asset review ─→ purge (imperative)
```

## Phase 0 — Client foundations (blocks everything)

1. VMDR XML client: Basic + session auth (JWT optional where the subscription opts in),
   `X-Requested-With`, **secure XML parsing (no external DTD/entity resolution)**,
   `SIMPLE_RETURN` handling — **parsing `CODE`/`TEXT` rather than branching on HTTP
   status, since some calls return 200 on error** — WARNING/`id_min` pagination,
   409 limit handling with `X-RateLimit-ToWait-Sec`, endpoint-safe retry policy (no
   blind retry of destructive calls). Separate encoders for the asset-group
   create-vs-edit parameter asymmetry.
2. qps client (shared by WAS 3.0 / AM 2.0): ServiceRequest envelope, JSON preference,
   `hasMoreRecords`/`lastId` pagination. Gateway client (CSAM QQL) is a **third**
   variant — JWT bearer, **429** limit regime, no concurrency headers.
3. Tenant capability probe (doc 08) + resolution of the remaining **design-gating
   `Unverified` items**: the option-profile vulnerability-detection / authentication /
   brute-force parameter names, the not-found error code, CIDR acceptance, and the
   WAS `WasScanSchedule` recurrence element names.

## Phase 1 — Targeting primitives (P0)

4. `qualys_asset_group` (+ data source) — authoritative `set_*` updates.
5. `qualys_asset_tag` (+ data sources) — static/dynamic/hierarchy.
6. `qualys_vm_option_profile` (+ data source) — field-level CRUD; suppress
   non-round-trippable fields; model `default` and `global` as coupled (setting default
   forces global), and do not mark fields ForceNew without evidence.

   *Blocked sub-task:* the Vulnerability Detection and per-technology authentication
   parameter names are still unknown, so the first release of this resource must either
   omit those fields or gate them behind a verified parameter list.

## Phase 2 — Scanning backbone (P1)

7. `qualys_ip_registration` (add/update; destroy = state-only / opt-in purge).
8. `qualys_network` (create/update only; explicit workbook-declared selection).
9. `qualys_virtual_scanner` (create → export `activation_code` as Computed+Sensitive,
   **never overwritten with an empty read**; VLANs/routes as authoritative whole lists;
   network assignment via the separate `assign_network_id` call; optional
   `wait_for_online` with a timeout sized for the **4-hour** platform heartbeat, not
   seconds) — with documented handoff to the infrastructure provider for VM deployment.
10. `qualys_scan_schedule` (+ data source) — full recurrence/timezone/notification
    model; start-time fields written as one always-complete group behind
    `set_start_time=1`; `ACTIVE` handled as a four-valued enum, not a bool;
    distribution-group IDs referenced, never raw emails.

## Phase 3 — Authenticated scanning (P1–P2)

11. `qualys_vault`, then `qualys_auth_record_windows` / `qualys_auth_record_unix`
    (write-only secrets, vault references) + `data.qualys_auth_records`.
12. `qualys_host_tag_assignment`; `qualys_domain`.

## Phase 4 — WAS onboarding (P2)

13. `qualys_web_application` (+ tags), `qualys_was_option_profile`,
    `qualys_was_auth_record`, then `qualys_was_scan_schedule` once its field model is
    verified.

## Phase 5 — Operations subsystems (P2–P3, largely imperative)

14. Stale-asset review & purge subsystem: `data.qualys_host_detections` +
    `data.qualys_host_assets` (stale filters) → human-confirmed purge → verification
    re-read.
15. Reporting: `data.qualys_report_templates`, `data.qualys_report_schedules`,
    imperative launch/fetch helpers; excluded IPs resource.
16. Deferred backlog: report-template CRUD resources, `qualys_user`, KB helper.

## Parallel track — CloudView migration

17. Track the announced CloudView-connector deprecation; design
    `qualys_gcp_connector` v2 on Connector v3 (`/qps/rest/3.0/.../am/gcpassetdataconnector`)
    with a state-migration path, keeping the current resource working meanwhile.

## Explicitly out of the MVP (no API support — doc 08)

Report-schedule resources, distribution-group management, network deletion, appliance
replacement, physical appliance lifecycle, scan-schedule "launch now", scheduled maps.
