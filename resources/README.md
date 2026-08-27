# Qualys resources bundle

A reusable Terraform-root template for driving `sjackson0109/terraform-provider-qualys`
entirely from a named-Excel-table workbook. Copy or invoke it from a
consuming automation repository — `alltimeuk/qualys` is one such consumer,
and this bundle exists to give it (and any future one) a working starting
point rather than each rebuilding the same pattern independently.

Files:

- `Qualys_Configuration_Template_All_Resources.xlsx`: 52 named Excel data
  tables, one per Terraform-managed resource type, plus `qualys_business_unit`
  and `qualys_user` for the GUI-only sync below. Worksheet names are
  organisational only — the converter discovers tables by their Excel
  *table* name, not worksheet name or position.
- `convert-qualys-xlsx.yml`: Step 1 converter. Reads every named Excel
  table across every worksheet and emits one JSON document. Intentionally
  has no schema of its own — whatever columns a table has become that
  table's JSON fields, `__list`/`__json` suffixes aside — so keeping the
  workbook and `terraform.tf` in sync is the operator's responsibility,
  not something this converter checks.
- `provider.tf` / `variables.tf` / `terraform.tf`: the Terraform root.
  Every managed resource uses `for_each` over
  `try(local.qualys_config.<resource_type>, [])`, so a missing/empty
  collection creates zero resources.
- `qualys_gui_sync.py` / `.github/workflows/qualys-gui-sync.yaml`: Step 2.
  Creates/updates/deletes **business units** and **user accounts** from the
  workbook via the legacy GUI session endpoints — these two objects have no
  API at all, so neither the Terraform provider nor Step 1's JSON can reach
  them.

The YAML is intentionally stored under `resources/` in the provider
repository as a reusable workflow asset/template. GitHub will only execute
workflow files located under `.github/workflows/` in the repository that is
actually running the workflow. Copy or invoke the template from the
consuming automation repository as appropriate.

Provider authentication is via `QUALYS_URL`, `QUALYS_USERNAME`, and
`QUALYS_PASSWORD`.

List-valued workbook columns use `__list` and semicolon-separated input.
JSON-valued/nested columns use `__json` and contain valid JSON. Both
suffixes are removed by the converter.

## Managed vs external references

- `<name>_key` / `<name>_keys__list` — logical reference to the `key` of a
  managed row in another table. Terraform resolves it to the real Qualys id
  (`qualys_network.managed["corp"].id`). Operators never enter numeric ids
  for managed objects.
- `external_<name>_name` / `external_<name>_names__list` (or `_title(s)`
  for objects Qualys itself calls by title — asset groups, static search
  lists) — the human-readable name/title of an object **not** managed by
  this root (an existing network, asset group, scanner appliance, tag,
  vault, WAS web application/report template/authentication record/DNS
  override, static search list, VM/WAS option profile, admin user, role…).
  Terraform resolves it to the id the provider actually needs via a
  `data "qualys_<type>s"` lookup in `terraform.tf`'s "Reference resolution"
  section. **The workbook never carries a raw Qualys numeric id.** A small,
  fixed set of object types have no Qualys API to look them up by at all
  (WAS proxies, distribution groups, findings, AssetView/CSAM host asset
  ids); those stay genuinely id-only, documented at each such column with a
  comment explaining why.

