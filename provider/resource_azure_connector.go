package provider

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/sjackson0109/terraform-provider-qualys/cloudview/azure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAzureConnector() *schema.Resource {
	return &schema.Resource{
		Description:   "A Qualys connector used for scanning Azure subscription assets",
		CreateContext: resourceAzureConnectorCreate,
		ReadContext:   resourceAzureConnectorRead,
		UpdateContext: resourceAzureConnectorUpdate,
		DeleteContext: resourceAzureConnectorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"connector_id": {
				Description: "The unique ID for this connector instance",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"name": {
				Description: "Name of the connector",
				Type:        schema.TypeString,
				Required:    true,
			},
			"description": {
				Description: "A string describing this connector instance",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"application_id": {
				Description: "The unique ID of the Azure AD application registration used to authenticate",
				Type:        schema.TypeString,
				Required:    true,
			},
			"directory_id": {
				Description: "The unique ID of the Azure Active Directory (tenant ID)",
				Type:        schema.TypeString,
				Required:    true,
			},
			"subscription_id": {
				Description: "The unique ID of the Azure subscription to scan",
				Type:        schema.TypeString,
				Required:    true,
			},
			"authentication_key": {
				Description: "The client secret (authentication key) for the Azure AD application",
				Type:        schema.TypeString,
				Sensitive:   true,
				Required:    true,
			},
			"is_gov_cloud": {
				Description: "Whether this connector targets an Azure Government Cloud subscription",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"cloud_provider": {
				Description: "The cloud provider associated with this connector",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"subscription_name": {
				Description: "The name of the Azure subscription associated with this connector",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceAzureConnectorCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] create azure connector %q", d.Get("name").(string))

	opt := azure.NewConnectorConfig().
		WithName(d.Get("name").(string)).
		WithDescription(d.Get("description").(string)).
		WithApplicationID(d.Get("application_id").(string)).
		WithDirectoryID(d.Get("directory_id").(string)).
		WithSubscriptionID(d.Get("subscription_id").(string)).
		WithAuthenticationKey(d.Get("authentication_key").(string)).
		WithIsGovCloud(d.Get("is_gov_cloud").(bool))

	service := azureService(meta)
	connector, err := service.Create(opt)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(connector.ConnectorID)

	return resourceAzureConnectorRead(ctx, d, meta)
}

func resourceAzureConnectorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Reading azure connector %q ", d.Id())

	service := azureService(meta)
	connector, err := service.Get(d.Id())
	if err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}

	return resourceDataFromAzureConnector(ctx, d, connector)
}

func resourceAzureConnectorUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] update azure connector %s", d.Id())

	opt := azure.NewConnectorConfig().
		WithName(d.Get("name").(string)).
		WithDescription(d.Get("description").(string)).
		WithApplicationID(d.Get("application_id").(string)).
		WithDirectoryID(d.Get("directory_id").(string)).
		WithSubscriptionID(d.Get("subscription_id").(string)).
		WithAuthenticationKey(d.Get("authentication_key").(string)).
		WithIsGovCloud(d.Get("is_gov_cloud").(bool))

	service := azureService(meta)
	err := service.Update(d.Id(), opt)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceAzureConnectorRead(ctx, d, meta)
}

func resourceAzureConnectorDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Delete qualys azure connector %s", d.Id())

	service := azureService(meta)
	return diag.FromErr(service.Delete([]string{d.Id()}))
}

func resourceDataFromAzureConnector(_ context.Context, d *schema.ResourceData, connector *azure.Connector) diag.Diagnostics {
	_, ok := d.GetOk("authentication_key")
	if !ok {
		if err := d.Set("authentication_key", ""); err != nil {
			return diag.FromErr(err)
		}
	}

	return combineErrors(
		d.Set("connector_id", connector.ConnectorID),
		d.Set("name", connector.Name),
		d.Set("description", connector.Description),
		d.Set("application_id", connector.ApplicationID),
		d.Set("directory_id", connector.DirectoryID),
		d.Set("subscription_id", connector.SubscriptionID),
		d.Set("is_gov_cloud", connector.IsGovCloud),
		d.Set("cloud_provider", connector.Provider),
		d.Set("subscription_name", connector.SubscriptionName),
	)
}
