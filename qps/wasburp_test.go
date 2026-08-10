package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestImportWASBurpSendsFlatDataBody(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	err := c.ImportWASBurp(context.Background(), WASBurpImport{
		WebAppID:              "1524084",
		PurgeResults:          false,
		CloseUnreportedIssues: true,
		FileName:              "testBurpReportImport",
		BurpXML:               "<burp/>",
	})
	if err != nil {
		t.Fatalf("ImportWASBurp: %v", err)
	}
	if gotPath != "/qps/rest/3.0/import/was/burp" {
		t.Errorf("path = %q", gotPath)
	}

	sreq, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sreq["data"].(map[string]interface{})
	if data["webAppId"] != "1524084" || data["closeUnreportedIssues"] != true ||
		data["purgeResults"] != false || data["fileName"] != "testBurpReportImport" || data["burpXml"] != "<burp/>" {
		t.Errorf("data = %v (must be flat, not nested under a Wrapper object)", data)
	}
	if _, present := data["Finding"]; present {
		t.Errorf("data should not be wrapped in an object key: %v", data)
	}
}

func TestImportWASBurpRequiresWebAppIDAndXML(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent for an invalid import")
	}))
	defer srv.Close()

	if err := c.ImportWASBurp(context.Background(), WASBurpImport{BurpXML: "<burp/>"}); err == nil {
		t.Fatal("expected an error for a missing web app id")
	}
	if err := c.ImportWASBurp(context.Background(), WASBurpImport{WebAppID: "1"}); err == nil {
		t.Fatal("expected an error for missing Burp XML content")
	}
}
