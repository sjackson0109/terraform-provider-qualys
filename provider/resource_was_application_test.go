package provider

import "testing"

func TestWASApplicationRequiresNameAndURL(t *testing.T) {
	r := resourceWASApplication()
	if !r.Schema["name"].Required {
		t.Error("name should be required")
	}
	if !r.Schema["url"].Required {
		t.Error("url should be required")
	}
}

func TestWASApplicationTagIDsIsASet(t *testing.T) {
	r := resourceWASApplication()
	if r.Schema["tag_ids"].Type.String() != "TypeSet" {
		t.Errorf("tag_ids should be a set so ordering never causes a spurious diff, got %s",
			r.Schema["tag_ids"].Type)
	}
}

func TestWASApplicationAuthRecordIDsIsASet(t *testing.T) {
	r := resourceWASApplication()
	if r.Schema["auth_record_ids"].Type.String() != "TypeSet" {
		t.Errorf("auth_record_ids should be a set so ordering never causes a spurious diff, got %s",
			r.Schema["auth_record_ids"].Type)
	}
}

func TestWASApplicationCancelScansFieldsConflict(t *testing.T) {
	r := resourceWASApplication()
	at := r.Schema["cancel_scans_at"]
	after := r.Schema["cancel_scans_after_hours"]
	if len(at.ConflictsWith) != 1 || at.ConflictsWith[0] != "cancel_scans_after_hours" {
		t.Errorf("cancel_scans_at.ConflictsWith = %v", at.ConflictsWith)
	}
	if len(after.ConflictsWith) != 1 || after.ConflictsWith[0] != "cancel_scans_at" {
		t.Errorf("cancel_scans_after_hours.ConflictsWith = %v", after.ConflictsWith)
	}
}

func TestWASApplicationSwaggerAndPostmanConflict(t *testing.T) {
	r := resourceWASApplication()
	swagger := r.Schema["swagger_file"]
	postman := r.Schema["postman_collection"]
	if len(swagger.ConflictsWith) != 1 || swagger.ConflictsWith[0] != "postman_collection" {
		t.Errorf("swagger_file.ConflictsWith = %v", swagger.ConflictsWith)
	}
	if len(postman.ConflictsWith) != 1 || postman.ConflictsWith[0] != "swagger_file" {
		t.Errorf("postman_collection.ConflictsWith = %v", postman.ConflictsWith)
	}
}

func TestWASApplicationCrawlingScriptsIsComputedOnly(t *testing.T) {
	r := resourceWASApplication()
	cs := r.Schema["crawling_scripts"]
	if !cs.Computed || cs.Optional || cs.Required {
		t.Errorf("crawling_scripts should be read-only (Computed only), got Optional=%v Required=%v Computed=%v",
			cs.Optional, cs.Required, cs.Computed)
	}
}

func TestDiffStringSets(t *testing.T) {
	cases := []struct {
		name            string
		old, new        []string
		wantAdd, wantRm []string
	}{
		{"no change", []string{"1", "2"}, []string{"1", "2"}, nil, nil},
		{"add only", []string{"1"}, []string{"1", "2"}, []string{"2"}, nil},
		{"remove only", []string{"1", "2"}, []string{"1"}, nil, []string{"2"}},
		{"add and remove", []string{"1", "2"}, []string{"2", "3"}, []string{"3"}, []string{"1"}},
		{"empty to populated", nil, []string{"1"}, []string{"1"}, nil},
		{"populated to empty", []string{"1"}, nil, nil, []string{"1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			add, rm := diffStringSets(tc.old, tc.new)
			if !sameStringSet(add, tc.wantAdd) {
				t.Errorf("added = %v, want %v", add, tc.wantAdd)
			}
			if !sameStringSet(rm, tc.wantRm) {
				t.Errorf("removed = %v, want %v", rm, tc.wantRm)
			}
		})
	}
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]bool, len(a))
	for _, v := range a {
		seen[v] = true
	}
	for _, v := range b {
		if !seen[v] {
			return false
		}
	}
	return true
}
