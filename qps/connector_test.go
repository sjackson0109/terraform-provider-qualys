package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestCreateAWSConnectorSendsV3Envelope(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"data":[
		  {"AwsAssetDataConnector":{"id":1998546,"name":"dev-aws","connectorState":"QUEUED",
		   "arn":"arn:aws:iam::123456789012:role/qualys","externalId":"US1-1-2",
		   "allRegions":"true","disabled":"false","runFrequency":240,
		   "awsAccountId":"123456789012","qualysAwsAccountId":"205767712438",
		   "cloudviewUuid":"11111111-2222-3333-4444-555555555555","type":"AWS"}}]}}`)
	}))
	defer srv.Close()

	disabled := false
	conn, err := c.CreateAWSConnector(context.Background(), AWSConnectorInput{
		ConnectorBaseInput: ConnectorBaseInput{
			Name:              "dev-aws",
			RunFrequency:      240,
			Disabled:          &disabled,
			ActivationModules: []string{ActivationVM, ActivationCloudView},
			DefaultTagIDs:     []string{"42458382"},
		},
		ARN:        "arn:aws:iam::123456789012:role/qualys",
		ExternalID: "US1-1-2",
	})
	if err != nil {
		t.Fatalf("CreateAWSConnector: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/qps/rest/3.0/create/am/awsassetdataconnector" {
		t.Errorf("request = %s %s", gotMethod, gotPath)
	}

	// The request must nest under ServiceRequest > data > AwsAssetDataConnector
	// with collections wrapped in "set" (doc 12).
	sr, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sr["data"].(map[string]interface{})
	wire, _ := data["AwsAssetDataConnector"].(map[string]interface{})
	if wire == nil {
		t.Fatalf("request body missing ServiceRequest.data.AwsAssetDataConnector: %v", gotBody)
	}
	if wire["arn"] != "arn:aws:iam::123456789012:role/qualys" {
		t.Errorf("arn = %v", wire["arn"])
	}
	if d, ok := wire["disabled"].(bool); !ok || d {
		t.Errorf("disabled must be a real boolean false in requests, got %T %v",
			wire["disabled"], wire["disabled"])
	}
	activation, _ := wire["activation"].(map[string]interface{})
	if set, ok := activation["set"].(map[string]interface{}); !ok || set["ActivationModule"] == nil {
		t.Errorf("activation must nest under set.ActivationModule, got %v", wire["activation"])
	}
	tags, _ := wire["defaultTags"].(map[string]interface{})
	if set, ok := tags["set"].(map[string]interface{}); !ok || set["TagSimple"] == nil {
		t.Errorf("defaultTags must nest under set.TagSimple, got %v", wire["defaultTags"])
	}

	// The response's string-rendered booleans must decode.
	if conn.ID != "1998546" || !conn.AllRegions || conn.Disabled {
		t.Errorf("connector = %+v", conn)
	}
	if conn.CloudViewUUID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("cloudviewUuid = %q", conn.CloudViewUUID)
	}
}

func TestGetAzureConnectorDecodesAuthRecord(t *testing.T) {
	var gotPath, gotMethod string

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"data":[
		  {"AzureAssetDataConnector":{"id":77,"name":"dev-azure","connectorState":"FINISHED_SUCCESS",
		   "disabled":"false","runFrequency":240,"subscriptionName":"Dev Sub",
		   "authRecord":{"applicationId":"app-1","directoryId":"dir-1","subscriptionId":"sub-1"},
		   "cloudviewUuid":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","type":"AZURE"}}]}}`)
	}))
	defer srv.Close()

	conn, err := c.GetAzureConnector(context.Background(), "77")
	if err != nil {
		t.Fatalf("GetAzureConnector: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/qps/rest/3.0/get/am/azureassetdataconnector/77" {
		t.Errorf("request = %s %s", gotMethod, gotPath)
	}
	if conn.ApplicationID != "app-1" || conn.DirectoryID != "dir-1" || conn.SubscriptionID != "sub-1" {
		t.Errorf("authRecord fields = %+v", conn)
	}
	if conn.SubscriptionName != "Dev Sub" || conn.State != "FINISHED_SUCCESS" {
		t.Errorf("connector = %+v", conn)
	}
}

func TestAzureCreateCarriesAuthRecord(t *testing.T) {
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"data":[
		  {"AzureAssetDataConnector":{"id":78,"name":"dev-azure","authRecord":{}}}]}}`)
	}))
	defer srv.Close()

	_, err := c.CreateAzureConnector(context.Background(), AzureConnectorInput{
		ConnectorBaseInput: ConnectorBaseInput{Name: "dev-azure"},
		ApplicationID:      "app-1",
		DirectoryID:        "dir-1",
		SubscriptionID:     "sub-1",
		AuthenticationKey:  "s3cret",
	})
	if err != nil {
		t.Fatalf("CreateAzureConnector: %v", err)
	}

	sr, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sr["data"].(map[string]interface{})
	wire, _ := data["AzureAssetDataConnector"].(map[string]interface{})
	auth, _ := wire["authRecord"].(map[string]interface{})
	if auth["applicationId"] != "app-1" || auth["directoryId"] != "dir-1" ||
		auth["subscriptionId"] != "sub-1" || auth["authenticationKey"] != "s3cret" {
		t.Errorf("authRecord = %v", auth)
	}
}

func TestGCPCreateEmbedsKeyFileAsAuthRecord(t *testing.T) {
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"data":[
		  {"GcpAssetDataConnector":{"id":55,"name":"dev-gcp","connectorState":"QUEUED",
		   "authRecord":{"projectId":"my-project"}}}]}}`)
	}))
	defer srv.Close()

	key := `{"type":"service_account","project_id":"my-project","private_key":"-----BEGIN PRIVATE KEY-----x","client_email":"sa@my-project.iam.gserviceaccount.com"}`
	conn, err := c.CreateGCPConnector(context.Background(), GCPConnectorInput{
		ConnectorBaseInput: ConnectorBaseInput{Name: "dev-gcp", RunFrequency: 240},
		CredentialsJSON:    key,
	})
	if err != nil {
		t.Fatalf("CreateGCPConnector: %v", err)
	}

	sr, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sr["data"].(map[string]interface{})
	wire, _ := data["GcpAssetDataConnector"].(map[string]interface{})
	if wire["type"] != "GCP" {
		t.Errorf("type = %v; the GCP create API documents it as required", wire["type"])
	}
	// The key file must be embedded field-for-field, not re-mapped (its JSON
	// field names are the documented authRecord names).
	auth, _ := wire["authRecord"].(map[string]interface{})
	if auth["project_id"] != "my-project" || auth["client_email"] != "sa@my-project.iam.gserviceaccount.com" {
		t.Errorf("authRecord = %v", auth)
	}
	if conn.ProjectID != "my-project" {
		t.Errorf("ProjectID = %q; must fall back across projectId/project_id spellings", conn.ProjectID)
	}
}