The two are never ambiguous: they are separate columns, and Terraform
prefers the `_key` branch when both happen to be set on the same row (see
each resource block's `try(each.value.<x>_key, null) != null ? ... : ...`).

### Resolving external references by name

Every `external_<name>_name`/`external_<name>_names__list` column resolves
the same way, in the "Reference resolution" section near the top of
`terraform.tf`: one `data` source per object type, guarded by `count` so it
only runs when `var.resolve_references_by_name` is true *and* at least one
table that could use it is non-empty, feeding a `locals` map from
name/title to id that every consuming resource block reads from.

None of these import anything into state — they only resolve a value for
an ordinary reference field (`network_id`, `tag_ids`, `vault_id`, …), so a
name that doesn't match an existing object simply fails the lookup at plan
time with an ordinary Terraform "key does not exist" error. A single lookup
commonly feeds several unrelated resource blocks — `external_vault_name`
alone feeds all 30 authentication record types — so this is one API call
per object type, not one per field; that is also why the `data` blocks are
grouped in one section rather than placed beside each caller.

`qualys_user_scope_assignment`'s `external_user_name`/
`external_role_names__list` are a variant of the same idea, both resolved
from `data.qualys_am_users`: `external_user_name` against every returned
user's `username`, and each role name against the `{id, name}` pairs
already nested in every user's own `roles` list — there is no dedicated
"list roles" endpoint, so a role name only resolves if at least one user in
the tenant already has that role assigned.

`vault_key`/`external_vault_name` resolve both `vault_id` *and*
`vault_type` together: `vault_type` is a fixed property of the vault
itself, so knowing which vault also gives you that, rather than requiring a
second, easy-to-mismatch manually-entered `vault_type` column (dropped
entirely from the workbook for this reason).

Set `-var="resolve_references_by_name=false"` to disable it for offline
planning against fixture JSON with no real Qualys credentials.

## Business units and users (GUI-only)

The provider has no API for business units or user accounts, so these are
managed by `qualys_gui_sync.py`, which drives the same `/fo/options/*.php`
endpoints the Qualys web UI uses. That requires a browser-style session (a
login cookie plus a per-form `_form_id` CSRF token), so it cannot ride the
provider's stateless Basic-auth client — and consequently these two tables
are entirely outside `terraform.tf`/Step 1's JSON; `qualys_gui_sync.py`
reads the XLSX directly.

Two named tables drive it:

- `qualys_business_unit`: `enabled, key, title, asset_groups__list, comments, delete`
- `qualys_user`: `enabled, key, first_name, last_name, title, phone, fax, email, address_1, address_2, city, country, state, zip_code, language, date_format, time_zone, user_role, business_unit_title, gui, api, manage_vm, modify_remedy_policy, add_vhost, add_asset, option_profile, purge_host, modify_ntauth, manage_compliance, approve_exceptions, modify_compliance_policy, create_user_defined_control, modify_all_user_defined_control, manage_webapp, create_webapps, latest_vulnerabilities, scan_complete_notification, scan_notification, map_notification, report_notification, exception_notification, user_session_timeout, delete`

The user-to-business-unit relationship is set by the user's
`business_unit_title` (a user belongs to a business unit), so business
units are synced first and users reference them by title. Business-unit
`asset_groups__list` holds asset-group **titles**; they are resolved to ids
via the VMDR API when `QUALYS_API_URL` is set, otherwise passed through
as-is.

Environment for the script:

- `QUALYS_GUI_URL` — GUI host, e.g. `https://qualysguard.qg2.apps.qualys.eu`
- `QUALYS_USERNAME`, `QUALYS_PASSWORD` — GUI login
- `QUALYS_API_URL` — optional API host for asset-group title resolution

Idempotency is tracked in a state file (default `qualys_gui_state.json`)
mapping business-unit title -> id and user email -> id. First run creates
and records ids; later runs update in place. Rows with `delete` set are
deleted and removed from state. Because the state file is the source of
truth for ids, objects created or deleted out-of-band in the Qualys UI are
not detected.

Run locally:

```
QUALYS_GUI_URL=https://qualysguard.qg2.apps.qualys.eu \
QUALYS_USERNAME=... QUALYS_PASSWORD=... \
python resources/qualys_gui_sync.py resources/Qualys_Configuration_Template_All_Resources.xlsx --dry-run
```

The workflow requires GitHub secrets `QUALYS_GUI_URL`, `QUALYS_USERNAME`,
`QUALYS_PASSWORD`, and optionally `QUALYS_API_URL`.

## Known gaps

- **Credentials are stored as plain values in the workbook/JSON** (auth
  record `password`, connector `gcp_credentials_json`/`authentication_key`,
  vault `parameters`). Unlike `alltimeuk/qualys`'s `*_secret` /
  `secret:NAME` indirection through a GitHub Environment secrets map, this
  template has no such layer yet — a real deployment should not commit a
  filled-in copy of this workbook or its generated JSON anywhere secrets
  would be retained.
- **No test coverage.** Every claim in this file about `terraform.tf`
  parsing, planning, and resolving references correctly was verified
  manually against a real `terraform` binary and a real provider build —
  see the PR that introduced the current reference-resolution section for
  the exact commands — but nothing here runs that verification
  automatically the way `alltimeuk/qualys`'s `tests/` suite does for its
  own fork of this pattern.
