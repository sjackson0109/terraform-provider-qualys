package provider

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/sjackson0109/terraform-provider-qualys/cloudview/azure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAzureConnector() *schema.Resource {
	return &schema.Resource{
		Description: "Returns the details of the connector instances defined by the `connector_id`",
		ReadContext: dataSourceAzureConnectorRead,

		Schema: map[string]*schema.Schema{
			"connector_id": {
				Description: "The unique ID for this connector instance",
				Type:        schema.TypeString,
				Required:    true,
			},
			"cloud_provider": {
				Description: "The cloud provider associated with this connector",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"name": {
				Description: "Name of the connector",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"description": {
				Description: "A string describing this connector instance",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"application_id": {
				Description: "The unique ID of the Azure AD application registration used to authenticate",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"directory_id": {
				Description: "The unique ID of the Azure Active Directory (tenant ID)",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"subscription_id": {
				Description: "The unique ID of the Azure subscription associated with this connector",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"subscription_name": {
				Description: "The name of the Azure subscription associated with this connector",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"is_gov_cloud": {
				Description: "Whether this connector targets an Azure Government Cloud subscription",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"is_disabled": {
				Description: "Whether this connector is disabled",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"tags": {
				Description: "Tags this connector belongs to",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Name of tag",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"uuid": {
							Description: "ID of tag",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
			"last_synced_on": {
				Description: "Last sync timestamp",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"total_assets": {
				Description: "Total assets associated with this connector",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"state": {
				Description: "State of the connector",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceAzureConnectorRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Reading azure connector %q ", d.Get("connector_id"))

	service := azureService(meta)
	connector, err := service.Get(d.Get("connector_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(connector.ConnectorID)

	return combineErrors(
		d.Set("cloud_provider", connector.Provider),
		d.Set("connector_id", connector.ConnectorID),
		d.Set("name", connector.Name),
		d.Set("description", connector.Description),
		d.Set("application_id", connector.ApplicationID),
		d.Set("directory_id", connector.DirectoryID),
		d.Set("subscription_id", connector.SubscriptionID),
		d.Set("subscription_name", connector.SubscriptionName),
		d.Set("is_gov_cloud", connector.IsGovCloud),
		d.Set("is_disabled", connector.IsDisabled),
		d.Set("tags", flattenAzureTags(connector.Tags)),
		d.Set("last_synced_on", connector.LastSyncedOn),
		d.Set("total_assets", connector.TotalAssets),
		d.Set("state", connector.State),
	)
}

func flattenAzureTags(tags []azure.Tag) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(tags))
	for _, tag := range tags {
		out = append(out, map[string]interface{}{
			"name": tag.Name,
			"uuid": tag.UUID,
		})
	}
	return out
}