func TestGCPCreateRejectsNonJSONCredentials(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the API must not be called with undecodable credentials")
	}))
	defer srv.Close()

	_, err := c.CreateGCPConnector(context.Background(), GCPConnectorInput{
		ConnectorBaseInput: ConnectorBaseInput{Name: "dev-gcp", RunFrequency: 240},
		CredentialsJSON:    "not-json",
	})
	if err == nil {
		t.Fatal("expected an error for non-JSON credentials")
	}
}

func TestUpdateReturnsIDOnlyAndDeleteUsesIDPath(t *testing.T) {
	var paths []string

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"data":[
		  {"AwsAssetDataConnector":{"id":1998546}}]}}`)
	}))
	defer srv.Close()

	if err := c.UpdateAWSConnector(context.Background(), "1998546", AWSConnectorInput{
		ConnectorBaseInput: ConnectorBaseInput{Name: "renamed"},
	}); err != nil {
		t.Fatalf("UpdateAWSConnector: %v", err)
	}
	if err := c.DeleteAWSConnector(context.Background(), "1998546"); err != nil {
		t.Fatalf("DeleteAWSConnector: %v", err)
	}

	want := []string{
		"POST /qps/rest/3.0/update/am/awsassetdataconnector/1998546",
		"POST /qps/rest/3.0/delete/am/awsassetdataconnector/1998546",
	}
	if len(paths) != 2 || paths[0] != want[0] || paths[1] != want[1] {
		t.Errorf("paths = %v, want %v", paths, want)
	}
}

func TestConnectorIDRejectsCloudViewUUIDs(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the API must not be called with a v1 UUID id")
	}))
	defer srv.Close()

	_, err := c.GetAWSConnector(context.Background(), "11111111-2222-3333-4444-555555555555")
	if err == nil {
		t.Fatal("expected a UUID-form id to be rejected before reaching the API")
	}
}

func TestFindByCloudViewUUIDMatchesClientSide(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/qps/rest/3.0/search/am/gcpassetdataconnector" {
			t.Errorf("path = %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":2,"hasMoreRecords":"false","data":[
		  {"GcpAssetDataConnector":{"id":54,"name":"other","cloudviewUuid":"99999999-0000-0000-0000-000000000000"}},
		  {"GcpAssetDataConnector":{"id":55,"name":"mine","cloudviewUuid":"AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"}}]}}`)
	}))
	defer srv.Close()

	// Case-insensitive: v1 state may carry either case.
	conn, err := c.FindGCPConnectorByCloudViewUUID(context.Background(),
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err != nil {
		t.Fatalf("FindGCPConnectorByCloudViewUUID: %v", err)
	}
	if conn.ID != "55" || conn.Name != "mine" {
		t.Errorf("connector = %+v", conn)
	}
}

