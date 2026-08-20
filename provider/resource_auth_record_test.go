package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
)

// A record with neither a password nor a vault is accepted by Qualys, and scans
// using it silently fall back to unauthenticated. That is worse than an error,
// because the scan still succeeds while reporting far less.
func TestAuthRecordRequiresACredential(t *testing.T) {
	err := diffFor(t, resourceUnixAuthRecord(), map[string]interface{}{
		"title":    "linux-scan",
		"username": "scanner",
	})
	if err == nil {
		t.Fatal("expected a plan-time error when neither password nor vault_id is set")
	}
	if !strings.Contains(err.Error(), "vault_id") {
		t.Errorf("the error should point at the alternatives: %v", err)
	}
}

func TestAuthRecordAcceptsEitherCredentialSource(t *testing.T) {
	if err := diffFor(t, resourceUnixAuthRecord(), map[string]interface{}{
		"title": "linux-scan", "username": "scanner", "password": "s3cret",
	}); err != nil {
		t.Errorf("password-based record rejected: %v", err)
	}
	if err := diffFor(t, resourceUnixAuthRecord(), map[string]interface{}{
		"title": "linux-scan", "username": "scanner",
		"vault_id": "42", "vault_type": "CyberArk AIM",
	}); err != nil {
		t.Errorf("vault-based record rejected: %v", err)
	}
}

func TestAuthRecordPasswordIsSensitiveAndOptional(t *testing.T) {
	s := resourceUnixAuthRecord().Schema["password"]
	if s == nil {
		t.Fatal("password is not defined")
	}
	if !s.Sensitive {
		t.Error("password must be Sensitive")
	}
	// Required would prevent vault-based records, which are the better pattern.
	if s.Required {
		t.Error("password must be optional so a vault reference can be used instead")
	}
	if len(s.ConflictsWith) == 0 {
		t.Error("password and vault_id are alternatives and must conflict")
	}
}

// Qualys does not allow a Windows record's domain type to change after creation.
func TestWindowsDomainTypeForcesReplacement(t *testing.T) {
	s := resourceWindowsAuthRecord().Schema["domain_type"]
	if s == nil {
		t.Fatal("domain_type is not defined on the Windows record")
	}
	if !s.ForceNew {
		t.Error("domain_type must be ForceNew: Qualys fixes it at creation")
	}
}

// The Unix resource must not carry Windows-only attributes.
func TestUnixRecordHasNoWindowsAttributes(t *testing.T) {
	if _, present := resourceUnixAuthRecord().Schema["domain_type"]; present {
		t.Error("domain_type belongs to Windows records only")
	}
}

// unknownValue is the sentinel the SDK uses for values not known until apply
// (hcl2shim.UnknownVariableValue, which is internal to the SDK).
const unknownValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

// password = random_password.x.result is unknown at first plan and reads as "".
// The credential check must wait for apply rather than failing a valid config.
func TestAuthRecordAllowsUnknownCredentialAtPlanTime(t *testing.T) {
	err := diffFor(t, resourceUnixAuthRecord(), map[string]interface{}{
		"title":    "linux-scan",
		"username": "scanner",
		"password": unknownValue,
	})
	if err != nil {
		t.Fatalf("a computed password must not fail the plan: %v", err)
	}
}

// TestAuthRecordReadRefreshesVaultType is a regression test: vault_type is
// genuinely returned by the list API (vmdr/auth.go's authRecordGeneric
// decodes VAULT>VAULT_TYPE) but Read's values map used to omit it — a
// record referencing a vault would never round-trip its vault_type into
// state, producing a permanent diff for anyone who set it explicitly.
func TestAuthRecordReadRefreshesVaultType(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<AUTH_UNIX_LIST_OUTPUT><RESPONSE><AUTH_UNIX_LIST>
		  <AUTH_UNIX_RECORD>
		    <ID>9001</ID><TITLE>linux-scan</TITLE><USERNAME>scanner</USERNAME>
		    <VAULT><VAULT_ID>500</VAULT_ID><VAULT_TYPE>cyberark_aim</VAULT_TYPE></VAULT>
		  </AUTH_UNIX_RECORD>
		</AUTH_UNIX_LIST></RESPONSE></AUTH_UNIX_LIST_OUTPUT>`)
	}))
	defer srv.Close()
	c, err := vmdr.NewClient(vmdr.Config{
		BaseURL: srv.URL, Username: "u", Password: "p", HTTPClient: srv.Client(), MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("vmdr.NewClient: %v", err)
	}

	r := resourceUnixAuthRecord()
	d := r.Data(&terraform.InstanceState{ID: "9001"})
	diags := r.ReadContext(context.Background(), d, &clients{vmdr: c})
	if diags.HasError() {
		t.Fatalf("Read: %v", diags)
	}
	if got := d.Get("vault_type").(string); got != "cyberark_aim" {
		t.Errorf("vault_type = %q, want %q", got, "cyberark_aim")
	}
	if got := d.Get("vault_id").(string); got != "500" {
		t.Errorf("vault_id = %q, want %q", got, "500")
	}
}
