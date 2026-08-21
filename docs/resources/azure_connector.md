---
page_title: "qualys_azure_connector Resource - terraform-provider-qualys"
subcategory: ""
description: |-
  A Qualys connector used for scanning Azure subscription assets, via the Connector v3 API
---

# qualys_azure_connector (Resource)

A Qualys connector used for scanning Azure subscription assets, via the
Connector v3 API (`/qps/rest/3.0/.../am/azureassetdataconnector`).

~> This resource was migrated from the deprecated CloudView v1 API. State
created by pre-migration provider versions holds a UUID id; on the first
refresh the provider looks the connector up by that UUID (the v3 API reports
it as `cloudviewUuid`) and adopts the numeric v3 id automatically. If the
lookup fails, remove the resource from state and re-import it by its numeric
id. This migration path is validated against the official API documentation
but not yet against a live tenant — corrections from real tenants are
welcome as issues or PRs.

## Example Usage

```terraform
resource "qualys_azure_connector" "dev" {
  name        = "dev_azure"
  description = "Development subscription connector"

  application_id     = "00000000-0000-0000-0000-000000000000"
  directory_id       = "11111111-1111-1111-1111-111111111111"
  subscription_id    = "22222222-2222-2222-2222-222222222222"
  authentication_key = var.azure_client_secret

  run_frequency = 240
  activation    = ["VM", "CLOUDVIEW"]
}
```

## Schema

### Required

- **application_id** (String) The unique ID of the Azure AD application
  registration used to authenticate
- **directory_id** (String) The unique ID of the Azure Active Directory
  (tenant ID)
- **subscription_id** (String) The unique ID of the Azure subscription to
  scan
- **authentication_key** (String, Sensitive) The client secret for the
  Azure AD application. The API never returns it, so this value is
  write-only and drift is not detected.
- **name** (String) Name of the connector

### Optional

- **description** (String) A string describing this connector instance
- **run_frequency** (Number) How often the connector polls the cloud
  account, in minutes. Defaults to 240.
- **disabled** (Boolean) Disables connector execution without deleting it
- **is_remediation_enabled** (Boolean) Enables Qualys remediation for
  assets discovered by this connector
- **activation** (Set of String) Qualys modules activated for discovered
  assets: `VM`, `CERTVIEW`, `CLOUDVIEW`, `SCA`, `CSA`
- **default_tag_ids** (Set of String) Asset tag ids applied to every asset
  this connector discovers

### Read-Only

- **connector_id** (String) The connector's numeric Connector v3 id
- **is_gov_cloud** (Boolean) Whether this connector targets an Azure
  Government Cloud subscription. The Connector v3 API only reports this
  (unlike AWS's `is_gov_cloud`, which is a real write field); Azure has no
  field to set it.

  ~> **Breaking change:** this was previously `Optional`. The write path
  never actually worked — the value was silently discarded on every
  create/update and immediately overwritten by the next Read, producing a
  permanent, unresolvable diff for any configuration that set it
  explicitly. Remove `is_gov_cloud` from configuration; Terraform will now
  report it correctly without you setting it.
- **subscription_name** (String) The name of the Azure subscription
  associated with this connector
- **connector_state** (String) Lifecycle state (QUEUED, RUNNING,
  FINISHED_SUCCESS, FINISHED_ERRORS, ...)
- **last_sync** (String) When the connector last synchronised, ISO-8601 UTC
- **next_sync** (String) When the connector next synchronises, ISO-8601 UTC
- **cloudview_uuid** (String) The deprecated CloudView v1 API's UUID for
  this connector; empty for connectors created directly under v3
- **cloud_provider** (String) Always `AZURE`

## Import

```shell
terraform import qualys_azure_connector.dev <numeric-connector-id>
```
