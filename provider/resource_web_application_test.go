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
