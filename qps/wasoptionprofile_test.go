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

func TestCreateWASOptionProfileSendsJSONEnvelope(t *testing.T) {
	var gotPath string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS",
		  "data":[{"OptionProfile":{"id":42,"name":"Standard","performance":"MEDIUM"}}]}}`)
	}))
	defer srv.Close()

	profile, err := c.CreateWASOptionProfile(context.Background(), WASOptionProfileInput{
		Name: "Standard", Performance: WASPerformanceMedium,
	})
	if err != nil {
		t.Fatalf("CreateWASOptionProfile: %v", err)
	}
	if profile.ID != "42" || profile.Performance != "MEDIUM" {
		t.Errorf("profile = %+v", profile)
	}
	if gotPath != "/qps/rest/3.0/create/was/optionprofile" {
		t.Errorf("path = %q", gotPath)
	}
}

// Update must send threshold and sensitive-content fields even at their zero
// value, or clearing them in configuration would never reach Qualys and the
// resource would diff forever — the same class of bug fixed for asset-group
// CVSS fields.
func TestUpdateWASOptionProfileSendsClearedFields(t *testing.T) {
	var body map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	if err := c.UpdateWASOptionProfile(context.Background(), "42", WASOptionProfileInput{
		Name: "Standard", TimeoutErrorThreshold: 0, DetectCreditCardNumbers: false,
	}); err != nil {
		t.Fatalf("UpdateWASOptionProfile: %v", err)
	}

	sreq, _ := body["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	profile, _ := data["OptionProfile"].(map[string]interface{})
	if profile == nil {
		t.Fatalf("no WasOptionProfile in payload: %v", body)
	}
	if _, present := profile["timeoutErrorThreshold"]; !present {
		t.Error("timeoutErrorThreshold was omitted from the update; the old value would survive server-side")
	}
	sensitive, _ := profile["sensitiveContent"].(map[string]interface{})
	if sensitive == nil {
		t.Fatal("sensitiveContent was omitted from the update")
	}
	if _, present := sensitive["creditCardNumber"]; !present {
		t.Error("creditCardNumber was omitted from the update")
	}
}

func TestWASOptionProfileNotFoundIsRecognised(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"OBJECT_NOT_FOUND"}}`)
	}))
	defer srv.Close()

	_, err := c.GetWASOptionProfile(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateWASOptionProfileIsNotRetriedOnTransportError(t *testing.T) {
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

	if _, err := c.CreateWASOptionProfile(context.Background(), WASOptionProfileInput{Name: "x"}); err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("create was sent %d times; a lost response must not cause a re-send", calls)
	}
}