func TestActivationDecodingToleratesDocumentedShapes(t *testing.T) {
	// The XML response nesting is confirmed but no JSON response sample shows
	// the collection, so every plausible rendering must decode (doc 12).
	cases := []string{
		`{"list":{"ActivationModule":["VM","SCA"]}}`,
		`{"list":["VM","SCA"]}`,
		`["VM","SCA"]`,
	}
	for _, raw := range cases {
		got, recognized := decodeActivationModules(json.RawMessage(raw))
		if !recognized {
			t.Errorf("decodeActivationModules(%s): recognized = false, want true", raw)
		}
		if len(got) != 2 || got[0] != "VM" || got[1] != "SCA" {
			t.Errorf("decodeActivationModules(%s) = %v", raw, got)
		}
	}

	got, recognized := decodeActivationModules(json.RawMessage(`{"unexpected":1}`))
	if recognized {
		t.Errorf("an unrecognised shape must report recognized=false, so callers don't "+
			"overwrite previously-known state with a guess of empty; got recognized=true, modules=%v", got)
	}
	if got != nil {
		t.Errorf("unrecognised shape should yield nil modules, got %v", got)
	}

	empty, recognized := decodeActivationModules(nil)
	if !recognized {
		t.Error("an absent activation key must be recognized as genuinely empty, not treated " +
			"as an unrecognized shape — a real server-side clear must be able to reach state")
	}
	if len(empty) != 0 {
		t.Errorf("empty input should yield no modules, got %v", empty)
	}
}

func TestTagRefDecodingDistinguishesEmptyFromUnrecognized(t *testing.T) {
	refs, recognized := decodeTagRefs(json.RawMessage(`{"list":{"TagSimple":[{"id":1,"name":"prod"}]}}`))
	if !recognized || len(refs) != 1 || refs[0].ID != "1" || refs[0].Name != "prod" {
		t.Errorf("decodeTagRefs(list.TagSimple) = %v, recognized=%v", refs, recognized)
	}

	empty, recognized := decodeTagRefs(nil)
	if !recognized || len(empty) != 0 {
		t.Errorf("an absent defaultTags key must decode as recognized+empty, got %v, recognized=%v",
			empty, recognized)
	}

	_, recognized = decodeTagRefs(json.RawMessage(`{"unexpected":1}`))
	if recognized {
		t.Error("an unrecognised shape must report recognized=false")
	}
}

// TestUpdateSendsEmptyActivationAndTagsToClear is a regression test for a
// bug where clearing activation/default_tag_ids back to empty in config
// never reached the API: the wire encoder only sent the field when
// len(...) > 0, so a JSON "omitempty" pointer field was left nil and the
// key was omitted from the request body entirely, leaving the server's
// previous value in place forever.
func TestUpdateSendsEmptyActivationAndTagsToClear(t *testing.T) {
	var gotBody map[string]interface{}

	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","count":1,"data":[
		  {"AwsAssetDataConnector":{"id":1998546}}]}}`)
	}))
	defer srv.Close()

	// Deliberately empty/nil: this is what connectorBaseInput produces when
	// the user has cleared activation/default_tag_ids in configuration.
	err := c.UpdateAWSConnector(context.Background(), "1998546", AWSConnectorInput{
		ConnectorBaseInput: ConnectorBaseInput{Name: "dev-aws"},
	})
	if err != nil {
		t.Fatalf("UpdateAWSConnector: %v", err)
	}

	sr, _ := gotBody["ServiceRequest"].(map[string]interface{})
	data, _ := sr["data"].(map[string]interface{})
	wire, _ := data["AwsAssetDataConnector"].(map[string]interface{})

	activation, ok := wire["activation"]
	if !ok {
		t.Fatal("activation key must be present in the update body (even empty) so the API " +
			"can distinguish 'clear this' from 'leave unchanged'; omitting it entirely was the bug")
	}
	activationMap, _ := activation.(map[string]interface{})
	set, _ := activationMap["set"].(map[string]interface{})
	modules, _ := set["ActivationModule"].([]interface{})
	if len(modules) != 0 {
		t.Errorf("ActivationModule = %v, want an empty (but present) list", modules)
	}

	defaultTags, ok := wire["defaultTags"]
	if !ok {
		t.Fatal("defaultTags key must be present in the update body (even empty), same reasoning")
	}
	tagsMap, _ := defaultTags.(map[string]interface{})
	tagSet, _ := tagsMap["set"].(map[string]interface{})
	tagSimple, _ := tagSet["TagSimple"].([]interface{})
	if len(tagSimple) != 0 {
		t.Errorf("TagSimple = %v, want an empty (but present) list", tagSimple)
	}
}
