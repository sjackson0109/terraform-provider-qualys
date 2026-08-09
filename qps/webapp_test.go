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

func TestCreateWebAppSendsTagSetIdiom(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WebApp":{"id":555,"name":"storefront","url":"https://shop.example.com"}}]}}`)
	}))
	defer srv.Close()

	app, err := c.CreateWebApp(context.Background(), WebAppInput{
		Name: "storefront", URL: "https://shop.example.com", TagIDs: []string{"10", "20"},
	})
	if err != nil {
		t.Fatalf("CreateWebApp: %v", err)
	}
	if app.ID != "555" || app.URL != "https://shop.example.com" {
		t.Errorf("app = %+v", app)
	}
	if gotPath != "/qps/rest/3.0/create/was/webapp" {
		t.Errorf("path = %q", gotPath)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	webapp, _ := data["WebApp"].(map[string]interface{})
	tags, _ := webapp["tags"].(map[string]interface{})
	set, _ := tags["set"].(map[string]interface{})
	simples, _ := set["TagSimple"].([]interface{})
	if len(simples) != 2 {
		t.Fatalf("expected 2 tags sent via the set idiom, got %v", tags)
	}
}

func TestWebAppNotFoundIsRecognised(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
	}))
	defer srv.Close()

	_, err := c.GetWebApp(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// A create must not be re-sent after a transport failure: the server may have
// processed it, and a retry would duplicate the web application.
func TestCreateWebAppIsNotRetriedOnTransportError(t *testing.T) {
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

	if _, err := c.CreateWebApp(context.Background(), WebAppInput{Name: "x", URL: "https://x.example.com"}); err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("create was sent %d times; a lost response must not cause a re-send", calls)
	}
}

func TestSearchWebAppsDecodesListShape(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"false",
		  "data":[{"WebApp":{"id":1,"name":"a"}},{"WebApp":{"id":2,"name":"b"}}]}}`)
	}))
	defer srv.Close()

	apps, err := c.SearchWebApps(context.Background(), nil)
	if err != nil {
		t.Fatalf("SearchWebApps: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("got %d web applications, want 2", len(apps))
	}
}

func TestUpdateWebAppAuthRecordAssociationsSendsAddAndRemove(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	if err := c.UpdateWebAppAuthRecordAssociations(context.Background(), "555", []string{"10"}, []string{"20"}); err != nil {
		t.Fatalf("UpdateWebAppAuthRecordAssociations: %v", err)
	}
	if gotPath != "/qps/rest/3.0/update/was/webapp/555" {
		t.Errorf("path = %q", gotPath)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	webapp, _ := data["WebApp"].(map[string]interface{})
	authRecords, _ := webapp["authRecords"].(map[string]interface{})
	if authRecords == nil {
		t.Fatalf("no authRecords in payload: %v", webapp)
	}
	add, _ := authRecords["add"].(map[string]interface{})
	addList, _ := add["WebAppAuthRecord"].([]interface{})
	if len(addList) != 1 {
		t.Errorf("expected 1 added ref, got %v", add)
	}
	remove, _ := authRecords["remove"].(map[string]interface{})
	removeList, _ := remove["WebAppAuthRecord"].([]interface{})
	if len(removeList) != 1 {
		t.Errorf("expected 1 removed ref, got %v", remove)
	}
	if webapp["tags"] != nil || webapp["name"] != nil {
		t.Errorf("association update must not touch name/url/tags: %v", webapp)
	}
}

func TestUpdateWebAppAuthRecordAssociationsNoopWithNothingToDo(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent with nothing to add or remove")
	}))
	defer srv.Close()

	if err := c.UpdateWebAppAuthRecordAssociations(context.Background(), "555", nil, nil); err != nil {
		t.Fatalf("UpdateWebAppAuthRecordAssociations: %v", err)
	}
}
