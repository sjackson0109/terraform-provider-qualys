package qps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestCreateWASDNSOverrideSendsMappingsUnderSetIdiom(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"DnsOverride":{"id":57020,"name":"Customer Production DNS"}}]}}`)
	}))
	defer srv.Close()

	override, err := c.CreateWASDNSOverride(context.Background(), WASDNSOverrideInput{
		Name: "Customer Production DNS",
		Mappings: []WASDNSMapping{
			{HostName: "portal.customer.com", IPAddress: "10.100.5.20"},
			{HostName: "api.customer.com", IPAddress: "10.100.5.21"},
		},
	})
	if err != nil {
		t.Fatalf("CreateWASDNSOverride: %v", err)
	}
	if override.ID != "57020" {
		t.Errorf("override = %+v", override)
	}
	if gotPath != "/qps/rest/3.0/create/was/dnsoverride" {
		t.Errorf("path = %q", gotPath)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	dnsOverride, _ := data["DnsOverride"].(map[string]interface{})
	if dnsOverride == nil {
		t.Fatalf("no DnsOverride in payload: %v", data)
	}
	mappings, _ := dnsOverride["mappings"].(map[string]interface{})
	set, _ := mappings["set"].(map[string]interface{})
	entries, _ := set["DnsMapping"].([]interface{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 mappings sent via the set idiom, got %v", mappings)
	}
}

func TestCreateWASDNSOverrideRequiresAMapping(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent without a mapping")
	}))
	defer srv.Close()

	if _, err := c.CreateWASDNSOverride(context.Background(), WASDNSOverrideInput{Name: "x"}); err == nil {
		t.Fatal("expected an error")
	}
}

// The update endpoint takes no ID in the URL path — a real deviation from
// every other WAS object this package models. The ID must travel inside
// the request body instead.
func TestUpdateWASDNSOverrideSendsIDInBodyNotPath(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	if err := c.UpdateWASDNSOverride(context.Background(), "57020", WASDNSOverrideInput{
		Name:     "Customer Production DNS",
		Mappings: []WASDNSMapping{{HostName: "portal.customer.com", IPAddress: "10.100.5.30"}},
	}); err != nil {
		t.Fatalf("UpdateWASDNSOverride: %v", err)
	}
	if gotPath != "/qps/rest/3.0/update/was/dnsoverride" {
		t.Errorf("path = %q; update must not append an id segment", gotPath)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	dnsOverride, _ := data["DnsOverride"].(map[string]interface{})
	if dnsOverride["id"] != float64(57020) {
		t.Errorf("data.DnsOverride.id = %v, want the id carried in the body", dnsOverride["id"])
	}
}

// The delete endpoint likewise takes no ID in the URL path — it travels as
// a filter criterion instead.
func TestDeleteWASDNSOverrideSendsIDAsFilterNotPath(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	if err := c.DeleteWASDNSOverride(context.Background(), "57020"); err != nil {
		t.Fatalf("DeleteWASDNSOverride: %v", err)
	}
	if gotPath != "/qps/rest/3.0/delete/was/dnsoverride" {
		t.Errorf("path = %q; delete must not append an id segment", gotPath)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	filters, _ := sreq["filters"].(map[string]interface{})
	criteria, _ := filters["Criteria"].([]interface{})
	if len(criteria) != 1 {
		t.Fatalf("expected 1 filter criterion (id), got %v", criteria)
	}
	crit, _ := criteria[0].(map[string]interface{})
	if crit["field"] != "id" || crit["value"] != "57020" {
		t.Errorf("criterion = %v", crit)
	}
}

func TestWASDNSOverrideNotFoundIsRecognised(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
	}))
	defer srv.Close()

	_, err := c.GetWASDNSOverride(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateWASDNSOverrideIsNotRetriedOnTransportError(t *testing.T) {
	var calls int
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("test server does not support hijacking")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		conn.Close()
	}))
	defer srv.Close()

	in := WASDNSOverrideInput{Name: "x", Mappings: []WASDNSMapping{{HostName: "a.example.com", IPAddress: "10.0.0.1"}}}
	if _, err := c.CreateWASDNSOverride(context.Background(), in); err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("create was sent %d times; a lost response must not cause a re-send", calls)
	}
}

func TestSearchWASDNSOverridesDecodesListShape(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"false",
		  "data":[{"DnsOverride":{"id":1,"name":"a"}},{"DnsOverride":{"id":2,"name":"b"}}]}}`)
	}))
	defer srv.Close()

	overrides, err := c.SearchWASDNSOverrides(context.Background(), nil)
	if err != nil {
		t.Fatalf("SearchWASDNSOverrides: %v", err)
	}
	if len(overrides) != 2 {
		t.Fatalf("got %d overrides, want 2", len(overrides))
	}
}
