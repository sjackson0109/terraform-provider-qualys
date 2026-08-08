package provider

import (
	"strings"
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

// Two resources model objects the Qualys API cannot delete. Their destroy must
// warn rather than fail (which would make them un-destroyable) or silently
// succeed (which would imply the object is gone).
func TestResourcesWithNoDeleteAPIAreDocumented(t *testing.T) {
	for _, name := range []string{"qualys_network", "qualys_ip_registration"} {
		r := Provider().ResourcesMap[name]
		if r == nil {
			t.Errorf("%s is not registered", name)
			continue
		}
		if !strings.Contains(r.Description, "does not") {
			t.Errorf("%s must document that destroy does not remove the object in Qualys", name)
		}
	}
}

// Registering an address is not reversible through the API, so the address set
// and its network cannot be changed in place.
func TestIPRegistrationForcesReplacementOnIdentity(t *testing.T) {
	r := Provider().ResourcesMap["qualys_ip_registration"]
	if r == nil {
		t.Fatal("qualys_ip_registration is not registered")
	}
	for _, attr := range []string{"ips", "network_id"} {
		if s := r.Schema[attr]; s == nil || !s.ForceNew {
			t.Errorf("%s must be ForceNew: the API cannot un-register an address", attr)
		}
	}
}

// Purging is destructive and irreversible, so it must be opt-in.
func TestPurgeOnDestroyDefaultsOff(t *testing.T) {
	s := Provider().ResourcesMap["qualys_ip_registration"].Schema["purge_on_destroy"]
	if s == nil {
		t.Fatal("purge_on_destroy is not defined")
	}
	if s.Default != false {
		t.Error("purge_on_destroy must default to false; purging deletes vulnerability history")
	}
}
