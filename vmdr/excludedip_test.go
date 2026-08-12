package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestAddExcludedIPsSendsConfirmedParams(t *testing.T) {
	var gotAction string
	var gotForm url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotAction = r.Form.Get("action")
		gotForm = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><DATETIME>2026-01-01T00:00:00Z</DATETIME>
			<TEXT>OK</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	err := c.AddExcludedIPs(context.Background(), AddExcludedIPsInput{
		IPs:             []string{"10.0.0.1", "10.0.0.2"},
		Comment:         "quarantine",
		ExpiryDays:      30,
		AssetGroupNames: []string{"Prod"},
		NetworkID:       "42",
	})
	if err != nil {
		t.Fatalf("AddExcludedIPs: %v", err)
	}
	if gotAction != "add" {
		t.Errorf("action = %q, want add", gotAction)
	}
	if gotForm.Get("comment") != "quarantine" || gotForm.Get("expiry_days") != "30" ||
		gotForm.Get("dg_names") != "Prod" || gotForm.Get("network_id") != "42" {
		t.Errorf("form = %v", gotForm)
	}
	if gotForm.Get("ips") == "" {
		t.Error("ips param was not sent")
	}
}

func TestAddExcludedIPsRequiresAtLeastOneIP(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent with no IPs")
	}))
	defer srv.Close()

	if err := c.AddExcludedIPs(context.Background(), AddExcludedIPsInput{}); err == nil {
		t.Fatal("expected an error with no IPs")
	}
}

func TestRemoveExcludedIPsSendsRemoveAction(t *testing.T) {
	var gotAction string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotAction = r.Form.Get("action")
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>OK</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	if err := c.RemoveExcludedIPs(context.Background(), []string{"10.0.0.1"}, "", ""); err != nil {
		t.Fatalf("RemoveExcludedIPs: %v", err)
	}
	if gotAction != "remove" {
		t.Errorf("action = %q, want remove", gotAction)
	}
}

func TestRemoveAllExcludedIPsOmitsIPsParam(t *testing.T) {
	var gotAction string
	var sawIPs bool
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotAction = r.Form.Get("action")
		_, sawIPs = r.Form["ips"]
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>OK</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	if err := c.RemoveAllExcludedIPs(context.Background(), "clearing all", "42"); err != nil {
		t.Fatalf("RemoveAllExcludedIPs: %v", err)
	}
	if gotAction != "remove_all" {
		t.Errorf("action = %q, want remove_all", gotAction)
	}
	if sawIPs {
		// Confirmed: "ips is invalid for remove_all".
		t.Error("ips param was sent on remove_all, which the API documents as invalid for this action")
	}
}
