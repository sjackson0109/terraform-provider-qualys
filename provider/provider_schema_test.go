package provider

import (
	"testing"
)

// The SDK validates every resource and data source schema: attribute types,
// required/optional/computed combinations, descriptions and validators. A schema
// mistake fails here rather than at apply time in a practitioner's account.
func TestProviderSchemaIsValid(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("provider schema is invalid: %v", err)
	}
}

func TestProviderRequiresBaseURL(t *testing.T) {
	s := Provider().Schema["base_url"]
	if s == nil {
		t.Fatal("base_url is not defined")
	}
	// Qualys hosts subscriptions on several platforms with different hostnames.
	// Defaulting this would risk sending credentials to the wrong platform, so it
	// must stay required.
	if !s.Required {
		t.Error("base_url must be required; there is no safe default platform hostname")
	}
	if s.Default != nil {
		t.Error("base_url must not have a hardcoded default")
	}
}

func TestPasswordIsSensitive(t *testing.T) {
	if s := Provider().Schema["password"]; s == nil || !s.Sensitive {
		t.Error("password must be marked sensitive so it is redacted from plan output and logs")
	}
}

func TestAssetGroupNetworkForcesReplacement(t *testing.T) {
	r := Provider().ResourcesMap["qualys_asset_group"]
	if r == nil {
		t.Fatal("qualys_asset_group is not registered")
	}
	// The Qualys edit action exposes no set_network_id, so an in-place move
	// between networks is impossible; the schema must force replacement instead
	// of silently ignoring the change.
	if s := r.Schema["network_id"]; s == nil || !s.ForceNew {
		t.Error("network_id must be ForceNew: the API cannot move a group between networks")
	}
}

func TestAssetGroupIsImportable(t *testing.T) {
	r := Provider().ResourcesMap["qualys_asset_group"]
	if r == nil || r.Importer == nil {
		t.Error("qualys_asset_group must be importable so existing groups can be adopted")
	}
}
