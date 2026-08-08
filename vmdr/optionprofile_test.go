package vmdr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

const optionProfileBody = `<OPTION_PROFILES>
  <OPTION_PROFILE>
    <BASIC_INFO>
      <ID>51451401</ID>
      <GROUP_NAME>web-tier-scan</GROUP_NAME>
      <GROUP_TYPE>user</GROUP_TYPE>
      <USER_ID>11</USER_ID>
      <IS_DEFAULT>0</IS_DEFAULT>
      <IS_GLOBAL>1</IS_GLOBAL>
      <UPDATE_DATE>2026-01-15T10:00:00Z</UPDATE_DATE>
    </BASIC_INFO>
    <SCAN>
      <PORTS>
        <TCP_PORTS><TCP_PORTS_TYPE>standard</TCP_PORTS_TYPE></TCP_PORTS>
        <UDP_PORTS><UDP_PORTS_TYPE>light</UDP_PORTS_TYPE></UDP_PORTS>
      </PORTS>
      <SCAN_DEAD_HOSTS>0</SCAN_DEAD_HOSTS>
      <PERFORMANCE><OVERALL_PERFORMANCE>normal</OVERALL_PERFORMANCE></PERFORMANCE>
      <VULNERABILITY_DETECTION>
        <DETECTION_TYPE>complete</DETECTION_TYPE>
        <BASIC_HOST_INFO_CHECKS>1</BASIC_HOST_INFO_CHECKS>
      </VULNERABILITY_DETECTION>
      <TEST_AUTHENTICATION>1</TEST_AUTHENTICATION>
    </SCAN>
  </OPTION_PROFILE>
</OPTION_PROFILES>`

func TestOptionProfileListDecodesDocumentedShape(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, optionProfileBody)
	}))
	defer srv.Close()

	p, err := c.GetVMOptionProfile(context.Background(), "51451401")
	if err != nil {
		t.Fatalf("GetVMOptionProfile: %v", err)
	}
	if p.Title != "web-tier-scan" {
		t.Errorf("Title = %q", p.Title)
	}
	if !p.Global || p.Default {
		t.Errorf("Global=%v Default=%v, want true/false", p.Global, p.Default)
	}
	if p.ScanTCPPorts != "standard" || p.ScanUDPPorts != "light" {
		t.Errorf("ports = %q/%q", p.ScanTCPPorts, p.ScanUDPPorts)
	}
	if p.VulnerabilityDetection != "complete" {
		t.Errorf("VulnerabilityDetection = %q", p.VulnerabilityDetection)
	}
	if !p.TestAuthentication || !p.BasicHostInformationChecks {
		t.Errorf("booleans not decoded: %+v", p)
	}
}

// If the response nests profiles under RESPONSE, decoding must still work.
// Getting this wrong would make every read report ErrNotFound, which would drop
// a live resource from state immediately after it was created.
func TestOptionProfileListDecodesNestedShape(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<OPTION_PROFILES><RESPONSE>
		  <OPTION_PROFILE><BASIC_INFO><ID>51451401</ID>
		    <GROUP_NAME>web-tier-scan</GROUP_NAME></BASIC_INFO></OPTION_PROFILE>
		</RESPONSE></OPTION_PROFILES>`)
	}))
	defer srv.Close()

	p, err := c.GetVMOptionProfile(context.Background(), "51451401")
	if err != nil {
		t.Fatalf("a nested response should still decode: %v", err)
	}
	if p.Title != "web-tier-scan" {
		t.Errorf("Title = %q", p.Title)
	}
}

func TestGetVMOptionProfileNotFound(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<OPTION_PROFILES></OPTION_PROFILES>`)
	}))
	defer srv.Close()

	if _, err := c.GetVMOptionProfile(context.Background(), "999"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// Qualys silently falls back to complete detection when a custom selection
// cannot be resolved, so the client refuses to send that combination.
func TestCustomDetectionRequiresSearchLists(t *testing.T) {
	_, err := optionProfileParams(VMOptionProfileInput{
		Title: "x", VulnerabilityDetection: "custom",
	})
	if err == nil {
		t.Fatal("expected an error for custom detection with no search lists")
	}
}

func TestOptionProfileCreateSendsFieldLevelParams(t *testing.T) {
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT>
		  <ITEM_LIST><ITEM><KEY>ID</KEY><VALUE>51451401</VALUE></ITEM></ITEM_LIST>
		</RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	_, err := c.CreateVMOptionProfile(context.Background(), VMOptionProfileInput{
		Title:                  "web",
		VulnerabilityDetection: "custom",
		CustomSearchListIDs:    []string{"123", "456"},
		Authentication:         []string{"Windows", "Unix"},
		ScanTCPPorts:           "full",
	})
	if err != nil {
		t.Fatalf("CreateVMOptionProfile: %v", err)
	}

	if form.Get("scan_tcp_ports") != "full" {
		t.Errorf("scan_tcp_ports = %q", form.Get("scan_tcp_ports"))
	}
	if form.Get("custom_search_list_ids") != "123,456" {
		t.Errorf("custom_search_list_ids = %q", form.Get("custom_search_list_ids"))
	}
	// Authentication is a list of technologies, not per-technology booleans.
	if form.Get("authentication") != "Windows,Unix" {
		t.Errorf("authentication = %q", form.Get("authentication"))
	}
}
