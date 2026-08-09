package provider

import "testing"

func TestWebApplicationRequiresNameAndURL(t *testing.T) {
	r := resourceWebApplication()
	if !r.Schema["name"].Required {
		t.Error("name should be required")
	}
	if !r.Schema["url"].Required {
		t.Error("url should be required")
	}
}

func TestWebApplicationTagIDsIsASet(t *testing.T) {
	r := resourceWebApplication()
	if r.Schema["tag_ids"].Type.String() != "TypeSet" {
		t.Errorf("tag_ids should be a set so ordering never causes a spurious diff, got %s",
			r.Schema["tag_ids"].Type)
	}
}

func TestWebApplicationAuthRecordIDsIsASet(t *testing.T) {
	r := resourceWebApplication()
	if r.Schema["auth_record_ids"].Type.String() != "TypeSet" {
		t.Errorf("auth_record_ids should be a set so ordering never causes a spurious diff, got %s",
			r.Schema["auth_record_ids"].Type)
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
