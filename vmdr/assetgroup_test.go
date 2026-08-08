package vmdr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestCreateAssetGroupUsesBareParameterNames(t *testing.T) {
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>Asset Group Added Successfully</TEXT>
		  <ITEM_LIST><ITEM><KEY>ID</KEY><VALUE>4021975</VALUE></ITEM></ITEM_LIST>
		</RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	id, err := c.CreateAssetGroup(context.Background(), AssetGroupInput{
		Title:     "prod-web",
		NetworkID: "12345",
		IPs:       []string{"10.0.0.0/30"},
		Domains:   []string{"example.com"},
	})
	if err != nil {
		t.Fatalf("CreateAssetGroup: %v", err)
	}
	if id != "4021975" {
		t.Errorf("id = %q, want 4021975", id)
	}
	if form.Get("action") != "add" {
		t.Errorf("action = %q", form.Get("action"))
	}
	// add takes bare names.
	if form.Get("title") != "prod-web" {
		t.Errorf("title = %q; add must use the bare name", form.Get("title"))
	}
	if form.Get("set_title") != "" {
		t.Error("add must not send set_title; that is the edit spelling")
	}
	// CIDR is expanded on the wire.
	if got, want := form.Get("ips"), "10.0.0.0-10.0.0.3"; got != want {
		t.Errorf("ips = %q, want %q", got, want)
	}
	if form.Get("network_id") != "12345" {
		t.Errorf("network_id = %q", form.Get("network_id"))
	}
}

func TestUpdateAssetGroupIsAuthoritative(t *testing.T) {
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>Asset Group Updated Successfully</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	err := c.UpdateAssetGroup(context.Background(), "4021975", AssetGroupInput{
		Title: "prod-web",
		IPs:   []string{"10.0.0.1"},
	})
	if err != nil {
		t.Fatalf("UpdateAssetGroup: %v", err)
	}

	if form.Get("action") != "edit" {
		t.Errorf("action = %q", form.Get("action"))
	}
	// edit takes prefixed names.
	if form.Get("set_title") != "prod-web" {
		t.Errorf("set_title = %q; edit must use the prefixed name", form.Get("set_title"))
	}
	if form.Get("title") != "" {
		t.Error("edit must not send the bare title; that is the add spelling")
	}
	// Terraform is authoritative, so set_* is used rather than add_*/remove_*.
	if form.Get("add_ips") != "" || form.Get("remove_ips") != "" {
		t.Error("update must not use additive parameters")
	}
	if form.Get("set_ips") != "10.0.0.1" {
		t.Errorf("set_ips = %q", form.Get("set_ips"))
	}
	// An emptied list must be sent empty rather than omitted, or the server keeps
	// the old members and the resource never converges.
	if _, present := form["set_domains"]; !present {
		t.Error("set_domains must be sent even when empty so members can be cleared")
	}
	// The same applies to the CVSS scalars: omitting an empty one would keep the
	// old rating server-side while state records the cleared value.
	if _, present := form["set_cvss_enviro_cdp"]; !present {
		t.Error("set_cvss_enviro_cdp must be sent even when empty so it can be cleared")
	}
}

func TestUpdateAssetGroupDoesNotAttemptNetworkChange(t *testing.T) {
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	_ = c.UpdateAssetGroup(context.Background(), "1", AssetGroupInput{Title: "t", NetworkID: "999"})

	// The edit action exposes no set_network_id. Sending network_id would be
	// silently ignored at best, so the resource marks the field ForceNew instead.
	if form.Get("network_id") != "" || form.Get("set_network_id") != "" {
		t.Error("update must not attempt to change the network")
	}
}

func TestListAssetGroupsParsesAllAttributes(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<ASSET_GROUP_LIST_OUTPUT><RESPONSE>
		  <ASSET_GROUP_LIST>
		    <ASSET_GROUP>
		      <ID>4021975</ID>
		      <TITLE>prod-web</TITLE>
		      <OWNER_ID>11</OWNER_ID>
		      <UNIT_ID>0</UNIT_ID>
		      <NETWORK_IDS>0</NETWORK_IDS>
		      <BUSINESS_IMPACT>High</BUSINESS_IMPACT>
		      <CVSS><CDP>high</CDP><TD>medium</TD></CVSS>
		      <IP_SET><IP>10.0.0.5</IP><IP_RANGE>10.0.0.1-10.0.0.3</IP_RANGE></IP_SET>
		      <DOMAIN_LIST><DOMAIN>example.com</DOMAIN></DOMAIN_LIST>
		      <APPLIANCE_IDS>7,9</APPLIANCE_IDS>
		      <COMMENTS>web tier</COMMENTS>
		    </ASSET_GROUP>
		  </ASSET_GROUP_LIST>
		</RESPONSE></ASSET_GROUP_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	groups, err := c.ListAssetGroups(context.Background(), AssetGroupFilter{})
	if err != nil {
		t.Fatalf("ListAssetGroups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	g := groups[0]
	if g.ID != "4021975" || g.Title != "prod-web" {
		t.Errorf("identity wrong: %+v", g)
	}
	// IP and IP_RANGE are flattened and sorted into one normalised set.
	if len(g.IPs) != 2 || g.IPs[0] != "10.0.0.1-10.0.0.3" || g.IPs[1] != "10.0.0.5" {
		t.Errorf("IPs = %v", g.IPs)
	}
	if len(g.ApplianceIDs) != 2 {
		t.Errorf("ApplianceIDs = %v", g.ApplianceIDs)
	}
	if g.BusinessImpact != "High" || g.CVSSEnviroCDP != "high" {
		t.Errorf("attributes wrong: %+v", g)
	}
}

func TestGetAssetGroupReportsNotFoundOnEmptyList(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<ASSET_GROUP_LIST_OUTPUT><RESPONSE></RESPONSE></ASSET_GROUP_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	_, err := c.GetAssetGroup(context.Background(), "4021975")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateAssetGroupWithoutIDIsAnExplicitError(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>Asset Group Added Successfully</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	_, err := c.CreateAssetGroup(context.Background(), AssetGroupInput{Title: "t"})
	if err == nil {
		t.Fatal("expected an error when no ID is returned")
	}
	// The group may exist despite the missing ID; the message must not imply
	// nothing happened.
	if got := err.Error(); !strings.Contains(got, "import") {
		t.Errorf("error should tell the operator how to recover, got %q", got)
	}
}
