package provider

import (
	"strings"
	"testing"
)

func TestOptionProfileRejectsCustomDetectionWithoutSearchLists(t *testing.T) {
	err := diffFor(t, resourceVMOptionProfile(), map[string]interface{}{
		"title":                   "web-scan",
		"vulnerability_detection": "custom",
	})
	if err == nil {
		t.Fatal("expected a plan-time error: Qualys silently falls back to complete " +
			"detection, which would scan far more than intended")
	}
	if !strings.Contains(err.Error(), "custom_search_list_ids") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOptionProfileAcceptsCustomDetectionWithSearchLists(t *testing.T) {
	err := diffFor(t, resourceVMOptionProfile(), map[string]interface{}{
		"title":                   "web-scan",
		"vulnerability_detection": "custom",
		"custom_search_list_ids":  []interface{}{"12345"},
	})
	if err != nil {
		t.Errorf("valid custom detection rejected: %v", err)
	}
}

// Qualys makes a default profile global as a side effect. Accepting
// is_default=true with global=false would produce a resource whose state never
// matches its configuration.
func TestOptionProfileRejectsDefaultWithoutGlobal(t *testing.T) {
	err := diffFor(t, resourceVMOptionProfile(), map[string]interface{}{
		"title":      "default-profile",
		"is_default": true,
	})
	if err == nil {
		t.Fatal("expected a plan-time error: a default profile is also made global")
	}
	if !strings.Contains(err.Error(), "global") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOptionProfileAcceptsDefaultWithGlobal(t *testing.T) {
	err := diffFor(t, resourceVMOptionProfile(), map[string]interface{}{
		"title":      "default-profile",
		"is_default": true,
		"global":     true,
	})
	if err != nil {
		t.Errorf("valid default profile rejected: %v", err)
	}
}

func TestOptionProfilePortValidation(t *testing.T) {
	v := resourceVMOptionProfile().Schema["scan_tcp_ports"].ValidateFunc
	if _, errs := v("full", "scan_tcp_ports"); len(errs) != 0 {
		t.Errorf("valid value rejected: %v", errs)
	}
	if _, errs := v("everything", "scan_tcp_ports"); len(errs) == 0 {
		t.Error("expected an invalid port selection to be rejected")
	}
}

func TestSearchListRequiresQIDs(t *testing.T) {
	r := resourceStaticSearchList()
	if s := r.Schema["qids"]; s == nil || !s.Required {
		t.Error("qids must be required: an empty search list selects nothing and would " +
			"silently neuter any option profile referencing it")
	}
}
