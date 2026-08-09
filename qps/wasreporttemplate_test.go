package qps

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestSearchWASReportTemplatesDecodesListShape(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"false",
		  "data":[{"ReportTemplate":{"id":876048,"name":"Executive Summary","type":"WAS_WEBAPP_REPORT"}}]}}`)
	}))
	defer srv.Close()

	templates, err := c.SearchWASReportTemplates(context.Background(), nil)
	if err != nil {
		t.Fatalf("SearchWASReportTemplates: %v", err)
	}
	if len(templates) != 1 || templates[0].ID != "876048" || templates[0].Type != "WAS_WEBAPP_REPORT" {
		t.Errorf("templates = %+v", templates)
	}
}

func TestWASReportTemplateNotFoundIsRecognised(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
	}))
	defer srv.Close()

	_, err := c.GetWASReportTemplate(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
