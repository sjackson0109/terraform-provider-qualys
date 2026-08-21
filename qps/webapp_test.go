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

func TestCreateWebAppSendsAttributesAndDNSOverrides(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WebApp":{"id":556,"name":"storefront"}}]}}`)
	}))
	defer srv.Close()

	_, err := c.CreateWebApp(context.Background(), WebAppInput{
		Name: "storefront", URL: "https://shop.example.com",
		Attributes:           []WebAppAttribute{{Name: "Custom key 1", Value: "Custom value 1"}},
		DNSOverrideIDs:       []string{"2022"},
		DefaultDNSOverrideID: "2022",
	})
	if err != nil {
		t.Fatalf("CreateWebApp: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	webapp, _ := data["WebApp"].(map[string]interface{})

	attrs, _ := webapp["attributes"].(map[string]interface{})
	attrSet, _ := attrs["set"].(map[string]interface{})
	attrList, _ := attrSet["Attribute"].([]interface{})
	if len(attrList) != 1 {
		t.Fatalf("expected 1 attribute sent via the set idiom, got %v", attrs)
	}
	first, _ := attrList[0].(map[string]interface{})
	if first["name"] != "Custom key 1" || first["value"] != "Custom value 1" {
		t.Errorf("attribute = %v", first)
	}

	dnsOverrides, _ := webapp["dnsOverrides"].(map[string]interface{})
	dnsSet, _ := dnsOverrides["set"].(map[string]interface{})
	dnsList, _ := dnsSet["DnsOverride"].([]interface{})
	if len(dnsList) != 1 {
		t.Fatalf("expected 1 dnsOverride sent via the set idiom, got %v", dnsOverrides)
	}

	config, _ := webapp["config"].(map[string]interface{})
	defaultDNS, _ := config["defaultDnsOverride"].(map[string]interface{})
	if defaultDNS["id"] != float64(2022) {
		t.Errorf("config.defaultDnsOverride.id = %v", defaultDNS["id"])
	}
	if _, present := config["cancelScansAt"]; present {
		t.Errorf("cancelScansAt should be omitted when not set: %v", config)
	}
}

// A user-supplied "Gap Review" document documents a landmine this package
// reproduces faithfully: config is only sent when at least one of its three
// governed fields is set, because Qualys clears the cancellation setting if
// config is present without a cancellation element.
func TestUpdateWebAppOmitsConfigWhenNoConfigFieldsSet(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	if err := c.UpdateWebApp(context.Background(), "555", WebAppInput{Name: "storefront"}); err != nil {
		t.Fatalf("UpdateWebApp: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	webapp, _ := data["WebApp"].(map[string]interface{})
	if _, present := webapp["config"]; present {
		t.Errorf("config should be omitted entirely when no config fields are set: %v", webapp["config"])
	}
}

func TestCreateWebAppSendsSwaggerFile(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WebApp":{"id":557,"name":"api"}}]}}`)
	}))
	defer srv.Close()

	_, err := c.CreateWebApp(context.Background(), WebAppInput{
		Name: "api", URL: "https://api.example.com",
		SwaggerFile: &WebAppFile{Name: "openapi.yml", Content: "QkFTRTY0"},
	})
	if err != nil {
		t.Fatalf("CreateWebApp: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	webapp, _ := data["WebApp"].(map[string]interface{})
	swagger, _ := webapp["swaggerFile"].(map[string]interface{})
	if swagger["name"] != "openapi.yml" || swagger["content"] != "QkFTRTY0" {
		t.Errorf("swaggerFile = %v", swagger)
	}
	if _, present := webapp["postmanCollection"]; present {
		t.Errorf("swaggerFile and postmanCollection are mutually exclusive: %v", webapp)
	}
}

func TestGetWebAppDecodesCrawlingScriptsAndMalwareFlags(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"WebApp":{"id":558,"name":"storefront","malwareMonitoring":true,
		  "malwareNotification":false,
		  "crawlingScripts":{"count":1,"list":{"SeleniumScript":[{"id":2500,
		  "name":"name of the Script","data":"...","requiresAuthentication":true,
		  "startingUrl":"https://example.com/","startingUrlRegex":true}]}}}}]}}`)
	}))
	defer srv.Close()

	app, err := c.GetWebApp(context.Background(), "558")
	if err != nil {
		t.Fatalf("GetWebApp: %v", err)
	}
	if !app.MalwareMonitoring || app.MalwareNotification {
		t.Errorf("malware flags = monitoring:%v notification:%v", app.MalwareMonitoring, app.MalwareNotification)
	}
	if len(app.CrawlingScripts) != 1 {
		t.Fatalf("got %d crawling scripts, want 1", len(app.CrawlingScripts))
	}
	s := app.CrawlingScripts[0]
	if s.ID != "2500" || s.Name != "name of the Script" || !s.RequiresAuthentication ||
		s.StartingURL != "https://example.com/" || !s.StartingURLRegex {
		t.Errorf("crawling script = %+v", s)
	}
}

// TestUpdateWebAppSendsFalseMalwareFlagsAndEmptyAttributes is a regression
// test: MalwareMonitoring/MalwareNotification were only ever sent as `true`
// (guarded by `if in.MalwareMonitoring`), so setting either back to false
// in configuration produced a nil pointer that omitempty dropped from the
// request entirely — the server's previous (enabled) value was left in
// place forever. Attributes had the identical bug for the whole collection
// (`if len(in.Attributes) > 0`): clearing every attribute in configuration
// could never reach the API either.
func TestUpdateWebAppSendsFalseMalwareFlagsAndEmptyAttributes(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,
		  "data":[{"WebApp":{"id":558}}]}}`)
	}))
	defer srv.Close()

	err := c.UpdateWebApp(context.Background(), "558", WebAppInput{
		Name: "storefront", URL: "https://shop.example.com",
		MalwareMonitoring:   false,
		MalwareNotification: false,
		// Attributes deliberately nil/empty: this is "clear every
		// attribute", not "leave attributes alone".
	})
	if err != nil {
		t.Fatalf("UpdateWebApp: %v", err)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	webapp, _ := data["WebApp"].(map[string]interface{})

	monitoring, present := webapp["malwareMonitoring"]
	if !present {
		t.Fatal("malwareMonitoring key must be present (as false) so the API can " +
			"distinguish 'turn this off' from 'leave unchanged'; omitting it was the bug")
	}
	if monitoring != false {
		t.Errorf("malwareMonitoring = %v, want false", monitoring)
	}
	notification, present := webapp["malwareNotification"]
	if !present || notification != false {
		t.Errorf("malwareNotification = %v (present=%v), want false", notification, present)
	}

	attrs, present := webapp["attributes"]
	if !present {
		t.Fatal("attributes key must be present (even empty) so the API can " +
			"distinguish 'clear every attribute' from 'leave unchanged'")
	}
	attrsMap, _ := attrs.(map[string]interface{})
	attrSet, _ := attrsMap["set"].(map[string]interface{})
	attrList, _ := attrSet["Attribute"].([]interface{})
	if len(attrList) != 0 {
		t.Errorf("Attribute = %v, want an empty (but present) list", attrList)
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
