package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The API-level connector behaviour (envelope shapes, string-boolean
// tolerance, cloudviewUuid adoption lookup) is tested in the qps package
// against a TLS mock; these tests cover the plan-time surface.

// gcpProjectIDConflict carries the actual comparison logic behind
// validateGCPProjectMatchesKey's CustomizeDiff, and is unit tested
// directly. The CustomizeDiff wrapper itself relies on GetRawConfig to
// distinguish "the user explicitly set project_id in this configuration"
// from "d.Get is returning a value merely inherited from prior state" —
// this repo's diffFor test helper drives CustomizeDiff through the older
// *schema.Resource.Diff() path, which never populates GetRawConfig, so it
// cannot exercise that distinction either way (see the wrapper's doc
// comment). These tests cover what diffFor can meaningfully assert: the
// wrapper doesn't panic and doesn't misuse gcpProjectIDConflict's
// zero-value ("not explicitly configured") case.

func TestGCPProjectIDConflictRejectsExplicitMismatch(t *testing.T) {
	err := gcpProjectIDConflict(true, "other-project",
		`{"type":"service_account","project_id":"actual-project"}`)
	if err == nil {
		t.Fatal("expected an error: the connector scans the key's project, " +
			"so a contradicting explicit project_id is a config error")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGCPProjectIDConflictAcceptsExplicitMatch(t *testing.T) {
	err := gcpProjectIDConflict(true, "my-project",
		`{"type":"service_account","project_id":"my-project"}`)
	if err != nil {
		t.Errorf("a matching explicit project_id should be valid: %v", err)
	}
}

func TestGCPProjectIDConflictSkipsWhenNotExplicitlyConfigured(t *testing.T) {
	// This is the regression case: configuredExplicitly=false must be
	// treated as "nothing to validate" even when configuredProjectID
	// carries a stale value from prior state (e.g. project-a) that
	// contradicts a freshly rotated key for a different project
	// (project-b) — rejecting that would block a legitimate credential
	// rotation the user never asked to have validated against the old
	// project at all.
	err := gcpProjectIDConflict(false, "project-a",
		`{"type":"service_account","project_id":"project-b"}`)
	if err != nil {
		t.Errorf("configuredExplicitly=false must never produce an error, got: %v", err)
	}
}

func TestGCPConnectorPlanTimeCheckDoesNotErrorWithoutExplicitConfig(t *testing.T) {
	// diffFor cannot populate GetRawConfig (see comment above), so under
	// this harness validateGCPProjectMatchesKey always takes the
	// "nothing explicitly configured" branch — this just confirms the
	// wrapper itself doesn't panic or misfire in that branch.
	key := `{"type":"service_account","project_id":"my-project"}`
	if err := diffFor(t, resourceGCPConnector(), map[string]interface{}{
		"name": "dev-gcp", "gcp_credentials_json": key, "project_id": "my-project",
	}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := diffFor(t, resourceGCPConnector(), map[string]interface{}{
		"name": "dev-gcp", "gcp_credentials_json": key,
	}); err != nil {
		t.Errorf("unexpected error: %v", err)
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
