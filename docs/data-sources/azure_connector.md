---
page_title: "qualys_azure_connector Data Source - terraform-provider-qualys"
subcategory: ""
description: |-
  Looks up a Connector v3 Azure connector by numeric id or by name
---

# qualys_azure_connector (Data Source)

Looks up a Connector v3 Azure connector by numeric id or by name. Exactly
one of `connector_id` and `name` must be set; a name lookup fails if it
matches more than one connector.

## Example Usage

```terraform
data "qualys_azure_connector" "dev" {
  name = "dev_azure"
}

output "dev_subscription" {
  value = data.qualys_azure_connector.dev.subscription_name
}
```

## Schema

### Optional

- **connector_id** (String) The connector's numeric Connector v3 id
- **name** (String) Name of the connector

### Read-Only

- **description** (String) A string describing this connector instance
- **application_id** (String) The Azure AD application registration ID
- **directory_id** (String) The Azure AD tenant ID
- **subscription_id** (String) The scanned Azure subscription ID
- **subscription_name** (String) The scanned Azure subscription's name
- **is_gov_cloud** (Boolean) Whether the subscription is Azure Government
- **run_frequency** (Number) Polling interval in minutes
- **disabled** (Boolean) Whether connector execution is disabled
- **is_remediation_enabled** (Boolean) Whether Qualys remediation is enabled
- **activation** (Set of String) Activated Qualys modules
- **default_tag_ids** (Set of String) Tag ids applied to discovered assets
- **connector_state** (String) Lifecycle state
- **last_sync** (String) Last synchronisation, ISO-8601 UTC
- **next_sync** (String) Next synchronisation, ISO-8601 UTC
- **cloudview_uuid** (String) The deprecated CloudView v1 UUID, if any
- **cloud_provider** (String) Always `AZURE`
