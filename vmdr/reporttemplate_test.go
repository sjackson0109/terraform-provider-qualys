package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestListReportTemplatesUsesLegacyEndpoint(t *testing.T) {
	var gotPath string
	var gotForm url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = r.ParseForm()
		gotForm = r.Form
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8" ?>
<REPORT_TEMPLATE_LIST>
  <REPORT_TEMPLATE>
    <ID>235288</ID>
    <TYPE>Auto</TYPE>
    <TEMPLATE_TYPE>Scan</TEMPLATE_TYPE>
    <TITLE><![CDATA[Windows Authentication QIDs]]></TITLE>
    <USER>
      <LOGIN><![CDATA[acme_jk]]></LOGIN>
      <FIRSTNAME><![CDATA[Jason]]></FIRSTNAME>
      <LASTNAME><![CDATA[Kim]]></LASTNAME>
    </USER>
    <LAST_UPDATE>2018-02-12T18:09:10Z</LAST_UPDATE>
    <GLOBAL>0</GLOBAL>
  </REPORT_TEMPLATE>
</REPORT_TEMPLATE_LIST>`)
	}))
	defer srv.Close()

	templates, err := c.ListReportTemplates(context.Background())
	if err != nil {
		t.Fatalf("ListReportTemplates: %v", err)
	}
	// The legacy script lives outside the versioned /api/<version>/fo/ family.
	if gotPath != "/msp/report_template_list.php" {
		t.Errorf("path = %q, want /msp/report_template_list.php", gotPath)
	}
	if len(gotForm) != 0 {
		t.Errorf("form = %v; the script takes no filter parameters", gotForm)
	}
	if len(templates) != 1 {
		t.Fatalf("got %d templates, want 1", len(templates))
	}
	tpl := templates[0]
	if tpl.ID != "235288" || tpl.TemplateType != "Scan" || tpl.Title != "Windows Authentication QIDs" {
		t.Errorf("template = %+v", tpl)
	}
	if tpl.OwnerLogin != "acme_jk" || tpl.OwnerName != "Jason Kim" {
		t.Errorf("owner = %q / %q", tpl.OwnerLogin, tpl.OwnerName)
	}
	if tpl.Global {
		t.Error("GLOBAL=0 should decode to false")
	}
}

func TestListReportTemplatesSurfacesErrors(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `Unauthorized`)
	}))
	defer srv.Close()

	if _, err := c.ListReportTemplates(context.Background()); err == nil {
		t.Fatal("expected an error for a non-2xx response")
	}
}
