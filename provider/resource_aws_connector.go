package provider

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/form3tech-oss/terraform-provider-qualys/cloudview/aws"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAWSConnector() *schema.Resource {
	return &schema.Resource{
		Description:   "A Qualys connector used for scanning AWS account assets",
		CreateContext: resourceAWSConnectorCreate,
		ReadContext:   resourceAWSConnectorRead,
		UpdateContext: resourceAWSConnectorUpdate,
		DeleteContext: resourceAWSConnectorDelete,
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
			"arn": {
				Description: "The ARN of the IAM role Qualys assumes to scan this AWS account",
				Type:        schema.TypeString,
				Required:    true,
			},
			"external_id": {
				Description: "The external ID configured in the IAM role's trust policy",
				Type:        schema.TypeString,
				Required:    true,
			},
			"is_portal_connector": {
				Description: "Whether an AssetView connector is also created alongside this CloudView connector",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"is_gov_cloud": {
				Description: "Whether this connector targets an AWS GovCloud account",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"is_china_region": {
				Description: "Whether this connector targets an AWS China region account",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"cloud_provider": {
				Description: "The cloud provider associated with this connector",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"aws_account_id": {
				Description: "The AWS account ID associated with this connector",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceAWSConnectorCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] create aws connector %q", d.Get("name").(string))

	opt := aws.NewConnectorConfig().
		WithName(d.Get("name").(string)).
		WithDescription(d.Get("description").(string)).
		WithARN(d.Get("arn").(string)).
		WithExternalID(d.Get("external_id").(string)).
		WithIsPortalConnector(d.Get("is_portal_connector").(bool)).
		WithIsGovCloud(d.Get("is_gov_cloud").(bool)).
		WithIsChinaRegion(d.Get("is_china_region").(bool))

	service := awsService(meta)
	connector, err := service.Create(opt)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(connector.ConnectorID)

	return resourceAWSConnectorRead(ctx, d, meta)
}

func resourceAWSConnectorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Reading aws connector %q ", d.Id())

	service := awsService(meta)
	connector, err := service.Get(d.Id())
	if err != nil {
		d.SetId("")
		return diag.FromErr(err)
	}

	return resourceDataFromAWSConnector(ctx, d, connector)
}

func resourceAWSConnectorUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] update aws connector %s", d.Id())

	opt := aws.NewConnectorConfig().
		WithName(d.Get("name").(string)).
		WithDescription(d.Get("description").(string)).
		WithARN(d.Get("arn").(string)).
		WithExternalID(d.Get("external_id").(string)).
		WithIsPortalConnector(d.Get("is_portal_connector").(bool)).
		WithIsGovCloud(d.Get("is_gov_cloud").(bool)).
		WithIsChinaRegion(d.Get("is_china_region").(bool))

	service := awsService(meta)
	err := service.Update(d.Id(), opt)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceAWSConnectorRead(ctx, d, meta)
}

func resourceAWSConnectorDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Delete qualys aws connector %s", d.Id())

	service := awsService(meta)
	return diag.FromErr(service.Delete([]string{d.Id()}))
}

func resourceDataFromAWSConnector(_ context.Context, d *schema.ResourceData, connector *aws.Connector) diag.Diagnostics {
	return combineErrors(
		d.Set("connector_id", connector.ConnectorID),
		d.Set("name", connector.Name),
		d.Set("description", connector.Description),
		d.Set("arn", connector.ARN),
		d.Set("external_id", connector.ExternalID),
		d.Set("is_portal_connector", connector.IsPortalConnector),
		d.Set("is_gov_cloud", connector.IsGovCloud),
		d.Set("is_china_region", connector.IsChinaRegion),
		d.Set("cloud_provider", connector.Provider),
		d.Set("aws_account_id", connector.AWSAccountID),
	)
}
