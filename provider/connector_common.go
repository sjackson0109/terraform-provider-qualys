package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
)

// Shared schema and mapping for the three Connector v3 resources
// (qualys_gcp_connector, qualys_aws_connector, qualys_azure_connector).
// The per-cloud files carry only their cloud's credential fields.

// connectorCommonSchema returns the attributes every connector shares.
// The returned map is fresh per call: schema maps are mutated by the SDK
// and by the per-cloud resources, so sharing one instance would leak
// per-cloud attributes across resources.
func connectorCommonSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"connector_id": {
			Description: "The connector's id. Connector v3 ids are numeric; state " +
				"created by provider versions that used the deprecated CloudView API " +
				"holds a UUID instead, which Read adopts automatically (see the " +
				"migration notes in the resource documentation).",
			Type:     schema.TypeString,
			Computed: true,
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
		"run_frequency": {
			Description: "How often the connector polls the cloud account, in minutes. " +
				"The GCP API documents this as required; 240 matches the Qualys UI default.",
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      240,
			ValidateFunc: validation.IntAtLeast(1),
		},
		"disabled": {
			Description: "Disables connector execution without deleting it",
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
		},
		"is_remediation_enabled": {
			Description: "Enables Qualys remediation for assets discovered by this connector",
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
		},
		"activation": {
			Description: "Qualys modules activated for discovered assets " +
				"(VM, CERTVIEW, CLOUDVIEW, SCA, CSA)",
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
				ValidateFunc: validation.StringInSlice([]string{
					qps.ActivationVM, qps.ActivationCertView, qps.ActivationCloudView,
					qps.ActivationSCA, qps.ActivationCSA,
				}, false),
			},
		},
		"default_tag_ids": {
			Description: "Asset tag ids applied to every asset this connector discovers",
			Type:        schema.TypeSet,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"connector_state": {
			Description: "The connector's lifecycle state (QUEUED, RUNNING, " +
				"FINISHED_SUCCESS, FINISHED_ERRORS, ...)",
			Type:     schema.TypeString,
			Computed: true,
		},
		"last_sync": {
			Description: "When the connector last synchronised, ISO-8601 UTC",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"next_sync": {
			Description: "When the connector next synchronises, ISO-8601 UTC",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"cloudview_uuid": {
			Description: "The deprecated CloudView v1 API's UUID for this connector, " +
				"as reported by the v3 API. Empty for connectors that never existed in v1.",
			Type:     schema.TypeString,
			Computed: true,
		},
		"cloud_provider": {
			Description: "The cloud provider associated with this connector",
			Type:        schema.TypeString,
			Computed:    true,
		},
	}
}

// connectorBaseInput reads the shared attributes into a client input.
func connectorBaseInput(d *schema.ResourceData) qps.ConnectorBaseInput {
	disabled := d.Get("disabled").(bool)
	remediation := d.Get("is_remediation_enabled").(bool)
	return qps.ConnectorBaseInput{
		Name:                 d.Get("name").(string),
		Description:          d.Get("description").(string),
		RunFrequency:         d.Get("run_frequency").(int),
		Disabled:             &disabled,
		IsRemediationEnabled: &remediation,
		ActivationModules:    stringSet(d, "activation"),
		DefaultTagIDs:        stringSet(d, "default_tag_ids"),
	}
}

// setConnectorBaseData writes the shared read-side attributes.
func setConnectorBaseData(d *schema.ResourceData, base qps.ConnectorBase) error {
	tagIDs := make([]string, 0, len(base.DefaultTags))
	for _, t := range base.DefaultTags {
		tagIDs = append(tagIDs, t.ID)
	}
	values := map[string]interface{}{
		"connector_id":           base.ID,
		"name":                   base.Name,
		"description":            base.Description,
		"run_frequency":          base.RunFrequency,
		"disabled":               base.Disabled,
		"is_remediation_enabled": base.IsRemediationEnabled,
		"connector_state":        base.State,
		"last_sync":              base.LastSync,
		"next_sync":              base.NextSync,
		"cloudview_uuid":         base.CloudViewUUID,
	}
	// The response-side collection nesting is doc-tolerant rather than
	// Confirmed (doc 12); when decoding yields nothing for an attribute the
	// user did not configure, leave it unset rather than writing an empty
	// set over a configured value.
	if len(base.ActivationModules) > 0 || len(stringSet(d, "activation")) == 0 {
		values["activation"] = base.ActivationModules
	}
	if len(tagIDs) > 0 || len(stringSet(d, "default_tag_ids")) == 0 {
		values["default_tag_ids"] = tagIDs
	}
	return setAll(d, values)
}

// isCloudViewUUID reports whether an id in state is a CloudView v1 UUID
// rather than a numeric Connector v3 id.
func isCloudViewUUID(id string) bool {
	return strings.Contains(id, "-")
}

// connectorDataSourceSchema derives the shared data-source attributes from
// the resource schema: everything computed, except the two lookup inputs
// (connector_id or name, exactly one).
func connectorDataSourceSchema() map[string]*schema.Schema {
	s := connectorCommonSchema()
	for _, v := range s {
		v.Required = false
		v.Optional = false
		v.Computed = true
		v.Default = nil
		v.ValidateFunc = nil
	}
	s["connector_id"].Optional = true
	s["connector_id"].ExactlyOneOf = []string{"connector_id", "name"}
	s["connector_id"].Description = "The connector's numeric Connector v3 id"
	s["name"].Optional = true
	s["name"].ExactlyOneOf = []string{"connector_id", "name"}
	return s
}

// nameFilter builds the search criteria for a lookup by connector name.
func nameFilter(name string) *qps.Filters {
	return &qps.Filters{Criteria: []qps.Criterion{
		{Field: "name", Operator: "EQUALS", Value: name},
	}}
}
