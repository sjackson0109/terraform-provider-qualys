package provider

import (
	"strings"
	"testing"
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
