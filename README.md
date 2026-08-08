# terraform-provider-qualys

A Terraform provider for [Qualys](https://www.qualys.com/) VMDR configuration.

## Supported resources

| Resource | Purpose |
|---|---|
| `qualys_asset_group` | Named sets of scan targets, referenced by scans, schedules and reports |
| `qualys_asset_tag` | Static and dynamic asset tags, shared across VM, AssetView and WAS |
| `qualys_static_search_list` | Explicit sets of vulnerability QIDs |
| `qualys_vm_option_profile` | VM scan behaviour: ports, detection depth, performance, authentication |
| `qualys_network` | Qualys networks, for overlapping IP space |
| `qualys_ip_registration` | Registers addresses with the subscription and enables them for scanning |
| `qualys_scan_schedule` | Recurring VM scans |
| `qualys_gcp_connector` | CloudView GCP connector (see *Deprecation* below) |

Data sources: `qualys_asset_groups`, `qualys_asset_tags`, `qualys_networks`,
`qualys_host_assets`, `qualys_gcp_connector`.

### Operations the provider deliberately does not model

Some Qualys operations are jobs rather than configuration, and modelling them as
resources would misrepresent them:

- **Launching a scan or report** produces a run with no stable identity;
  re-running creates a new one rather than updating the old. Use
  `qualys_scan_schedule` for recurring scans.
- **Purging host data** is destructive and irreversible. It is available as an
  opt-in on `qualys_ip_registration` destroy, not as a resource of its own.

Some objects also cannot be deleted through the API at all — networks and
registered IP addresses among them. Those resources warn on destroy and explain
what remains, rather than failing or pretending to have removed something.

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
  removes it in Qualys; the provider uses the API's `set_*` parameters rather
  than the additive ones.
- **API versions are per capability.** Qualys versions each API independently
  and deprecates on a published schedule. The provider pins a version per
  capability and logs a one-time warning when a newer generation exists.

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
- API discovery notes for this provider — [`docs/discovery/`](docs/discovery/)
- Based on original work from <https://code.stanford.edu/xuwang/terraform-provider-qualys>
