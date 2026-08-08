# 07 — API Version Selection per Capability

> ## ⚠️ Revision — V2 is not a safe default any more
>
> A compiled reference of the VM/PC API guide (supplied for this discovery; derived
> secondary source) documents an **EOS/EOL migration programme already in flight**.
> Each superseded path gets **End of Support 6 months** after the newer version ships,
> then **End of Life 6 months after that**, at which point it is decommissioned.
>
> Migrations named as in flight:
>
> | Capability | Migrating to |
> |---|---|
> | Scan API | **v3.0** |
> | Schedule Scan API | **v5.0** |
> | Option Profile APIs | **v4.0–v6.0** (varies by sub-type) |
> | Asset Host / VM Detection API | **v5.0** |
> | Report Template Scan API | **v7.0** |
> | Compliance Control List | **v4.0** |
> | PC Posture Info | **pcrs v5.0** |
>
> Combined with the product decision to remove deprecated items, this inverts the
> original guidance. **The table below still records what is fully documented at V2,
> but V2 must now be treated as a starting point to migrate off, not a destination.**
>
> **Action before any client code is written:** confirm the current version and EOS
> date for each capability the MVP touches — the guide says the EOS date is carried in
> the response payload / docs per endpoint, so this is checkable at runtime against a
> tenant. Build the client with the **API version as a per-capability constant** (it
> already must be, since versions differ per capability) so a bump is a one-line change,
> and log the EOS date when a deprecated version is in use.
>
> Where a newer version's schema is not yet documented (the 2.0→4.0/5.0 option-profile
> delta, the 2.0→3.0 report delta), that gap is now **blocking rather than deferred**,
> because pinning V2 has a known expiry. See doc 08.

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
| Option profiles | V2 field-level (`/subscription/option_profile/vm|pc/`), V2 XML export/import, **plus newer major versions 4.0 (Release 10.36) and 5.0/7.0 (Release 10.39.1)** — there is **no 3.0** | **V2 field-level** for resource CRUD; V2 export/import for backup/migration helpers; **4.0/5.0/7.0 deferred** until the field/encoding delta is documented (`Unverified`) | field-level params round-trip better with Terraform diffs than whole-XML import; import silently rewrites search-list/brute-force fields. Note 5.0 carries at least one parameter V2 lacks (`allow_on_host_script_execution`), so pinning V2 forfeits newer options — revisit once the delta is known |
| VM scans | V2 only | **V2** | scan summary has two generations — prefer `/scan/vm/summary/` (newer, richer) where Manager role available |
| Scan schedules | V1 `scheduled_scans.php` vs V2 `schedule/scan/` | **V2** | V2 has full CRUD; V1 retained **only** if scheduled *maps* are ever required (legacy classification) |
| Reports (launch/list/fetch) | V2, plus a v3 endpoint confirmed at least for `action=fetch` | **V2** | V2 fully documented and still maintained (updated in Release 10.32); v3 scope/delta `Unverified` |
| Report templates | list = V1 `report_template_list.php`; CRUD = V2 `report/template/<type>/` | **V2 CRUD + V1 list** | V1 list is the only listing surface — a deliberate mixed-generation selection |
| Report schedules | V2 list/launch_now only | **V2 (read/trigger only)** | schedule definition GUI-only — no version provides create/update/delete |
| Scan authentication records | V2 only | **V2** | per-type slugs; Cisco IOS/Checkpoint via `unix` slug |
| Vaults | V2 only | **V2** | — |
| Users | V1 `user.php`/`user_list.php` (claimed V2 `Unverified`) | **V1** (only confirmed surface; deferred feature) | Basic-only, no session auth |
| Time zone codes | V1 helper | **V1** | reference data only |
| Tags / tagging | qps 2.0 (`am/tag`) — current per CSAM 2.16; CSAM gateway = inventory reads only | **qps 2.0** for tag CRUD + host tagging; gateway CSAM APIs **not** selected (read-only inventory, JWT, different host) | official guidance keeps tag management on qps 2.0 |
| WAS | qps 3.0 | **qps 3.0** | JSON preferred (WAS ≥4.5) for client simplicity; XML fallback |
| Cloud connectors (GCP, existing) | CloudView v1 (deprecated) vs Connector v3 (`qps/rest/3.0/.../am/gcpassetdataconnector`) | **Connector v3** — migrate off CloudView v1 | **Product decision: deprecated items are being removed.** New GCP connectors are already restricted to the Connectors application, so CloudView v1 is a dead end. The existing `qualys_gcp_connector` resource is re-pointed at Connector v3 with a state-migration path (doc 09) |

**Minimum-release implications:** Domain bulk delete ≥10.25; EC2 scan
`ec2_instance_ids` behaviours ≥10.16; option-profile import/export ≥8.10; schedule
`priority` ≥8.9; option-profile `allow_on_host_script_execution` ≥10.39.1 (and only on
API 5.0/7.0); scan-schedule `ACTIVE` values 2/3 postdate the 10.15 DTD. The provider
should not assume features below these releases on private-cloud platforms (PCP) that
lag SaaS versions — tenant validation (doc 08) must probe rather than assume.

**Versioning contract (Confirmed):** Qualys increments the major version only for
breaking changes, and older versions remain callable
(<https://notifications.qualys.com/api/2024/05/17/introducing-api-versioning-a-strategic-upgrade-for-enhanced-stability-and-control-for-api-integrations>);
since Aug 2025 a deprecated version gets 12 months (6 months End of Support + 6 months
EOL). Pinning V2 per capability is therefore safe in the near term, but each pin needs a
scheduled re-review rather than being treated as permanent.
