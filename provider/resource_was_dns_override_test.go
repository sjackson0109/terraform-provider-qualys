package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestWASDNSOverrideRequiresNameAndMapping(t *testing.T) {
	r := resourceWASDNSOverride()
	if !r.Schema["name"].Required {
		t.Error("name should be required")
	}
	if !r.Schema["mapping"].Required {
		t.Error("mapping should be required")
	}
	if r.Schema["mapping"].MinItems != 1 {
		t.Error("mapping should require at least one entry: a DNS override with none has no effect")
	}
}

func TestWASDNSOverrideMappingFieldsAreRequired(t *testing.T) {
	elem := resourceWASDNSOverride().Schema["mapping"].Elem.(*schema.Resource)
	if !elem.Schema["host_name"].Required {
		t.Error("mapping.host_name should be required")
	}
	if !elem.Schema["ip_address"].Required {
		t.Error("mapping.ip_address should be required")
	}
}
