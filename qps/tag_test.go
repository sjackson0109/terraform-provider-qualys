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
		fmt.Fprint(w, `{"responseCode":"SUCCESS","count":1,
		  "data":[{"Tag":{"id":12345,"name":"prod","ruleType":"STATIC"}}]}`)
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
	if _, ok := gotBody["data"]; !ok {
		t.Errorf("request body missing data envelope: %v", gotBody)
	}
}

func TestObjectNotFoundIsRecognised(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"responseCode":"OBJECT_NOT_FOUND",
		  "responseErrorDetails":{"errorMessage":"No Tag found for id 999"}}`)
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
		fmt.Fprint(w, `{"responseCode":"UNAUTHORIZED"}`)
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
		var req ServiceRequest
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &req)

		if pages == 1 {
			fmt.Fprint(w, `{"responseCode":"SUCCESS","count":1,"hasMoreRecords":"true","lastId":100,
			  "data":[{"Tag":{"id":100,"name":"a"}}]}`)
			return
		}
		if req.Filters != nil {
			secondCriteria = req.Filters.Criteria
		}
		fmt.Fprint(w, `{"responseCode":"SUCCESS","count":1,"hasMoreRecords":"false",
		  "data":[{"Tag":{"id":101,"name":"b"}}]}`)
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
		fmt.Fprint(w, `{"responseCode":"SUCCESS","count":1,"hasMoreRecords":"true","lastId":1,
		  "data":[{"Tag":{"id":1,"name":"a"}}]}`)
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
		fmt.Fprint(w, `{"responseCode":"SUCCESS","data":[{"Tag":{"id":5,"name":"root","parentTagId":0}}]}`)
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
