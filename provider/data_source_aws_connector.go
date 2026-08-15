package provider

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/form3tech-oss/terraform-provider-qualys/cloudview/aws"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAWSConnector() *schema.Resource {
	return &schema.Resource{
		Description: "Returns the details of the connector instances defined by the `connector_id`",
		ReadContext: dataSourceAWSConnectorRead,

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
			"arn": {
				Description: "The ARN of the IAM role Qualys assumes to scan this AWS account",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"external_id": {
				Description: "The external ID configured in the IAM role's trust policy",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"is_portal_connector": {
				Description: "Whether an AssetView connector is also created alongside this CloudView connector",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"is_gov_cloud": {
				Description: "Whether this connector targets an AWS GovCloud account",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"is_china_region": {
				Description: "Whether this connector targets an AWS China region account",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"is_disabled": {
				Description: "Whether this connector is disabled",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"aws_account_id": {
				Description: "The AWS account ID associated with this connector",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"account_alias": {
				Description: "The AWS account alias associated with this connector",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"base_account_id": {
				Description: "The AWS base account ID associated with this connector",
				Type:        schema.TypeString,
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

func dataSourceAWSConnectorRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Reading aws connector %q ", d.Get("connector_id"))

	service := awsService(meta)
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
		d.Set("arn", connector.ARN),
		d.Set("external_id", connector.ExternalID),
		d.Set("is_portal_connector", connector.IsPortalConnector),
		d.Set("is_gov_cloud", connector.IsGovCloud),
		d.Set("is_china_region", connector.IsChinaRegion),
		d.Set("is_disabled", connector.IsDisabled),
		d.Set("aws_account_id", connector.AWSAccountID),
		d.Set("account_alias", connector.AccountAlias),
		d.Set("base_account_id", connector.BaseAccountID),
		d.Set("tags", flattenAWSTags(connector.Tags)),
		d.Set("last_synced_on", connector.LastSyncedOn),
		d.Set("total_assets", connector.TotalAssets),
		d.Set("state", connector.State),
	)
}

func flattenAWSTags(tags []aws.Tag) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(tags))
	for _, tag := range tags {
		out = append(out, map[string]interface{}{
			"name": tag.Name,
			"uuid": tag.UUID,
		})
	}
	return out
}
