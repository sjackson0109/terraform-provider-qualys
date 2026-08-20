package provider

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
)

func resourceGCPConnector() *schema.Resource {
	s := connectorCommonSchema()
	s["gcp_credentials_json"] = &schema.Schema{
		Description: "The JSON key file for a GCP service account, verbatim. The " +
			"Connector v3 API embeds its fields directly, so the file's own " +
			"project_id determines the scanned project. The API never returns " +
			"key material, so this value is write-only and drift is not detected.",
		Type:      schema.TypeString,
		Sensitive: true,
		Required:  true,
	}
	s["project_id"] = &schema.Schema{
		Description: "GCP project id. Derived from gcp_credentials_json; if set " +
			"explicitly it must match the key file's project_id.",
		Type:     schema.TypeString,
		Optional: true,
		Computed: true,
	}

	return &schema.Resource{
		Description: "A Qualys connector used for scanning GCP project assets, " +
			"via the Connector v3 API",
		CreateContext: resourceGCPConnectorCreate,
		ReadContext:   resourceGCPConnectorRead,
		UpdateContext: resourceGCPConnectorUpdate,
		DeleteContext: resourceGCPConnectorDelete,
		CustomizeDiff: validateGCPProjectMatchesKey,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: s,
	}
}

// gcpProjectIDConflict is the actual comparison behind
// validateGCPProjectMatchesKey, extracted so it is unit-testable without
// driving a real Terraform plan.
//
// configuredExplicitly must reflect only what the user's current
// configuration literally contains — never a value merely inherited from
// prior state. See validateGCPProjectMatchesKey for why that distinction is
// the entire point of this check.
func gcpProjectIDConflict(configuredExplicitly bool, configuredProjectID, credentialsJSON string) error {
	if !configuredExplicitly || configuredProjectID == "" || credentialsJSON == "" {
		return nil
	}
	var key struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal([]byte(credentialsJSON), &key); err != nil || key.ProjectID == "" {
		// An undecodable key fails at apply with a clearer message; an absent
		// project_id in the key leaves nothing to contradict.
		return nil
	}
	if key.ProjectID != configuredProjectID {
		return errors.New("project_id does not match the key file's project_id; " +
			"the connector scans the key's project, so either fix the mismatch or " +
			"leave project_id unset to derive it")
	}
	return nil
}

// validateGCPProjectMatchesKey catches a project_id that contradicts the key
// file at plan time. The API scans the key's project regardless, so a
// mismatched explicit project_id is a config error, not a preference.
//
// The check only fires when project_id is genuinely present in the current
// configuration, determined via GetRawConfig rather than d.Get. project_id
// is Optional+Computed, so on an update where the user never set it, d.Get
// would return whatever value Read previously stored in state from a
// *different* key — comparing that stale value against a newly rotated
// key's project_id would reject a legitimate credential/project rotation
// the user never asked to be validated against the old project at all.
// GetRawConfig reflects only what Terraform's current configuration
// literally contains (verified against this SDK version's source,
// helper/schema/resource_diff.go: it returns exactly
// terraform.InstanceDiff.RawConfig, a null value when Terraform sent none).
//
// This wrapper is deliberately thin and not itself unit tested:
// GetRawConfig is correctly populated by real Terraform runs over the
// plugin protocol, but this project's own diffFor test helper drives
// CustomizeDiff through the older *schema.Resource.Diff() path, which
// predates GetRawConfig and never populates it — so a unit test built on
// diffFor cannot observe this function's real behaviour either way. See
// gcpProjectIDConflict, which carries the actual logic and is fully unit
// tested, and the acceptance test in resource_gcp_connector_acc_test.go,
// gated on TF_ACC=1 (needs a terraform binary this environment does not
// have installed — see docs/discovery/12-connector-v3-migration.md).
func validateGCPProjectMatchesKey(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	raw := d.GetRawConfig()
	explicit := false
	configured := ""
	if !raw.IsNull() {
		if pidVal := raw.GetAttr("project_id"); !pidVal.IsNull() && pidVal.IsKnown() {
			explicit = true
			configured = pidVal.AsString()
		}
	}
	return gcpProjectIDConflict(explicit, configured, d.Get("gcp_credentials_json").(string))
}

func gcpConnectorInput(d *schema.ResourceData) qps.GCPConnectorInput {
	return qps.GCPConnectorInput{
		ConnectorBaseInput: connectorBaseInput(d),
		CredentialsJSON:    d.Get("gcp_credentials_json").(string),
	}
}

func resourceGCPConnectorCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] create gcp connector %q", d.Get("name").(string))

	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	conn, err := client.CreateGCPConnector(ctx, gcpConnectorInput(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(conn.ID)
	return resourceGCPConnectorRead(ctx, d, meta)
}

func resourceGCPConnectorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] read gcp connector %q", d.Id())

	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	// State written by provider versions that used the deprecated CloudView
	// v1 API holds the connector's UUID. The v3 API reports that UUID as
	// cloudviewUuid, so a one-time lookup adopts the numeric v3 id in place.
	if isCloudViewUUID(d.Id()) {
		conn, err := client.FindGCPConnectorByCloudViewUUID(ctx, d.Id())
		if err != nil {
			if errors.Is(err, qps.ErrNotFound) {
				return diag.Errorf("gcp connector: state id %s is a CloudView v1 UUID and "+
					"no Connector v3 connector reports it as cloudviewUuid; remove it from "+
					"state and re-import by its numeric v3 id "+
					"(terraform state rm, then terraform import)", d.Id())
			}
			return diag.FromErr(err)
		}
		log.Printf("[INFO] gcp connector: adopted Connector v3 id %s for CloudView UUID %s",
			conn.ID, d.Id())
		d.SetId(conn.ID)
	}

	conn, err := client.GetGCPConnector(ctx, d.Id())
	if err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			return qpsNotFoundOnRead(d, "GCP connector")
		}
		return diag.FromErr(err)
	}

	if err := setConnectorBaseData(d, conn.ConnectorBase); err != nil {
		return diag.FromErr(err)
	}
	return combineErrors(
		d.Set("project_id", conn.ProjectID),
		d.Set("cloud_provider", "GCP"),
	)
}

func resourceGCPConnectorUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] update gcp connector %s", d.Id())

	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	// The v3 update responds with the id only; Read refreshes the rest.
	if err := client.UpdateGCPConnector(ctx, d.Id(), gcpConnectorInput(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceGCPConnectorRead(ctx, d, meta)
}

func resourceGCPConnectorDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] delete gcp connector %s", d.Id())

	client, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := client.DeleteGCPConnector(ctx, d.Id()); err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			// Already gone (or already out of scope) from this caller's
			// perspective — either way, the desired end state of Delete is
			// achieved, so this must succeed rather than error.
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
