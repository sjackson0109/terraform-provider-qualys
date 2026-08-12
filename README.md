# terraform-provider-qualys

A Terraform provider for [Qualys](https://www.qualys.com/) VMDR and WAS
(Web Application Scanning) configuration.

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
| `qualys_scan_schedule` | A recurring VM scan |
| `qualys_virtual_scanner` | A virtual scanner appliance, exporting the activation code for VM deployment |
| `qualys_auth_record_windows` | A Windows scan authentication record |
| `qualys_auth_record_unix` | A Unix/Linux (SSH) scan authentication record |
| `qualys_auth_record_*` (28 more) | Every other Confirmed scan authentication record type — databases, web/app servers, network devices, hypervisors, containers; see [`docs/resources/auth_record_other_types.md`](docs/resources/auth_record_other_types.md) |
| `qualys_vault` | A credential vault; `title`/`type` are managed, per-type connection parameters pass through verbatim |
| `qualys_excluded_ips` | Excludes IPs from scanning |
| `qualys_host_tag_assignment` | Assigns asset tags to an AssetView/CSAM host asset |
| `qualys_gcp_connector` | CloudView GCP connector (see *Deprecation* below) |

WAS (Web Application Scanning):

| Resource | Purpose |
|---|---|
| `qualys_web_application` | The scan target that option profiles, auth records and scan schedules reference |
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
| `qualys_scan_schedules` | Look up VM scan schedules |
| `qualys_option_profiles` | Look up VM scan option profiles |
| `qualys_report_templates` | Look up VM/PA report templates |
| `qualys_domains` | Look up asset domains and their netblocks |
| `qualys_web_applications` | Look up WAS web applications |
| `qualys_was_findings` | Look up individual WAS scan findings, optionally enriched from the Qualys KnowledgeBase |
| `qualys_was_report_templates` | Look up WAS report templates |
| `qualys_gcp_connector` | Look up a CloudView GCP connector by ID |
| `qualys_tagged_assets` | Look up AssetView/CSAM host assets by tag |
| `qualys_auth_records` | Look up scan authentication records of one type |
| `qualys_scans` | Look up VM scan jobs |
| `qualys_reports` | Look up generated VM/PA reports |
| `qualys_report_schedules` | Look up scheduled report definitions (list-only: there is no report schedule write API) |
| `qualys_tenant_capabilities` | Probe the subscription's installed platform and per-module versions |

### Operations the provider deliberately does not model

Some Qualys operations are jobs rather than configuration, and modelling them as
resources would misrepresent them:

- **Launching a scan or report** produces a run with no stable identity;
  re-running creates a new one rather than updating the old. Use
  `qualys_scan_schedule` / `qualys_was_scan_schedule` for recurring scans, and
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
- **VM/PA users and distribution groups**: `/msp/user_list.php` read access
  exists, but write-side user management uses a legacy, differently-authenticated
  V1 API this project has deliberately deferred rather than build against
  thin evidence (tracked as P3 in [`docs/discovery/06-terraform-mapping.md`](docs/discovery/06-terraform-mapping.md)).

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

## Deprecation

`qualys_gcp_connector` currently uses the CloudView connector API, which Qualys
has announced as deprecated in favour of the Connector v3 APIs. New GCP
connectors can already only be created from the Connectors application. This
resource is scheduled to move to Connector v3 with a state migration.

## Development

```shell
make build    # fmt check, vet, install
make test     # unit tests
make docs     # regenerate docs/ with tfplugindocs
```

`make docs` requires network access to download the Terraform CLI. The files
under `docs/` are kept in sync by hand when that is unavailable; regenerate them
whenever you can, and prefer the generated output on any conflict.

## References

- Qualys API (VM and PA) documentation — <https://docs.qualys.com/en/vm/qweb-all-api/>
- Qualys WAS API documentation — <https://docs.qualys.com/en/was/api/>
- API discovery notes for this provider — [`docs/discovery/`](docs/discovery/)
- See [`LICENSE`](LICENSE) for this project's licensing and provenance.
