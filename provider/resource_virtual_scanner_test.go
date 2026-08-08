package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The activation code cannot be recovered once the appliance activates, so it
// must be stored and must not leak into logs or plan output.
func TestActivationCodeIsSensitiveAndComputed(t *testing.T) {
	s := resourceVirtualScanner().Schema["activation_code"]
	if s == nil {
		t.Fatal("activation_code is not defined")
	}
	if !s.Sensitive {
		t.Error("activation_code must be Sensitive: it personalises an appliance VM")
	}
	if !s.Computed {
		t.Error("activation_code must be Computed: it is issued by Qualys, not configured")
	}
}

func TestVirtualScannerPollingIntervalRange(t *testing.T) {
	v := resourceVirtualScanner().Schema["polling_interval"].ValidateFunc
	if _, errs := v(180, "polling_interval"); len(errs) != 0 {
		t.Errorf("the documented default was rejected: %v", errs)
	}
	for _, bad := range []int{30, 3600} {
		if _, errs := v(bad, "polling_interval"); len(errs) == 0 {
			t.Errorf("expected %d to be rejected; the documented range is 60-360", bad)
		}
	}
}

// asset_group_id must be omitted for Manager accounts, so it cannot be changed
// in place without recreating the appliance in a different group.
func TestVirtualScannerAssetGroupForcesNew(t *testing.T) {
	if s := resourceVirtualScanner().Schema["asset_group_id"]; s == nil || !s.ForceNew {
		t.Error("asset_group_id must be ForceNew: an appliance cannot change asset group in place")
	}
}

// VLAN and route lists are replaced wholesale by the API, so the schema must
// make clear that declaring the block takes ownership.
func TestVLANBlockDocumentsAuthority(t *testing.T) {
	s := resourceVirtualScanner().Schema["vlan"]
	if s == nil {
		t.Fatal("vlan is not defined")
	}
	if s.Elem == nil {
		t.Fatal("vlan has no element schema")
	}
	elem, ok := s.Elem.(*schema.Resource)
	if !ok {
		t.Fatal("vlan element is not a resource")
	}
	for _, required := range []string{"vlan_id", "ip", "netmask", "name"} {
		if f := elem.Schema[required]; f == nil || !f.Required {
			t.Errorf("vlan.%s must be required: the API encodes all four fields positionally, "+
				"so a missing one shifts the rest", required)
		}
	}
}
