package qps

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func TestGetPortalVersionFlattensTopLevelFields(t *testing.T) {
	var gotPath, gotMethod string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		fmt.Fprint(w, `{"PORTAL_VERSION":"8.24.0.0","WAS_VERSION":"9.10.0.0-1",
			"VM_VERSION":"11.5","module":{"nested":"value"}}`)
	}))
	defer srv.Close()

	pv, err := c.GetPortalVersion(context.Background())
	if err != nil {
		t.Fatalf("GetPortalVersion: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/qps/rest/portal/version" {
		t.Errorf("method/path = %s %s", gotMethod, gotPath)
	}
	if pv.Raw["PORTAL_VERSION"] != "8.24.0.0" || pv.Raw["WAS_VERSION"] != "9.10.0.0-1" {
		t.Errorf("raw = %v", pv.Raw)
	}
	// A nested object is not a bare JSON string, so it is kept as its own
	// JSON text rather than dropped.
	if pv.Raw["module"] == "" {
		t.Error("expected nested field to be preserved as raw JSON text")
	}
}

func TestGetPortalVersionReportsHTTPError(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "Forbidden")
	}))
	defer srv.Close()

	if _, err := c.GetPortalVersion(context.Background()); err == nil {
		t.Fatal("expected an error on HTTP 403")
	}
}
