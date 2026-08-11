package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestUpdateHostAssetTagsSendsAddAndRemove(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	if err := c.UpdateHostAssetTags(context.Background(), "42", []string{"1", "2"}, []string{"3"}); err != nil {
		t.Fatalf("UpdateHostAssetTags: %v", err)
	}
	if gotPath != "/qps/rest/2.0/update/am/hostasset/42" {
		t.Errorf("path = %q", gotPath)
	}

	sr, ok := gotBody["ServiceRequest"].(map[string]interface{})
	if !ok {
		t.Fatalf("body = %v", gotBody)
	}
	data, ok := sr["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data = %v", sr)
	}
	hostAsset, ok := data["HostAsset"].(map[string]interface{})
	if !ok {
		t.Fatalf("HostAsset = %v", data)
	}
	tags, ok := hostAsset["tags"].(map[string]interface{})
	if !ok {
		t.Fatalf("tags = %v", hostAsset)
	}
	if _, ok := tags["add"]; !ok {
		t.Error("expected tags.add to be sent")
	}
	if _, ok := tags["remove"]; !ok {
		t.Error("expected tags.remove to be sent")
	}
}

func TestUpdateHostAssetTagsNoOpWhenNothingChanges(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent when add and remove are both empty")
	}))
	defer srv.Close()

	if err := c.UpdateHostAssetTags(context.Background(), "42", nil, nil); err != nil {
		t.Fatalf("UpdateHostAssetTags: %v", err)
	}
}

func TestSearchHostAssetsDecodesTags(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,
			"hasMoreRecords":"false","data":[{"HostAsset":{"id":42,"name":"host1",
			"dnsHostName":"host1.example.com","trackingMethod":"IP",
			"tags":{"list":{"TagSimple":[{"id":7,"name":"Prod"}]}}}}]}}`)
	}))
	defer srv.Close()

	assets, err := c.SearchHostAssets(context.Background(), &Filters{Criteria: []Criterion{
		{Field: "tagName", Operator: "EQUALS", Value: "Prod"},
	}})
	if err != nil {
		t.Fatalf("SearchHostAssets: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("got %d assets, want 1", len(assets))
	}
	a := assets[0]
	if a.ID != "42" || a.Name != "host1" || a.DNSHostName != "host1.example.com" ||
		a.TrackingMethod != "IP" || len(a.Tags) != 1 || a.Tags[0].ID != "7" || a.Tags[0].Name != "Prod" {
		t.Errorf("asset = %+v", a)
	}
}
