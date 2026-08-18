package provider

import (
	"context"
	"errors"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
)

func resourceAzureConnector() *schema.Resource {
	s := connectorCommonSchema()
	s["application_id"] = &schema.Schema{
		Description: "The unique ID of the Azure AD application registration used to authenticate",
		Type:        schema.TypeString,
		Required:    true,
	}
	s["directory_id"] = &schema.Schema{
		Description: "The unique ID of the Azure Active Directory (tenant ID)",
		Type:        schema.TypeString,
		Required:    true,
	}
	s["subscription_id"] = &schema.Schema{
		Description: "The unique ID of the Azure subscription to scan",
		Type:        schema.TypeString,
		Required:    true,
	}
	s["authentication_key"] = &schema.Schema{
		Description: "The client secret (authentication key) for the Azure AD " +
			"application. The API never returns it, so this value is write-only " +
			"and drift is not detected.",
		Type:      schema.TypeString,
		Sensitive: true,
		Required:  true,
	}
	s["is_gov_cloud"] = &schema.Schema{
		Description: "Whether this connector targets an Azure Government Cloud " +
			"subscription. Read-only under the Connector v3 API.",
		Type:       schema.TypeBool,
		Optional:   true,
		Computed:   true,
		Deprecated: "The Connector v3 API has no Azure gov-cloud write field; the value is read-only.",
	}
	s["subscription_name"] = &schema.Schema{
		Description: "The name of the Azure subscription associated with this connector",
		Type:        schema.TypeString,
		Computed:    true,
	}

	return &schema.Resource{
		Description: "A Qualys connector used for scanning Azure subscription assets, " +
			"via the Connector v3 API",
		CreateContext: resourceAzureConnectorCreate,
		ReadContext:   resourceAzureConnectorRead,
		UpdateContext: resourceAzureConnectorUpdate,
		DeleteContext: resourceAzureConnectorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: s,
	}
}

func azureConnectorInput(d *schema.ResourceData) qps.AzureConnectorInput {
	return qps.AzureConnectorInput{
		ConnectorBaseInput: connectorBaseInput(d),
		ApplicationID:      d.Get("application_id").(string),
		DirectoryID:        d.Get("directory_id").(string),
		SubscriptionID:     d.Get("subscription_id").(string),
		AuthenticationKey:  d.Get("authentication_key").(string),
	}
}

func resourceAzureConnectorCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] create azure connector %q", d.Get("name").(string))

	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	conn, err := client.CreateAzureConnector(ctx, azureConnectorInput(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(conn.ID)
	return resourceAzureConnectorRead(ctx, d, meta)
}

func resourceAzureConnectorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] read azure connector %q", d.Id())

	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	// State written by provider versions that used the deprecated CloudView
	// v1 API holds the connector's UUID. The v3 API reports that UUID as
	// cloudviewUuid, so a one-time lookup adopts the numeric v3 id in place.
	if isCloudViewUUID(d.Id()) {
		conn, err := client.FindAzureConnectorByCloudViewUUID(ctx, d.Id())
		if err != nil {
			if errors.Is(err, qps.ErrNotFound) {
				return diag.Errorf("azure connector: state id %s is a CloudView v1 UUID and "+
					"no Connector v3 connector reports it as cloudviewUuid; remove it from "+
					"state and re-import by its numeric v3 id "+
					"(terraform state rm, then terraform import)", d.Id())
			}
			return diag.FromErr(err)
		}
		log.Printf("[INFO] azure connector: adopted Connector v3 id %s for CloudView UUID %s",
			conn.ID, d.Id())
		d.SetId(conn.ID)
	}

	conn, err := client.GetAzureConnector(ctx, d.Id())
	if err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

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

func resourceAzureConnectorUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] update azure connector %s", d.Id())

	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	// The v3 update responds with the id only; Read refreshes the rest.
	if err := client.UpdateAzureConnector(ctx, d.Id(), azureConnectorInput(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceAzureConnectorRead(ctx, d, meta)
}

func resourceAzureConnectorDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] delete azure connector %s", d.Id())

	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(client.DeleteAzureConnector(ctx, d.Id()))
}
