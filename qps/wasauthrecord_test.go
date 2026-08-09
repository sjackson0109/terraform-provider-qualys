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

func TestCreateWASAuthRecordSendsFormFieldsUnderSetIdiom(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WebAppAuthRecord":{"id":77,"name":"storefront-login"}}]}}`)
	}))
	defer srv.Close()

	rec, err := c.CreateWASAuthRecord(context.Background(), WASAuthRecordInput{
		Name: "storefront-login",
		Form: &WASFormRecord{
			SubType: "STANDARD",
			Fields: []WASAuthField{
				{Name: "username", Value: "scanner"},
				{Name: "password", Value: "s3cret", Secured: true},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateWASAuthRecord: %v", err)
	}
	if rec.ID != "77" {
		t.Errorf("rec = %+v", rec)
	}
	if gotPath != "/qps/rest/3.0/create/was/webappauthrecord" {
		t.Errorf("path = %q", gotPath)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	record, _ := data["WebAppAuthRecord"].(map[string]interface{})
	formRecord, _ := record["formRecord"].(map[string]interface{})
	if formRecord == nil {
		t.Fatalf("no formRecord in payload: %v", record)
	}
	fields, _ := formRecord["fields"].(map[string]interface{})
	set, _ := fields["set"].(map[string]interface{})
	entries, _ := set["WebAppAuthFormRecordField"].([]interface{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 fields sent via the set idiom, got %v", fields)
	}
}

func TestCreateWASAuthRecordEncodesServerCredentialsAsFields(t *testing.T) {
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WebAppAuthRecord":{"id":78,"name":"basic-auth"}}]}}`)
	}))
	defer srv.Close()

	_, err := c.CreateWASAuthRecord(context.Background(), WASAuthRecordInput{
		Name:   "basic-auth",
		Server: &WASServerRecord{Username: "scanner", Password: "s3cret", Domain: "CORP"},
	})
	if err != nil {
		t.Fatalf("CreateWASAuthRecord: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	record, _ := data["WebAppAuthRecord"].(map[string]interface{})
	serverRecord, _ := record["serverRecord"].(map[string]interface{})
	if serverRecord == nil {
		t.Fatalf("no serverRecord in payload: %v", record)
	}
	fields, _ := serverRecord["fields"].(map[string]interface{})
	set, _ := fields["set"].(map[string]interface{})
	entries, _ := set["WebAppAuthFormRecordField"].([]interface{})
	if len(entries) != 3 {
		t.Fatalf("expected username, domain and password fields, got %v", entries)
	}
}

// CreateWASAuthRecord must reject a request with neither record type: it has
// no credential the crawler could use, and the API accepting it silently
// would only surface as unauthenticated scan results much later.
func TestCreateWASAuthRecordRequiresARecordType(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent without a record type")
	}))
	defer srv.Close()

	if _, err := c.CreateWASAuthRecord(context.Background(), WASAuthRecordInput{Name: "x"}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestWASAuthRecordNotFoundIsRecognised(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
	}))
	defer srv.Close()

	_, err := c.GetWASAuthRecord(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// A create must not be re-sent after a transport failure: the server may have
// processed it, and a retry would duplicate the record.
func TestCreateWASAuthRecordIsNotRetriedOnTransportError(t *testing.T) {
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

	in := WASAuthRecordInput{Name: "x", Server: &WASServerRecord{Username: "u", Password: "p"}}
	if _, err := c.CreateWASAuthRecord(context.Background(), in); err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("create was sent %d times; a lost response must not cause a re-send", calls)
	}
}

func TestSearchWASAuthRecordsDecodesListShape(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"false",
		  "data":[{"WebAppAuthRecord":{"id":1,"name":"a"}},{"WebAppAuthRecord":{"id":2,"name":"b"}}]}}`)
	}))
	defer srv.Close()

	recs, err := c.SearchWASAuthRecords(context.Background(), nil)
	if err != nil {
		t.Fatalf("SearchWASAuthRecords: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
}
