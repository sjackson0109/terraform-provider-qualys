package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The API-level connector behaviour (envelope shapes, string-boolean
// tolerance, cloudviewUuid adoption lookup) is tested in the qps package
// against a TLS mock; these tests cover the plan-time surface.

func TestGCPConnectorRejectsMismatchedProjectID(t *testing.T) {
	err := diffFor(t, resourceGCPConnector(), map[string]interface{}{
		"name":                 "dev-gcp",
		"gcp_credentials_json": `{"type":"service_account","project_id":"actual-project"}`,
		"project_id":           "other-project",
	})
	if err == nil {
		t.Fatal("expected a plan-time error: the connector scans the key's project, " +
			"so a contradicting project_id is a config error")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGCPConnectorAcceptsMatchingOrDerivedProjectID(t *testing.T) {
	key := `{"type":"service_account","project_id":"my-project"}`

	if err := diffFor(t, resourceGCPConnector(), map[string]interface{}{
		"name": "dev-gcp", "gcp_credentials_json": key, "project_id": "my-project",
	}); err != nil {
		t.Errorf("a matching explicit project_id should be valid: %v", err)
	}
	if err := diffFor(t, resourceGCPConnector(), map[string]interface{}{
		"name": "dev-gcp", "gcp_credentials_json": key,
	}); err != nil {
		t.Errorf("an unset project_id should be valid (derived from the key): %v", err)
	}
}

func TestConnectorActivationRejectsUnknownModule(t *testing.T) {
	elem, ok := resourceAWSConnector().Schema["activation"].Elem.(*schema.Schema)
	if !ok || elem.ValidateFunc == nil {
		t.Fatal("activation set elements must carry a validator")
	}
	if _, errs := elem.ValidateFunc("VM", "activation"); len(errs) != 0 {
		t.Errorf("VM is a documented module: %v", errs)
	}
	if _, errs := elem.ValidateFunc("NOT_A_MODULE", "activation"); len(errs) == 0 {
		t.Error("expected an undocumented activation module to be rejected")
	}
}

func TestConnectorSecretsAreMarkedSensitive(t *testing.T) {
	if !resourceGCPConnector().Schema["gcp_credentials_json"].Sensitive {
		t.Error("gcp_credentials_json must be Sensitive")
	}
	if !resourceAzureConnector().Schema["authentication_key"].Sensitive {
		t.Error("authentication_key must be Sensitive")
	}
}

func TestConnectorDataSourcesRequireExactlyOneLookup(t *testing.T) {
	// The SDK enforces ExactlyOneOf at validate time; assert the schema
	// declares it rather than exercising the SDK's own machinery.
	for name, s := range map[string]*struct{ id, n []string }{
		"gcp": {
			dataSourceGCPConnector().Schema["connector_id"].ExactlyOneOf,
			dataSourceGCPConnector().Schema["name"].ExactlyOneOf,
		},
		"aws": {
			dataSourceAWSConnector().Schema["connector_id"].ExactlyOneOf,
			dataSourceAWSConnector().Schema["name"].ExactlyOneOf,
		},
		"azure": {
			dataSourceAzureConnector().Schema["connector_id"].ExactlyOneOf,
			dataSourceAzureConnector().Schema["name"].ExactlyOneOf,
		},
	} {
		if len(s.id) != 2 || len(s.n) != 2 {
			t.Errorf("%s data source: connector_id and name must declare ExactlyOneOf", name)
		}
	}
}

// The common schema map must be fresh per call: per-cloud resources add their
// own attributes, and a shared instance would leak them across resources.
func TestConnectorSchemasDoNotLeakAcrossClouds(t *testing.T) {
	if _, ok := resourceAWSConnector().Schema["gcp_credentials_json"]; ok {
		t.Error("the AWS connector schema carries a GCP attribute; " +
			"connectorCommonSchema must return a fresh map per call")
	}
	if _, ok := resourceAzureConnector().Schema["arn"]; ok {
		t.Error("the Azure connector schema carries an AWS attribute")
	}
}
