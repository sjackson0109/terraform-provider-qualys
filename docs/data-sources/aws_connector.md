---
page_title: "qualys_aws_connector Data Source - terraform-provider-qualys"
subcategory: ""
description: |-
  Looks up a Connector v3 AWS connector by numeric id or by name
---

# qualys_aws_connector (Data Source)

Looks up a Connector v3 AWS connector by numeric id or by name. Exactly one
of `connector_id` and `name` must be set; a name lookup fails if it matches
more than one connector.

## Example Usage

```terraform
data "qualys_aws_connector" "dev" {
  name = "dev_aws"
}

output "qualys_trust_account" {
  value = data.qualys_aws_connector.dev.qualys_aws_account_id
}
```

## Schema

### Optional

- **connector_id** (String) The connector's numeric Connector v3 id
- **name** (String) Name of the connector

### Read-Only

- **description** (String) A string describing this connector instance
- **arn** (String) The ARN of the IAM role Qualys assumes
- **external_id** (String) The external ID in the role's trust policy
- **all_regions** (Boolean) Whether all AWS regions are scanned
- **is_gov_cloud** (Boolean) Whether the account is AWS GovCloud
- **is_china_region** (Boolean) Whether the account is an AWS China region
- **aws_account_id** (String) The scanned AWS account ID
- **qualys_aws_account_id** (String) The Qualys-owned AWS account that
  assumes the role
- **run_frequency** (Number) Polling interval in minutes
- **disabled** (Boolean) Whether connector execution is disabled
- **is_remediation_enabled** (Boolean) Whether Qualys remediation is enabled
- **activation** (Set of String) Activated Qualys modules
- **default_tag_ids** (Set of String) Tag ids applied to discovered assets
- **connector_state** (String) Lifecycle state
- **last_sync** (String) Last synchronisation, ISO-8601 UTC
- **next_sync** (String) Next synchronisation, ISO-8601 UTC
- **cloudview_uuid** (String) The deprecated CloudView v1 UUID, if any
- **cloud_provider** (String) Always `AWS`
