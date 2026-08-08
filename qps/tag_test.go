package qps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(t *testing.T, h http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	c, err := NewClient(Config{
		BaseURL:    srv.URL,
		Username:   "u",
		Password:   "p",
		HTTPClient: srv.Client(),
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, srv
}

func TestCreateTagSendsJSONEnvelope(t *testing.T) {
	var gotPath, gotAccept, gotContentType string
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,
		  "data":[{"Tag":{"id":12345,"name":"prod","ruleType":"STATIC"}}]}}`)
	}))
	defer srv.Close()

	tag, err := c.CreateTag(context.Background(), TagInput{Name: "prod", RuleType: RuleTypeStatic})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	if tag.ID != "12345" || tag.Name != "prod" {
		t.Errorf("tag = %+v", tag)
	}
	if gotPath != "/qps/rest/2.0/create/am/tag" {
		t.Errorf("path = %q", gotPath)
	}
	// JSON only applies when both headers are set; the API defaults to XML.
	if gotAccept != "application/json" || gotContentType != "application/json" {
		t.Errorf("Accept=%q Content-Type=%q; both are required for JSON", gotAccept, gotContentType)
	}
	// The portal API requires the request nested under a ServiceRequest key;
	// a bare {"data": ...} payload is ignored by the real service.
	sreq, ok := gotBody["ServiceRequest"].(map[string]interface{})
	if !ok {
		t.Fatalf("request body missing the ServiceRequest wrapper: %v", gotBody)
	}
	if _, ok := sreq["data"]; !ok {
		t.Errorf("ServiceRequest missing data envelope: %v", sreq)
	}
}

func TestObjectNotFoundIsRecognised(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND",
		  "responseErrorDetails":{"errorMessage":"No Tag found for id 999"}}}`)
	}))
	defer srv.Close()

	_, err := c.GetTag(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUnauthorizedIsRecognised(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"UNAUTHORIZED"}}`)
	}))
	defer srv.Close()

	_, err := c.GetTag(context.Background(), "1")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestSearchFollowsIDCursor(t *testing.T) {
	var pages int
	var secondCriteria []Criterion

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		var wrapped struct {
			ServiceRequest ServiceRequest `json:"ServiceRequest"`
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &wrapped)
		req := wrapped.ServiceRequest

		if pages == 1 {
			fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"hasMoreRecords":"true","lastId":100,
			  "data":[{"Tag":{"id":100,"name":"a"}}]}}`)
			return
		}
		if req.Filters != nil {
			secondCriteria = req.Filters.Criteria
		}
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"hasMoreRecords":"false",
		  "data":[{"Tag":{"id":101,"name":"b"}}]}}`)
	}))
	defer srv.Close()

	tags, err := c.SearchTags(context.Background(), &Filters{
		Criteria: []Criterion{{Field: "name", Operator: "CONTAINS", Value: "x"}},
	})
	if err != nil {
		t.Fatalf("SearchTags: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(tags))
	}
	if pages != 2 {
		t.Errorf("pages = %d, want 2", pages)
	}

	// The cursor is added, and the caller's own criteria survive paging.
	var sawCursor, sawOriginal bool
	for _, cr := range secondCriteria {
		if cr.Field == "id" && cr.Operator == "GREATER" && cr.Value == "100" {
			sawCursor = true
		}
		if cr.Field == "name" {
			sawOriginal = true
		}
	}
	if !sawCursor {
		t.Errorf("second page did not carry the id cursor: %+v", secondCriteria)
	}
	if !sawOriginal {
		t.Errorf("second page dropped the caller's criteria: %+v", secondCriteria)
	}
}

func TestSearchRefusesPartialResultsOnRunaway(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"hasMoreRecords":"true","lastId":1,
		  "data":[{"Tag":{"id":1,"name":"a"}}]}}`)
	}))
	defer srv.Close()

	err := c.SearchAll(context.Background(), "/qps/rest/2.0/search/am/tag", nil, 100, 3,
		func(json.RawMessage) error { return nil })
	if err == nil {
		t.Fatal("expected an error rather than silently truncated results")
	}
}

// A root tag's parent comes back as 0; that must present as unset so it matches
// a configuration that omits the attribute.
func TestRootTagParentNormalisesToEmpty(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","data":[{"Tag":{"id":5,"name":"root","parentTagId":0}}]}}`)
	}))
	defer srv.Close()

	tag, err := c.GetTag(context.Background(), "5")
	if err != nil {
		t.Fatalf("GetTag: %v", err)
	}
	if tag.Parent != "" {
		t.Errorf("Parent = %q, want empty for a root tag", tag.Parent)
	}
}

func TestNewClientRejectsPlaintextAndMissingBaseURL(t *testing.T) {
	if _, err := NewClient(Config{Username: "u", Password: "p"}); err == nil {
		t.Error("expected an error with no BaseURL")
	}
	if _, err := NewClient(Config{BaseURL: "http://x", Username: "u", Password: "p"}); err == nil {
		t.Error("expected an error for a plaintext BaseURL")
	}
}

// A create must not be re-sent after a transport failure: the server may have
// processed it, and a retry would duplicate the tag.
func TestCreateTagIsNotRetriedOnTransportError(t *testing.T) {
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

	if _, err := c.CreateTag(context.Background(), TagInput{Name: "prod"}); err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("create was sent %d times; a lost response must not cause a re-send", calls)
	}
}

// Update must send the fields it manages even when empty, or clearing an
// attribute in configuration never reaches Qualys and the resource diffs forever.
func TestUpdateTagSendsClearedFields(t *testing.T) {
	var body map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	if err := c.UpdateTag(context.Background(), "12345", TagInput{Name: "prod", Color: ""}); err != nil {
		t.Fatalf("UpdateTag: %v", err)
	}

	sreq, _ := body["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	tag, _ := data["Tag"].(map[string]interface{})
	if tag == nil {
		t.Fatalf("no Tag in payload: %v", body)
	}
	if _, present := tag["color"]; !present {
		t.Error("color was omitted from the update; the old value would survive server-side")
	}
	// ruleType is the exception: an empty rule type is not meaningful and the API
	// rejects it, so it is omitted rather than sent blank.
	if _, present := tag["ruleType"]; present {
		t.Error("an empty ruleType should be omitted, not sent")
	}
}

// The documented response shape nests under ServiceResponse; the bare shape is
// tolerated too, so a service speaking either way decodes rather than failing
// with an empty responseCode.
func TestBareResponseShapeStillDecodes(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"responseCode":"SUCCESS","data":[{"Tag":{"id":7,"name":"x"}}]}`)
	}))
	defer srv.Close()

	tag, err := c.GetTag(context.Background(), "7")
	if err != nil {
		t.Fatalf("GetTag: %v", err)
	}
	if tag.ID != "7" {
		t.Errorf("ID = %q", tag.ID)
	}
}

// GET and body-less POST calls must still send Content-Type: the portal API
// only answers in JSON when both negotiation headers are present.
func TestBodylessCallsStillNegotiateJSON(t *testing.T) {
	var gotContentType string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","data":[{"Tag":{"id":7,"name":"x"}}]}}`)
	}))
	defer srv.Close()

	if _, err := c.GetTag(context.Background(), "7"); err != nil {
		t.Fatalf("GetTag: %v", err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q on a GET; the response would come back as XML", gotContentType)
	}
}
