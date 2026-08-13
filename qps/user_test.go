package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestGetUserDecodesMultiItemScopeAndRoles(t *testing.T) {
	var gotPath, gotMethod string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","data":[{"User":{
			"id":12345678,"username":"jdoe","firstName":"John","lastName":"Doe",
			"emailAddress":"jdoe@example.com","title":"Software Engineer",
			"scopeTags":{"list":[{"TagData":{"id":1,"name":"Production"}},{"TagData":{"id":2,"name":"Development"}}]},
			"roleList":{"list":[{"RoleData":{"id":1,"name":"Role1"}},{"RoleData":{"id":2,"name":"Role2"}}]}
			}}]}}`)
	}))
	defer srv.Close()

	u, err := c.GetUser(context.Background(), "12345678")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/qps/rest/2.0/get/am/user/12345678" {
		t.Errorf("method/path = %s %s", gotMethod, gotPath)
	}
	if u.Username != "jdoe" || u.FirstName != "John" || u.EmailAddress != "jdoe@example.com" {
		t.Errorf("user = %+v", u)
	}
	if len(u.ScopeTags) != 2 || u.ScopeTags[0].Name != "Production" && u.ScopeTags[1].Name != "Production" {
		t.Errorf("scope tags = %+v", u.ScopeTags)
	}
	if len(u.Roles) != 2 {
		t.Errorf("roles = %+v", u.Roles)
	}
}

func TestGetUserDecodesSingleBareScopeTag(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","data":[{"User":{
			"id":1,"username":"solo","scopeTags":{"id":9,"name":"OnlyOne"}
			}}]}}`)
	}))
	defer srv.Close()

	u, err := c.GetUser(context.Background(), "1")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if len(u.ScopeTags) != 1 || u.ScopeTags[0].ID != "9" || u.ScopeTags[0].Name != "OnlyOne" {
		t.Errorf("scope tags = %+v", u.ScopeTags)
	}
}

func TestGetUserNotFound(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","data":[]}}`)
	}))
	defer srv.Close()

	if _, err := c.GetUser(context.Background(), "999"); err == nil {
		t.Fatal("expected an error for an empty result")
	}
}

func TestSearchUsersSendsCriteria(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS","hasMoreRecords":"false","data":[
			{"User":{"id":1,"username":"jdoe"}}]}}`)
	}))
	defer srv.Close()

	users, err := c.SearchUsers(context.Background(), &Filters{Criteria: []Criterion{
		{Field: "username", Operator: "EQUALS", Value: "jdoe"},
	}})
	if err != nil {
		t.Fatalf("SearchUsers: %v", err)
	}
	if len(users) != 1 || users[0].Username != "jdoe" {
		t.Errorf("users = %+v", users)
	}

	sr, ok := gotBody["ServiceRequest"].(map[string]interface{})
	if !ok {
		t.Fatalf("body = %v", gotBody)
	}
	if _, ok := sr["filters"]; !ok {
		t.Errorf("expected filters in request body: %v", sr)
	}
}

func TestUpdateUserScopeSendsAddAndRemove(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		fmt.Fprint(w, `{"ServiceResponse":{"responseCode":"SUCCESS"}}`)
	}))
	defer srv.Close()

	err := c.UpdateUserScope(context.Background(), "42", UserScopeUpdate{
		AddTagNames:     []string{"New Tag"},
		RemoveRoleNames: []string{"Old Role"},
	})
	if err != nil {
		t.Fatalf("UpdateUserScope: %v", err)
	}
	if gotPath != "/qps/rest/2.0/update/am/user/42" {
		t.Errorf("path = %q", gotPath)
	}

	sr := gotBody["ServiceRequest"].(map[string]interface{})
	data := sr["data"].(map[string]interface{})
	user := data["User"].(map[string]interface{})
	scopeTags := user["scopeTags"].(map[string]interface{})
	if _, ok := scopeTags["add"]; !ok {
		t.Error("expected scopeTags.add")
	}
	roleList := user["roleList"].(map[string]interface{})
	if _, ok := roleList["remove"]; !ok {
		t.Error("expected roleList.remove")
	}
}

func TestUpdateUserScopeRequiresAtLeastOneChange(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent with no changes")
	}))
	defer srv.Close()

	if err := c.UpdateUserScope(context.Background(), "42", UserScopeUpdate{}); err == nil {
		t.Fatal("expected an error with no add/remove entries")
	}
}
