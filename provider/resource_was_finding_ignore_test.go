package provider

import "testing"

func TestWASFindingIgnoreRequiresFindingIDAndComment(t *testing.T) {
	r := resourceWASFindingIgnore()
	if !r.Schema["finding_id"].Required {
		t.Error("finding_id should be required")
	}
	if !r.Schema["finding_id"].ForceNew {
		t.Error("finding_id should be ForceNew: this resource ignores one finding, not a moving target")
	}
	if !r.Schema["comment"].Required {
		t.Error("comment should be required: ignoring without a justification defeats the audit trail")
	}
}
