package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func TestListDomainsDecodesNetblockRanges(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8" ?>
<DOMAIN_LIST>
  <DOMAIN>
    <DOMAIN_NAME>qualysguard.com</DOMAIN_NAME>
    <DOMAIN_ID>47943018</DOMAIN_ID>
    <NETWORK>
      <NETWORK_NAME>Global Default Network</NETWORK_NAME>
      <NETWORK_ID>0</NETWORK_ID>
    </NETWORK>
    <NETBLOCK>
      <RANGE>
        <START>10.10.10.10</START>
        <END>20.20.20.20</END>
      </RANGE>
    </NETBLOCK>
  </DOMAIN>
</DOMAIN_LIST>`)
	}))
	defer srv.Close()

	domains, err := c.ListDomains(context.Background())
	if err != nil {
		t.Fatalf("ListDomains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("got %d domains, want 1", len(domains))
	}
	d := domains[0]
	if d.ID != "47943018" || d.Name != "qualysguard.com" || d.NetworkID != "0" {
		t.Errorf("domain = %+v", d)
	}
	if len(d.Netblocks) != 1 || d.Netblocks[0] != "10.10.10.10-20.20.20.20" {
		t.Errorf("netblocks = %v", d.Netblocks)
	}
}

func TestListDomainsSendsNoFilterParams(t *testing.T) {
	var gotQuery string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		fmt.Fprint(w, `<DOMAIN_LIST></DOMAIN_LIST>`)
	}))
	defer srv.Close()

	if _, err := c.ListDomains(context.Background()); err != nil {
		t.Fatalf("ListDomains: %v", err)
	}
	if gotQuery != "action=list" {
		t.Errorf("query = %q; no filter parameters are confirmed for this endpoint", gotQuery)
	}
}
