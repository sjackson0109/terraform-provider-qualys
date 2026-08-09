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

// A "Gap Review" document, quoting the official Qualys STANDARD create
// example verbatim, corrected this package's earlier assumption that
// STANDARD credentials are sent as flat username/password elements: they go
// through the same generic fields/set/WebAppAuthFormRecordField list as
// CUSTOM records.
func TestCreateWASAuthRecordSendsStandardFormCredentialsThroughFieldsList(t *testing.T) {
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
			SubType:  WASAuthFormStandard,
			LoginURL: "https://shop.example.com/login",
			SSLOnly:  true,
			Username: "scanner",
			Password: "s3cret",
		},
	})
	if err != nil {
		t.Fatalf("CreateWASAuthRecord: %v", err)
	}
	if rec.ID != "77" {
		t.Errorf("rec = %+v", rec)
	}
	if gotPath != "/qps/rest/3.0/create/was/webauthrecord" {
		t.Errorf("path = %q", gotPath)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	record, _ := data["WebAppAuthRecord"].(map[string]interface{})
	formRecord, _ := record["formRecord"].(map[string]interface{})
	if formRecord == nil {
		t.Fatalf("no formRecord in payload: %v", record)
	}
	if _, present := formRecord["username"]; present {
		t.Errorf("a STANDARD record must not send a flat username element: %v", formRecord)
	}
	if _, present := formRecord["password"]; present {
		t.Errorf("a STANDARD record must not send a flat password element: %v", formRecord)
	}
	if formRecord["loginUrl"] != "https://shop.example.com/login" {
		t.Errorf("loginUrl should remain a flat element: %v", formRecord)
	}
	fields, _ := formRecord["fields"].(map[string]interface{})
	set, _ := fields["set"].(map[string]interface{})
	entries, _ := set["WebAppAuthFormRecordField"].([]interface{})
	if len(entries) != 2 {
		t.Fatalf("expected username and password sent via the fields/set list, got %v", formRecord["fields"])
	}
	first, _ := entries[0].(map[string]interface{})
	if first["name"] != "username" || first["value"] != "scanner" {
		t.Errorf("first field = %v", first)
	}
	second, _ := entries[1].(map[string]interface{})
	if second["name"] != "password" || second["value"] != "s3cret" || second["secured"] != true {
		t.Errorf("second field = %v", second)
	}
}

func TestCreateWASAuthRecordSendsSeleniumScript(t *testing.T) {
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WebAppAuthRecord":{"id":81,"name":"selenium-login"}}]}}`)
	}))
	defer srv.Close()

	_, err := c.CreateWASAuthRecord(context.Background(), WASAuthRecordInput{
		Name: "selenium-login",
		Form: &WASFormRecord{
			SubType: WASAuthFormSelenium,
			SeleniumScript: &WASSeleniumScript{
				Name:  "seleniumScriptOK",
				Data:  "<selenium-ide-script/>",
				Regex: "selenium",
			},
			SeleniumCreds: true,
			Username:      "scanner",
			Password:      "s3cret",
		},
	})
	if err != nil {
		t.Fatalf("CreateWASAuthRecord: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	record, _ := data["WebAppAuthRecord"].(map[string]interface{})
	formRecord, _ := record["formRecord"].(map[string]interface{})
	if formRecord["type"] != "SELENIUM" {
		t.Errorf("type = %v", formRecord["type"])
	}
	script, _ := formRecord["seleniumScript"].(map[string]interface{})
	if script["name"] != "seleniumScriptOK" || script["regex"] != "selenium" {
		t.Errorf("seleniumScript = %v", script)
	}
	if formRecord["seleniumCreds"] != true {
		t.Errorf("seleniumCreds = %v", formRecord["seleniumCreds"])
	}
}

func TestCreateWASAuthRecordSendsOAuth2Record(t *testing.T) {
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WebAppAuthRecord":{"id":82,"name":"oauth2-client-creds"}}]}}`)
	}))
	defer srv.Close()

	_, err := c.CreateWASAuthRecord(context.Background(), WASAuthRecordInput{
		Name: "oauth2-client-creds",
		OAuth2: &WASOAuth2Record{
			GrantType:      WASOAuth2GrantClientCreds,
			AccessTokenURL: "https://auth.example.com/oauth/token",
			ClientID:       "client-id",
			ClientSecret:   "client-secret",
			Scope:          "scope",
		},
	})
	if err != nil {
		t.Fatalf("CreateWASAuthRecord: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	record, _ := data["WebAppAuthRecord"].(map[string]interface{})
	if _, present := record["formRecord"]; present {
		t.Errorf("an OAuth2-only record must not also send formRecord: %v", record)
	}
	oauth2, _ := record["oauth2Record"].(map[string]interface{})
	if oauth2 == nil {
		t.Fatalf("no oauth2Record in payload: %v", record)
	}
	if oauth2["grantType"] != "CLIENT_CREDS" || oauth2["accessTokenUrl"] != "https://auth.example.com/oauth/token" ||
		oauth2["clientId"] != "client-id" || oauth2["clientSecret"] != "client-secret" || oauth2["scope"] != "scope" {
		t.Errorf("oauth2Record = %v", oauth2)
	}
}

func TestCreateWASAuthRecordSendsCustomFormFieldsUnderSetIdiom(t *testing.T) {
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WebAppAuthRecord":{"id":79,"name":"custom-login"}}]}}`)
	}))
	defer srv.Close()

	_, err := c.CreateWASAuthRecord(context.Background(), WASAuthRecordInput{
		Name: "custom-login",
		Form: &WASFormRecord{
			SubType: WASAuthFormCustom,
			Fields: []WASAuthField{
				{Name: "user", Value: "scanner"},
				{Name: "pass", Value: "s3cret", Secured: true},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateWASAuthRecord: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	record, _ := data["WebAppAuthRecord"].(map[string]interface{})
	formRecord, _ := record["formRecord"].(map[string]interface{})
	fields, _ := formRecord["fields"].(map[string]interface{})
	set, _ := fields["set"].(map[string]interface{})
	entries, _ := set["WebAppAuthFormRecordField"].([]interface{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 fields sent via the set idiom, got %v", fields)
	}
}

func TestCreateWASAuthRecordSendsServerCredentialsAsFlatFields(t *testing.T) {
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
	if serverRecord["username"] != "scanner" || serverRecord["password"] != "s3cret" ||
		serverRecord["domain"] != "CORP" {
		t.Errorf("serverRecord did not send flat credential elements: %v", serverRecord)
	}
	if _, present := serverRecord["fields"]; present {
		t.Errorf("a server record must not send the generic fields list: %v", serverRecord)
	}
}

func TestCreateWASAuthRecordSendsComments(t *testing.T) {
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WebAppAuthRecord":{"id":80,"name":"x","comments":"Created via API"}}]}}`)
	}))
	defer srv.Close()

	rec, err := c.CreateWASAuthRecord(context.Background(), WASAuthRecordInput{
		Name:     "x",
		Comments: "Created via API",
		Server:   &WASServerRecord{Username: "u", Password: "p"},
	})
	if err != nil {
		t.Fatalf("CreateWASAuthRecord: %v", err)
	}
	if rec.Comments != "Created via API" {
		t.Errorf("rec.Comments = %q", rec.Comments)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	record, _ := data["WebAppAuthRecord"].(map[string]interface{})
	if record["comments"] != "Created via API" {
		t.Errorf("comments = %v", record["comments"])
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
