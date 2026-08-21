package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
)

// TestIPRegistrationReadRefreshesOwnerAndComment is a regression test:
// owner/comment are genuinely returned by ListHosts (vmdr/ip.go sets
// details=All specifically so OWNER/COMMENTS come back), but Read's values
// map used to omit them — an operator changing a host's owner or comment
// outside Terraform was never detected, and the next apply for an
// unrelated change would silently revert it to whatever stale value
// Terraform still had in state.
func TestIPRegistrationReadRefreshesOwnerAndComment(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<HOST_LIST_OUTPUT><RESPONSE><HOST_LIST>
		  <HOST>
		    <ID>1001</ID><IP>10.0.0.5</IP><TRACKING_METHOD>IP</TRACKING_METHOD>
		    <OWNER>alice</OWNER><COMMENTS>prod web tier</COMMENTS>
		  </HOST>
		</HOST_LIST></RESPONSE></HOST_LIST_OUTPUT>`)
	}))
	defer srv.Close()
	c, err := vmdr.NewClient(vmdr.Config{
		BaseURL: srv.URL, Username: "u", Password: "p", HTTPClient: srv.Client(), MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("vmdr.NewClient: %v", err)
	}

	r := resourceIPRegistration()
	// ips is a TypeSet with a custom hash function (hashNormalizedIP); its
	// InstanceState flatmap key is that hash, not a sequential index, so
	// the set is populated via d.Set (schema-aware) rather than a
	// hand-built InstanceState.Attributes map.
	d := r.Data(&terraform.InstanceState{ID: "irrelevant"})
	if err := d.Set("ips", []interface{}{"10.0.0.5"}); err != nil {
		t.Fatalf("Set ips: %v", err)
	}
	diags := r.ReadContext(context.Background(), d, &clients{vmdr: c})
	if diags.HasError() {
		t.Fatalf("Read: %v", diags)
	}
	if got := d.Get("owner").(string); got != "alice" {
		t.Errorf("owner = %q, want %q", got, "alice")
	}
	if got := d.Get("comment").(string); got != "prod web tier" {
		t.Errorf("comment = %q, want %q", got, "prod web tier")
	}
}
