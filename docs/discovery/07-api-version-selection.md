# 07 — API Version Selection per Capability

Deliverable 7. Per the task rule, versions are selected **per capability**, not one
global VMDR version. "V1" = legacy `/msp/*.php`; "V2" = `/api/2.0/fo/`; "v3" =
`/api/3.0/fo/` (where observed); "qps 2.0/3.0" = portal APIs.

| Capability | Versions available | **Selected** | Rationale / compatibility notes |
|---|---|---|---|
| Host assets (list/add/update) | V2 only (current) | **V2** | current, DTD-documented; host-list DTD path changed in 10.9 — use `/asset/host/dtd/list/output.dtd` |
| Host purge | V2 | **V2** | only surface |
| Host detections | V2 | **V2** | only surface |
| Asset groups | V1 `asset_group_list.php` (legacy) vs V2 `asset/group/` | **V2** | V2 has full CRUD + `set_*` semantics; V1 list-only, predecessor flagged for retirement |
| Networks | V2 only | **V2** | no delete in any version |
| Domains | V2 (Domain V2 API) | **V2** | Basic-auth only; bulk delete needs platform ≥10.25 (Dec 2023) — minimum-release note |
| Scanner appliances | V2 only | **V2** | create returns activation code |
| Option profiles | V2 field-level (`/subscription/option_profile/vm|pc/`), V2 XML export/import, v3 variant observed | **V2 field-level** for resource CRUD; V2 export/import for backup/migration helpers; **v3 deferred** until documented delta is known (`Unverified`) | field-level params round-trip better with Terraform diffs than whole-XML import; import silently rewrites search-list/brute-force fields |
| VM scans | V2 only | **V2** | scan summary has two generations — prefer `/scan/vm/summary/` (newer, richer) where Manager role available |
| Scan schedules | V1 `scheduled_scans.php` vs V2 `schedule/scan/` | **V2** | V2 has full CRUD; V1 retained **only** if scheduled *maps* are ever required (legacy classification) |
| Reports (launch/list/fetch) | V2 (an `/api/3.0/fo/report/` DTD reference observed) | **V2** | v3 details `Unverified`; V2 fully documented |
| Report templates | list = V1 `report_template_list.php`; CRUD = V2 `report/template/<type>/` | **V2 CRUD + V1 list** | V1 list is the only listing surface — a deliberate mixed-generation selection |
| Report schedules | V2 list/launch_now only | **V2 (read/trigger only)** | schedule definition GUI-only — no version provides create/update/delete |
| Scan authentication records | V2 only | **V2** | per-type slugs; Cisco IOS/Checkpoint via `unix` slug |
| Vaults | V2 only | **V2** | — |
| Users | V1 `user.php`/`user_list.php` (claimed V2 `Unverified`) | **V1** (only confirmed surface; deferred feature) | Basic-only, no session auth |
| Time zone codes | V1 helper | **V1** | reference data only |
| Tags / tagging | qps 2.0 (`am/tag`) — current per CSAM 2.16; CSAM gateway = inventory reads only | **qps 2.0** for tag CRUD + host tagging; gateway CSAM APIs **not** selected (read-only inventory, JWT, different host) | official guidance keeps tag management on qps 2.0 |
| WAS | qps 3.0 | **qps 3.0** | JSON preferred (WAS ≥4.5) for client simplicity; XML fallback |
| Cloud connectors (GCP, existing) | CloudView v1 (deprecated) vs Connector v3 (`qps/rest/3.0/.../am/gcpassetdataconnector`) | **maintain CloudView v1 now; plan migration to Connector v3** | official deprecation announced; new GCP connectors already restricted to the Connectors app; migration is a tracked follow-up (doc 09) |

**Minimum-release implications:** Domain bulk delete ≥10.25; EC2 scan
`ec2_instance_ids` behaviours ≥10.16; option-profile import/export ≥8.10; schedule
`priority` ≥8.9. The provider should not assume features below these releases on
private-cloud platforms (PCP) that lag SaaS versions — tenant validation (doc 08)
must probe rather than assume.
