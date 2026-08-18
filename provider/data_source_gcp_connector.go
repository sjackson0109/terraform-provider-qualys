package provider

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
)

func dataSourceGCPConnector() *schema.Resource {
	s := connectorDataSourceSchema()
	s["project_id"] = &schema.Schema{
		Description: "GCP project id the connector scans",
		Type:        schema.TypeString,
		Computed:    true,
	}

	return &schema.Resource{
		Description: "Looks up a Connector v3 GCP connector by numeric id or by name",
		ReadContext: dataSourceGCPConnectorRead,
		Schema:      s,
	}
}

func dataSourceGCPConnectorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	var conn *qps.GCPConnector
	if id := d.Get("connector_id").(string); id != "" {
		log.Printf("[DEBUG] read gcp connector %q", id)
		conn, err = client.GetGCPConnector(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		name := d.Get("name").(string)
		log.Printf("[DEBUG] search gcp connector named %q", name)
		conns, err := client.SearchGCPConnectors(ctx, nameFilter(name))
		if err != nil {
			return diag.FromErr(err)
		}
		if len(conns) == 0 {
			return diag.Errorf("no gcp connector named %q", name)
		}
		if len(conns) > 1 {
			return diag.Errorf("%d gcp connectors named %q; look the connector up "+
				"by connector_id instead", len(conns), name)
		}
		conn = conns[0]
	}

	d.SetId(conn.ID)
	if err := setConnectorBaseData(d, conn.ConnectorBase); err != nil {
		return diag.FromErr(err)
	}
	return combineErrors(
		d.Set("project_id", conn.ProjectID),
		d.Set("cloud_provider", "GCP"),
	)
}
