---
page_title: "qualys_aws_connector Resource - terraform-provider-qualys"
subcategory: ""
description: |-
  A Qualys connector used for scanning AWS account assets, via the Connector v3 API
---

# qualys_aws_connector (Resource)

A Qualys connector used for scanning AWS account assets, via the Connector v3
API (`/qps/rest/3.0/.../am/awsassetdataconnector`).

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
resource "qualys_aws_connector" "dev" {
  name        = "dev_aws"
  description = "Development account connector"
  arn         = "arn:aws:iam::123456789012:role/qualys-connector-role"
  external_id = "US1-123456-1234567890123"

  run_frequency = 240
  activation    = ["VM", "CLOUDVIEW"]
}
```

## Schema

### Required

- **arn** (String) The ARN of the IAM role Qualys assumes to scan this AWS
  account. Trust the account in `qualys_aws_account_id` in the role's policy,
  conditioned on `external_id`.
- **external_id** (String) The external ID configured in the IAM role's
  trust policy
- **name** (String) Name of the connector

### Optional

- **description** (String) A string describing this connector instance
- **all_regions** (Boolean) Whether the connector scans all AWS regions.
  Defaults to `true`. The Connector v3 API's per-region endpoint selection
  is not carried by this provider yet, so `false` currently produces a
  connector with no regions selected.
- **is_gov_cloud** (Boolean) Whether this connector targets an AWS GovCloud
  account
- **run_frequency** (Number) How often the connector polls the cloud
  account, in minutes. Defaults to 240.
- **disabled** (Boolean) Disables connector execution without deleting it
- **is_remediation_enabled** (Boolean) Enables Qualys remediation for
  assets discovered by this connector
- **activation** (Set of String) Qualys modules activated for discovered
  assets: `VM`, `CERTVIEW`, `CLOUDVIEW`, `SCA`, `CSA`
- **default_tag_ids** (Set of String) Asset tag ids applied to every asset
  this connector discovers
- **is_portal_connector** (Boolean, Deprecated) No effect; the Connector v3
  API always manages the portal (AssetView) connector itself

### Read-Only

- **connector_id** (String) The connector's numeric Connector v3 id
- **is_china_region** (Boolean) Whether this connector targets an AWS China
  region account. The Connector v3 API only reports this; it has no field
  to set it.

  ~> **Breaking change:** this was previously `Optional`. The write path
  never actually worked — the value was silently discarded on every
  create/update and immediately overwritten by the next Read, producing a
  permanent, unresolvable diff for any configuration that set it
  explicitly. Remove `is_china_region` from configuration; Terraform will
  now report it correctly without you setting it.
- **aws_account_id** (String) The AWS account ID associated with this
  connector
- **qualys_aws_account_id** (String) The Qualys-owned AWS account that
  assumes the role; the account to trust in the IAM role's policy
- **connector_state** (String) Lifecycle state (QUEUED, RUNNING,
  FINISHED_SUCCESS, FINISHED_ERRORS, ...)
- **last_sync** (String) When the connector last synchronised, ISO-8601 UTC
- **next_sync** (String) When the connector next synchronises, ISO-8601 UTC
- **cloudview_uuid** (String) The deprecated CloudView v1 API's UUID for
  this connector; empty for connectors created directly under v3
- **cloud_provider** (String) Always `AWS`

## Import

```shell
terraform import qualys_aws_connector.dev <numeric-connector-id>
```
