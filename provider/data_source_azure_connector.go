package provider

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
)

func dataSourceAzureConnector() *schema.Resource {
	s := connectorDataSourceSchema()
	s["application_id"] = &schema.Schema{
		Description: "The unique ID of the Azure AD application registration used to authenticate",
		Type:        schema.TypeString,
		Computed:    true,
	}
	s["directory_id"] = &schema.Schema{
		Description: "The unique ID of the Azure Active Directory (tenant ID)",
		Type:        schema.TypeString,
		Computed:    true,
	}
	s["subscription_id"] = &schema.Schema{
		Description: "The unique ID of the Azure subscription the connector scans",
		Type:        schema.TypeString,
		Computed:    true,
	}
	s["subscription_name"] = &schema.Schema{
		Description: "The name of the Azure subscription associated with this connector",
		Type:        schema.TypeString,
		Computed:    true,
	}
	s["is_gov_cloud"] = &schema.Schema{
		Description: "Whether this connector targets an Azure Government Cloud subscription",
		Type:        schema.TypeBool,
		Computed:    true,
	}

	return &schema.Resource{
		Description: "Looks up a Connector v3 Azure connector by numeric id or by name",
		ReadContext: dataSourceAzureConnectorRead,
		Schema:      s,
	}
}

func dataSourceAzureConnectorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	var conn *qps.AzureConnector
	if id := d.Get("connector_id").(string); id != "" {
		log.Printf("[DEBUG] read azure connector %q", id)
		conn, err = client.GetAzureConnector(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		name := d.Get("name").(string)
		log.Printf("[DEBUG] search azure connector named %q", name)
		conns, err := client.SearchAzureConnectors(ctx, nameFilter(name))
		if err != nil {
			return diag.FromErr(err)
		}
		if len(conns) == 0 {
			return diag.Errorf("no azure connector named %q", name)
		}
		if len(conns) > 1 {
			return diag.Errorf("%d azure connectors named %q; look the connector up "+
				"by connector_id instead", len(conns), name)
		}
		conn = conns[0]
	}

	d.SetId(conn.ID)
	if err := setConnectorBaseData(d, conn.ConnectorBase); err != nil {
		return diag.FromErr(err)
	}
	return combineErrors(
		d.Set("application_id", conn.ApplicationID),
		d.Set("directory_id", conn.DirectoryID),
		d.Set("subscription_id", conn.SubscriptionID),
		d.Set("subscription_name", conn.SubscriptionName),
		d.Set("is_gov_cloud", conn.IsGovCloud),
		d.Set("cloud_provider", "AZURE"),
	)
}
