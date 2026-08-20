# terraform-provider-qualys

A Terraform provider for [Qualys](https://www.qualys.com/) VMDR and WAS
(Web Application Scanning) configuration.

### Breaking: resources/data sources renamed for naming consistency

Where the same concept exists in both VM/PA and WAS, both sides are now
consistently `vm_`/`was_`-prefixed (matching the convention `qualys_vm_option_profile`/
`qualys_was_option_profile` and `data.qualys_vm_findings`/`data.qualys_was_findings`
already established); and where a VM/PA-only concept could be confused with
a future WAS equivalent, it is prefixed `vm_` too:

| Old name | New name |
|---|---|
| `qualys_scan_schedule` | `qualys_vm_scan_schedule` |
| `data.qualys_scan_schedules` | `data.qualys_vm_scan_schedules` |
| `data.qualys_report_schedules` | `data.qualys_vm_report_schedules` |
| `data.qualys_report_templates` | `data.qualys_vm_report_templates` |
| `qualys_web_application` | `qualys_was_application` |
| `data.qualys_web_applications` | `data.qualys_was_applications` |
| `data.qualys_scans` | `data.qualys_vm_scans` |
| `data.qualys_option_profiles` | `data.qualys_vm_option_profiles` |
| `data.qualys_auth_records` | `data.qualys_vm_auth_records` |

If you have existing state referencing the old names, update your
configuration to the new resource/data source type and run, for each
affected instance:

```shell
terraform state mv 'qualys_scan_schedule.example' 'qualys_vm_scan_schedule.example'
```

Data sources need no state migration — just update the type name in
configuration.

## Supported resources

VM/PA (asset, network and scanning configuration):

| Resource | Purpose |
|---|---|
| `qualys_asset_group` | A named set of scan targets, referenced by scans, schedules and reports |
| `qualys_asset_tag` | Static and dynamic asset tags, shared across VM, AssetView and WAS |
| `qualys_static_search_list` | An explicit set of vulnerability QIDs |
| `qualys_vm_option_profile` | VM scan behaviour: ports, detection depth, performance, authentication |
| `qualys_network` | Qualys networks, for overlapping IP space |
| `qualys_ip_registration` | Registers addresses with the subscription and enables them for scanning |
| `qualys_vm_scan_schedule` | A recurring VM scan |
| `qualys_virtual_scanner` | A virtual scanner appliance, exporting the activation code for VM deployment |
| `qualys_auth_record_windows` | A Windows scan authentication record |
| `qualys_auth_record_unix` | A Unix/Linux (SSH) scan authentication record |
| `qualys_auth_record_*` (28 more) | Every other Confirmed scan authentication record type — databases, web/app servers, network devices, hypervisors, containers; see [`docs/resources/auth_record_other_types.md`](docs/resources/auth_record_other_types.md) |
| `qualys_vault` | A credential vault; `title`/`type` are managed, per-type connection parameters pass through verbatim |
| `qualys_excluded_ips` | Excludes IPs from scanning |
| `qualys_host_tag_assignment` | Assigns asset tags to an AssetView/CSAM host asset |
| `qualys_user_scope_assignment` | Assigns scope tags and roles to a subscription user via the Administration RBAC API |
| `qualys_gcp_connector` | GCP asset data connector (Connector v3; see *Connector v3 migration* below) |
| `qualys_aws_connector` | AWS asset data connector (Connector v3) |
| `qualys_azure_connector` | Azure asset data connector (Connector v3) |

WAS (Web Application Scanning):

| Resource | Purpose |
|---|---|
| `qualys_was_application` | The scan target that option profiles, auth records and scan schedules reference |
| `qualys_was_option_profile` | WAS scan behaviour, referenced by web application scans and scan schedules |
| `qualys_was_auth_record` | Form (`STANDARD`/`CUSTOM`/`SELENIUM`), server or OAuth2 credentials for authenticated crawling |
| `qualys_was_dns_override` | Resolves one or more hostnames to user-defined IP addresses during crawling and scanning |
| `qualys_was_scan_schedule` | A recurring WAS scan against a single web application |
| `qualys_was_report_schedule` | A recurring, automated WAS report generation and delivery |
| `qualys_was_finding_ignore` | Declares a WAS finding ignored, with an audit-trail comment |

## Supported data sources

| Data source | Purpose |
|---|---|
| `qualys_asset_groups` | Look up asset groups |
| `qualys_asset_tags` | Look up asset tags |
| `qualys_networks` | Look up networks |
| `qualys_host_assets` | Look up registered host assets |
| `qualys_host_detections` | Look up hosts together with their last VM scan time (stale-asset review) |
| `qualys_vm_findings` | Look up individual VM vulnerability findings, one host + one QID + one detection instance |
| `qualys_scanner_appliances` | Look up scanner appliances |
| `qualys_vaults` | Look up credential vaults |
| `qualys_vm_scan_schedules` | Look up VM scan schedules |
| `qualys_vm_option_profiles` | Look up VM scan option profiles |
| `qualys_vm_report_templates` | Look up VM/PA report templates |
| `qualys_domains` | Look up asset domains and their netblocks |
| `qualys_was_applications` | Look up WAS web applications |
| `qualys_was_findings` | Look up individual WAS scan findings, optionally enriched from the Qualys KnowledgeBase |
| `qualys_was_report_templates` | Look up WAS report templates |
| `qualys_gcp_connector` | Look up a GCP connector by numeric id or name |
| `qualys_aws_connector` | Look up an AWS connector by numeric id or name |
| `qualys_azure_connector` | Look up an Azure connector by numeric id or name |
| `qualys_tagged_assets` | Look up AssetView/CSAM host assets by tag |
| `qualys_vm_auth_records` | Look up scan authentication records of one type |
| `qualys_vm_scans` | Look up VM scan jobs |
| `qualys_reports` | Look up generated VM/PA reports |
| `qualys_vm_report_schedules` | Look up scheduled report definitions (list-only: there is no report schedule write API) |
| `qualys_tenant_capabilities` | Probe the subscription's installed platform and per-module versions |
| `qualys_users` | Look up subscription users, roles, business unit and assigned asset groups (legacy `/msp/user_list.php`) |
| `qualys_am_users` | Look up subscription users, scope tags and roles via the modern Administration RBAC API |

### Operations the provider deliberately does not model

Some Qualys operations are jobs rather than configuration, and modelling them as
resources would misrepresent them:

- **Launching a scan or report** produces a run with no stable identity;
  re-running creates a new one rather than updating the old. Use
  `qualys_vm_scan_schedule` / `qualys_was_scan_schedule` for recurring scans, and
  `qualys_was_report_schedule` for recurring WAS reports.
- **Purging host data** is destructive and irreversible. It is available as an
  opt-in on `qualys_ip_registration` destroy, not as a resource of its own.
- **Retesting or overriding the severity of a WAS finding** are lifecycle
  actions, not declarative state, and are not modelled as resources; ignoring
  a finding is (`qualys_was_finding_ignore`), since accepted-risk/false-positive
  is a state Terraform can own.

Some objects also cannot be deleted through the API at all — networks and
registered IP addresses among them. Those resources warn on destroy and explain
what remains, rather than failing or pretending to have removed something.

### Known gaps that remain unbuilt

A handful of API surfaces are still not modelled, deliberately, because this
project found no confirmed request/response field names for them and refuses
to guess wire shapes — see [`docs/discovery/08-tenant-validation-and-gaps.md`](docs/discovery/08-tenant-validation-and-gaps.md)
for the full, evidence-graded list. The current set:

- **WAS Catalog, search lists and parameter sets.** The endpoints and CRUD
  verbs are documented (`/qps/rest/3.0/.../was/catalog`, `.../searchlist`,
  `.../parameter`), but no source found by this project names their request
  or response field names beyond "same CRUD pattern" — not enough to build
  against without fabricating a schema.
- **WAS finding severity override**, raw HTTP finding fields (`payload`,
  `request`, `response`), `malwareScheduling`'s recurrence structure, and
  tag-based multi-app WAS scan targeting: each confirmed to exist, none with
  a confirmed payload shape.
- **Domain V2 write operations** (`add`/`update`/`delete`): `data.qualys_domains`
  covers the confirmed read side; no confirmed create/update/delete parameter
  names exist for this project to build against.
- **`time_zone_code_list.php` output fields**: the endpoint's existence is
  confirmed, its response schema is not.
- **Legacy VM/PA user creation, Business Unit management, and distribution
  groups.** Read access to legacy users is built (`data.qualys_users`,
  Confirmed against a primary-source excerpt); a modern, tag/role-scoped
  alternative user API is also built (`data.qualys_am_users` and
  `qualys_user_scope_assignment`, Corroborated against the open-source
  [qualysdk](https://github.com/0x41424142/qualysdk) project — see below).
  Legacy user creation/edit (`user.php` `add`/`edit`) and all Business Unit
  management (`business_unit.php`) remain completely unresearched — three
  separate research passes (official docs, web search summaries, and this
  same third-party SDK, which has no Business Unit module at all) found
  nothing. `add_user.htm`/`edit_user.htm` do not appear to exist as pages at
  all (confirmed absent via archive.org, not just unreachable).

  What *is* now built points at a different answer to "customer only sees
  their own assets" than Business Units: the Administration RBAC API scopes
  a user entirely through tags and roles, with no Business Unit field at
  all, which converges with the AM/CSAM tagging model this provider already
  uses everywhere else (`qualys_asset_tag`, `qualys_host_tag_assignment`).
  Whether that's the actual replacement for the legacy Business Unit model,
  or a parallel and unrelated permission system, is not confirmed either —
  see doc 08's ninth research pass.

  See [`docs/discovery/08-tenant-validation-and-gaps.md`](docs/discovery/08-tenant-validation-and-gaps.md)
  for the full evidence trail and what's needed to close this next.

Full documentation is under [`docs/`](docs/), with runnable configuration in
[`examples/`](examples/).

## Getting started

```shell
export QUALYS_URL=https://qualysapi.qualys.com   # your platform's API server
export QUALYS_USERNAME=<your qualys login>
export QUALYS_PASSWORD=<your qualys password>
```

```hcl
provider "qualys" {
  # Recommended: supply credentials via the environment variables above rather
  # than in configuration.
}

resource "qualys_asset_group" "prod_web" {
  title      = "prod-web"
  network_id = var.qualys_network_id
  ips        = ["10.20.0.0/24"]
}
```

### `base_url` must be set explicitly

Qualys hosts subscriptions on several platforms, each with a different API
hostname. The provider will not guess one: pointing at the wrong platform would
send your credentials to it. Find yours under **Help > About** in the Qualys UI,
or at <https://www.qualys.com/platform-identification/>.

### If a brand-new service account fails everything

An API-only account — one with no UI access — stays inactive until the Qualys
EULA has been accepted, and every API call fails until then. The provider
detects this and reports it explicitly rather than as a generic authorisation
error.

## Design notes

A few behaviours are deliberate and worth knowing before you read the code:

- **Two API families, two clients.** The VM/PA XML API (`vmdr/`) and the portal
  APIs (`qps/`, covering Asset Management & Tagging and WAS) share only
  credentials. They differ in transport, envelope, encoding, pagination and
  error model, so they are separate clients rather than one with mode switches.
- **XML parsing never fetches DTDs.** Qualys responses reference a DTD by URL;
  those are treated as schema documentation, never retrieved. Entity expansion
  is disabled too, which closes billion-laughs — that needs no network access,
  so refusing remote fetches alone would not stop it.
- **Destructive operations are never retried automatically.** A blocked or timed
  out response does not prove the operation did not take effect, and repeating a
  purge or delete on that assumption is not safe.
- **A qps `OBJECT_NOT_FOUND` on Read never silently drops a resource from
  state.** The portal APIs return this same code both when an object was
  genuinely deleted and when it still exists but has fallen outside the
  caller's scope (a role, scope-tag or business unit change) — the response
  gives no way to tell the two apart. Clearing the id in the ambiguous case
  is exactly what causes Terraform to plan an unattended recreate of an
  object that may still exist, so every qps-backed resource's Read instead
  leaves state untouched and returns an error explaining why, with the
  `terraform state rm` command to run once you've confirmed the object is
  actually gone (`provider/notfound.go`). `Delete` is the mirror case and
  keeps the opposite, correct behaviour: an `OBJECT_NOT_FOUND` there means
  the desired end state is already achieved, so it succeeds rather than
  errors, keeping `terraform destroy` idempotent. VM/PA (`vmdr/`) resources
  are unaffected by this policy — that client classifies not-found via a
  narrow, deliberately conservative text match rather than a single
  ambiguous code (see `vmdr/errors.go`'s `ErrNotFound` doc comment), so the
  same ambiguity does not arise there.
- **Member lists are authoritative.** Removing an entry from configuration
  removes it in Qualys; the provider uses the API's `set_*`/`set` parameters
  rather than the additive ones, except where an object's own API confirms an
  add/remove idiom instead (e.g. WAS auth-record ↔ web-application association).
- **API versions are per capability.** Qualys versions each API independently
  and deprecates on a published schedule. The provider pins a version per
  capability and logs a one-time warning when a newer generation exists.
- **Findings data sources never aggregate.** `qualys_vm_findings` and
  `qualys_was_findings` each return one row per individual finding — never
  grouped by host, QID or web application — with a deterministic identity and
  ordering, since downstream automation typically serialises the result into a
  canonical dataset that must stay stable between reads of unchanged data.
- **Evidence-graded documentation.** Where a field or wire shape comes from an
  official, directly verified source it is documented as such; where it comes
  from a derived or unofficial source it is marked `Corroborated`, with the
  reasoning kept in [`docs/discovery/`](docs/discovery/) rather than presented
  as more certain than it is.

## Connector v3 migration

`qualys_gcp_connector`, `qualys_aws_connector` and `qualys_azure_connector`
now use the Connector v3 APIs (`/qps/rest/3.0/.../am/*assetdataconnector`),
replacing the CloudView v1 API that Qualys has deprecated.

Configurations are compatible: the credential attributes are unchanged (the
GCP key file, the AWS `arn`/`external_id` pair, the four Azure
service-principal fields), and the resources gain optional `run_frequency`,
`disabled`, `is_remediation_enabled`, `activation` and `default_tag_ids`
attributes. GCP's `project_id` is now derived from the key file and may be
left unset.

**Breaking:** `qualys_aws_connector.is_china_region` and
`qualys_azure_connector.is_gov_cloud` are now read-only. Both were
`Optional` under the old CloudView v1 client but never had a working write
path under Connector v3 — any value set for them was silently discarded on
every apply and immediately overwritten by the next refresh, producing a
permanent diff for anyone who configured them explicitly. Remove them from
configuration; Terraform reports their real value without you setting it.
AWS's `is_gov_cloud` is unaffected — it is a genuine, working write field
under Connector v3, unlike Azure's identically-named attribute.

State migrates itself: v1 state holds a connector UUID where v3 uses numeric
ids, and on the first refresh the provider finds the connector whose
`cloudviewUuid` matches and adopts the numeric id. If that lookup fails
(for example, the tenant's v3 listing does not include the connector),
fall back to re-importing:

```shell
terraform state rm qualys_gcp_connector.example
terraform import qualys_gcp_connector.example <numeric-connector-id>
```

The v3 wire behaviour is implemented from the official API documentation
(evidence in [`docs/discovery/12-connector-v3-migration.md`](docs/discovery/12-connector-v3-migration.md)),
not yet validated against a live tenant — the docs are internally
inconsistent in places (HTTP verbs for some operations, response collection
nesting), and the code takes the documented-safest reading of each. Reports
from real tenants, confirming or correcting, are very welcome as issues
or pull requests.

## Development

```shell
make build    # fmt check, vet, install
make test     # unit tests
make docs     # regenerate docs/ with tfplugindocs
```

`make docs` requires network access to download the Terraform CLI. The files
under `docs/` are kept in sync by hand when that is unavailable; regenerate them
whenever you can, and prefer the generated output on any conflict.

CI (`.github/workflows/ci.yaml`) additionally runs `staticcheck`,
`govulncheck` and `go test -race` on every push and pull request, reading
the Go version from `go.mod`'s own `go`/`toolchain` directives rather than
a hardcoded value in the workflow.

### Acceptance tests

`provider/resource_gcp_connector_acc_test.go` demonstrates a real
Terraform-protocol acceptance test (`TF_ACC=1`, via the SDK's
`helper/resource` package) against a local mock server — covering create,
a credential-rotation update, a no-op plan and import for the GCP
connector. It needs a `terraform` binary on `PATH` (or
`TF_ACC_TERRAFORM_PATH` set); it does not run by default, so it never
affects a plain `go test ./...`. This is the first resource covered, not
the last — `qualys_asset_tag`, `qualys_vm_option_profile`,
`qualys_vm_scan_schedule`, `qualys_was_application`,
`qualys_was_auth_record`, a WAS scheduling resource and the AWS/Azure
connectors are natural next candidates, following the same
mock-server-backed pattern.

## References

- Qualys API (VM and PA) documentation — <https://docs.qualys.com/en/vm/qweb-all-api/>
- Qualys WAS API documentation — <https://docs.qualys.com/en/was/api/>
- API discovery notes for this provider — [`docs/discovery/`](docs/discovery/)
- See [`LICENSE`](LICENSE) for this project's licensing and provenance.
