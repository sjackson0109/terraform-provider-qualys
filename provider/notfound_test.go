package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
)

// qpsNotFoundServer returns a TLS test server whose every response is a QPS
// OBJECT_NOT_FOUND error — standing in for both "the object was deleted" and
// "the object is outside the caller's scope", which the real API does not
// distinguish either (qps.ErrNotFound's doc comment) — plus the meta value a
// resource's Read expects.
func qpsNotFoundServer(t *testing.T) (*httptest.Server, interface{}) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
	}))
	c, err := qps.NewClient(qps.Config{
		BaseURL:    srv.URL,
		Username:   "u",
		Password:   "p",
		HTTPClient: srv.Client(),
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("qps.NewClient: %v", err)
	}
	return srv, &clients{qps: c}
}

// TestQPSNotFoundOnReadPreservesStateAndErrors tests the shared policy
// function directly: on an ambiguous OBJECT_NOT_FOUND, it must return an
// Error-severity diagnostic (a Warning alone does not stop Terraform from
// planning a recreate) telling the operator how to confirm deletion
// explicitly, and it must never touch the resource's id itself.
func TestQPSNotFoundOnReadPreservesStateAndErrors(t *testing.T) {
	d := &idOnly{id: "12345"}
	diags := qpsNotFoundOnRead(d, "asset tag")

	if d.id != "12345" {
		t.Errorf("qpsNotFoundOnRead must never clear the resource id itself; id = %q", d.id)
	}
	if len(diags) != 1 {
		t.Fatalf("expected exactly one diagnostic, got %d", len(diags))
	}
	if diags[0].Severity != diag.Error {
		t.Errorf("severity = %v, want diag.Error — a Warning does not stop Terraform from "+
			"planning a recreate, which is exactly what this policy must prevent", diags[0].Severity)
	}
	if !strings.Contains(diags[0].Detail, "terraform state rm") {
		t.Errorf("detail should tell the operator how to confirm deletion explicitly: %q", diags[0].Detail)
	}
	if !strings.Contains(diags[0].Detail, "12345") {
		t.Errorf("detail should name the affected id: %q", diags[0].Detail)
	}

	// The safe sequence must be explicit and ordered: check scope, then
	// independently verify, and only then state rm — never state rm as the
	// first or only suggested action, which would reintroduce the
	// duplicate-object risk this whole policy exists to prevent.
	detail := diags[0].Detail
	scopeIdx := strings.Index(detail, "Check scope")
	verifyIdx := strings.Index(detail, "independently verify")
	rmIdx := strings.Index(detail, "terraform state rm")
	if scopeIdx == -1 || verifyIdx == -1 || rmIdx == -1 {
		t.Fatalf("detail must instruct checking scope AND independently verifying AND "+
			"(only then) state rm; got %q", detail)
	}
	if !(scopeIdx < verifyIdx && verifyIdx < rmIdx) {
		t.Errorf("the three steps must appear in order (scope check, then independent "+
			"verification, then state rm as the last resort); got scope=%d verify=%d rm=%d",
			scopeIdx, verifyIdx, rmIdx)
	}
	if !strings.Contains(detail, "last resort") {
		t.Error("detail should explicitly frame state rm as a last resort, not a default response")
	}
}

type idOnly struct{ id string }

func (f *idOnly) Id() string { return f.id }

// readAgainstNotFound builds a ResourceData bound to id, runs r's ReadContext
// against a server that always returns OBJECT_NOT_FOUND, and returns the
// resulting diagnostics and the id left in state afterward.
func readAgainstNotFound(t *testing.T, r *schema.Resource, id string) (diag.Diagnostics, string) {
	t.Helper()
	srv, meta := qpsNotFoundServer(t)
	defer srv.Close()

	d := r.Data(&terraform.InstanceState{ID: id})
	diags := r.ReadContext(context.Background(), d, meta)
	return diags, d.Id()
}

// TestConnectorResourcesPreserveStateOnAmbiguousNotFound exercises the full
// SDK Read path (not just the shared helper) for a representative resource
// from each affected group: a plain object (asset tag), an assignment
// resource whose Read goes through a search rather than a direct get (host
// tag assignment), and one of the three Connector v3 resources.
func TestConnectorResourcesPreserveStateOnAmbiguousNotFound(t *testing.T) {
	cases := []struct {
		name string
		r    *schema.Resource
	}{
		{"asset tag", resourceAssetTag()},
		{"host tag assignment", resourceHostTagAssignment()},
		{"aws connector", resourceAWSConnector()},
		{"azure connector", resourceAzureConnector()},
		{"gcp connector", resourceGCPConnector()},
		{"was application", resourceWASApplication()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const id = "999"
			diags, gotID := readAgainstNotFound(t, tc.r, id)

			if gotID != id {
				t.Errorf("%s: id changed from %q to %q on an ambiguous not-found — "+
					"Terraform will plan an unintended recreate", tc.name, id, gotID)
			}
			if !diags.HasError() {
				t.Errorf("%s: expected an error diagnostic on ambiguous not-found, got %v", tc.name, diags)
			}
		})
	}
}

// TestDeleteIsIdempotentOnNotFound confirms Delete's opposite, correct
// policy: a not-found response during Delete means the desired final state
// (the object is gone, from this caller's perspective) has already been
// achieved, so it must succeed rather than error.
func TestDeleteIsIdempotentOnNotFound(t *testing.T) {
	cases := []struct {
		name string
		r    *schema.Resource
	}{
		{"asset tag", resourceAssetTag()},
		{"aws connector", resourceAWSConnector()},
		{"azure connector", resourceAzureConnector()},
		{"gcp connector", resourceGCPConnector()},
		{"was application", resourceWASApplication()},
		{"was auth record", resourceWASAuthRecord()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, meta := qpsNotFoundServer(t)
			defer srv.Close()

			d := tc.r.Data(&terraform.InstanceState{ID: "999"})
			diags := tc.r.DeleteContext(context.Background(), d, meta)
			if diags.HasError() {
				t.Errorf("%s: Delete against an already-absent object must be idempotent, got %v",
					tc.name, diags)
			}
		})
	}
}
