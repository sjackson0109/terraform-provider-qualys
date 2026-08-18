package provider

import (
	"context"
	"errors"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
)

func resourceAWSConnector() *schema.Resource {
	s := connectorCommonSchema()
	s["arn"] = &schema.Schema{
		Description: "The ARN of the IAM role Qualys assumes to scan this AWS account",
		Type:        schema.TypeString,
		Required:    true,
	}
	s["external_id"] = &schema.Schema{
		Description: "The external ID configured in the IAM role's trust policy",
		Type:        schema.TypeString,
		Required:    true,
	}
	s["all_regions"] = &schema.Schema{
		Description: "Whether the connector scans all AWS regions. The Connector v3 " +
			"API's per-region endpoint selection is not carried by this provider yet, " +
			"so false currently produces a connector with no regions selected.",
		Type:     schema.TypeBool,
		Optional: true,
		Default:  true,
	}
	s["is_gov_cloud"] = &schema.Schema{
		Description: "Whether this connector targets an AWS GovCloud account",
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
	}
	s["is_china_region"] = &schema.Schema{
		Description: "Whether this connector targets an AWS China region account. " +
			"Read-only under the Connector v3 API.",
		Type:       schema.TypeBool,
		Optional:   true,
		Computed:   true,
		Deprecated: "The Connector v3 API has no China-region write field; the value is read-only.",
	}
	s["is_portal_connector"] = &schema.Schema{
		Description: "No effect. The Connector v3 API always manages the portal " +
			"(AssetView) connector itself.",
		Type:       schema.TypeBool,
		Optional:   true,
		Deprecated: "The Connector v3 API has no portal-connector field; this attribute is ignored.",
	}
	s["aws_account_id"] = &schema.Schema{
		Description: "The AWS account ID associated with this connector",
		Type:        schema.TypeString,
		Computed:    true,
	}
	s["qualys_aws_account_id"] = &schema.Schema{
		Description: "The Qualys-owned AWS account that assumes the role; the " +
			"account to trust in the IAM role's policy",
		Type:     schema.TypeString,
		Computed: true,
	}

	return &schema.Resource{
		Description: "A Qualys connector used for scanning AWS account assets, " +
			"via the Connector v3 API",
		CreateContext: resourceAWSConnectorCreate,
		ReadContext:   resourceAWSConnectorRead,
		UpdateContext: resourceAWSConnectorUpdate,
		DeleteContext: resourceAWSConnectorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: s,
	}
}

func awsConnectorInput(d *schema.ResourceData) qps.AWSConnectorInput {
	allRegions := d.Get("all_regions").(bool)
	govCloud := d.Get("is_gov_cloud").(bool)
	return qps.AWSConnectorInput{
		ConnectorBaseInput: connectorBaseInput(d),
		ARN:                d.Get("arn").(string),
		ExternalID:         d.Get("external_id").(string),
		AllRegions:         &allRegions,
		IsGovCloud:         &govCloud,
	}
}

func resourceAWSConnectorCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] create aws connector %q", d.Get("name").(string))

	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	conn, err := client.CreateAWSConnector(ctx, awsConnectorInput(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(conn.ID)
	return resourceAWSConnectorRead(ctx, d, meta)
}

func resourceAWSConnectorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] read aws connector %q", d.Id())

	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	// State written by provider versions that used the deprecated CloudView
	// v1 API holds the connector's UUID. The v3 API reports that UUID as
	// cloudviewUuid, so a one-time lookup adopts the numeric v3 id in place.
	if isCloudViewUUID(d.Id()) {
		conn, err := client.FindAWSConnectorByCloudViewUUID(ctx, d.Id())
		if err != nil {
			if errors.Is(err, qps.ErrNotFound) {
				return diag.Errorf("aws connector: state id %s is a CloudView v1 UUID and "+
					"no Connector v3 connector reports it as cloudviewUuid; remove it from "+
					"state and re-import by its numeric v3 id "+
					"(terraform state rm, then terraform import)", d.Id())
			}
			return diag.FromErr(err)
		}
		log.Printf("[INFO] aws connector: adopted Connector v3 id %s for CloudView UUID %s",
			conn.ID, d.Id())
		d.SetId(conn.ID)
	}

	conn, err := client.GetAWSConnector(ctx, d.Id())
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

func resourceAWSConnectorUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] update aws connector %s", d.Id())

	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	// The v3 update responds with the id only; Read refreshes the rest.
	if err := client.UpdateAWSConnector(ctx, d.Id(), awsConnectorInput(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceAWSConnectorRead(ctx, d, meta)
}

func resourceAWSConnectorDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] delete aws connector %s", d.Id())

	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(client.DeleteAWSConnector(ctx, d.Id()))
}
