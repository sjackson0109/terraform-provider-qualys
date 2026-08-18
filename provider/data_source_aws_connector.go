package provider

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
)

func dataSourceAWSConnector() *schema.Resource {
	s := connectorDataSourceSchema()
	s["arn"] = &schema.Schema{
		Description: "The ARN of the IAM role Qualys assumes to scan this AWS account",
		Type:        schema.TypeString,
		Computed:    true,
	}
	s["external_id"] = &schema.Schema{
		Description: "The external ID configured in the IAM role's trust policy",
		Type:        schema.TypeString,
		Computed:    true,
	}
	s["all_regions"] = &schema.Schema{
		Description: "Whether the connector scans all AWS regions",
		Type:        schema.TypeBool,
		Computed:    true,
	}
	s["is_gov_cloud"] = &schema.Schema{
		Description: "Whether this connector targets an AWS GovCloud account",
		Type:        schema.TypeBool,
		Computed:    true,
	}
	s["is_china_region"] = &schema.Schema{
		Description: "Whether this connector targets an AWS China region account",
		Type:        schema.TypeBool,
		Computed:    true,
	}
	s["aws_account_id"] = &schema.Schema{
		Description: "The AWS account ID associated with this connector",
		Type:        schema.TypeString,
		Computed:    true,
	}
	s["qualys_aws_account_id"] = &schema.Schema{
		Description: "The Qualys-owned AWS account that assumes the role",
		Type:        schema.TypeString,
		Computed:    true,
	}

	return &schema.Resource{
		Description: "Looks up a Connector v3 AWS connector by numeric id or by name",
		ReadContext: dataSourceAWSConnectorRead,
		Schema:      s,
	}
}

func dataSourceAWSConnectorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	var conn *qps.AWSConnector
	if id := d.Get("connector_id").(string); id != "" {
		log.Printf("[DEBUG] read aws connector %q", id)
		conn, err = client.GetAWSConnector(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		name := d.Get("name").(string)
		log.Printf("[DEBUG] search aws connector named %q", name)
		conns, err := client.SearchAWSConnectors(ctx, nameFilter(name))
		if err != nil {
			return diag.FromErr(err)
		}
		if len(conns) == 0 {
			return diag.Errorf("no aws connector named %q", name)
		}
		if len(conns) > 1 {
			return diag.Errorf("%d aws connectors named %q; look the connector up "+
				"by connector_id instead", len(conns), name)
		}
		conn = conns[0]
	}

	d.SetId(conn.ID)
	if err := setConnectorBaseData(d, conn.ConnectorBase); err != nil {
		return diag.FromErr(err)
	}
	return combineErrors(
		d.Set("arn", conn.ARN),
		d.Set("external_id", conn.ExternalID),
		d.Set("all_regions", conn.AllRegions),
		d.Set("is_gov_cloud", conn.IsGovCloud),
		d.Set("is_china_region", conn.IsChinaConfigured),
		d.Set("aws_account_id", conn.AWSAccountID),
		d.Set("qualys_aws_account_id", conn.QualysAWSAccountID),
		d.Set("cloud_provider", "AWS"),
	)
}
