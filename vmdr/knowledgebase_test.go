package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestGetKnowledgeBaseEntriesBatchesIntoOneRequest(t *testing.T) {
	var calls int
	var form url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = r.ParseForm()
		form = r.Form
		fmt.Fprint(w, `<KNOWLEDGE_BASE_VULN_LIST_OUTPUT><RESPONSE><VULN_LIST>
		  <VULN><QID>105015</QID><TITLE>SSL Weak Cipher</TITLE><CATEGORY>General remote services</CATEGORY>
		    <CVSS><BASE>4.3</BASE></CVSS><CVSS_V3><BASE>5.9</BASE></CVSS_V3>
		    <PCI_FLAG>1</PCI_FLAG><PATCHABLE>0</PATCHABLE></VULN>
		  <VULN><QID>38170</QID><TITLE>SSH Weak MAC</TITLE><CATEGORY>General remote services</CATEGORY>
		    <CVSS><BASE>2.6</BASE></CVSS><PCI_FLAG>0</PCI_FLAG><PATCHABLE>1</PATCHABLE></VULN>
		</VULN_LIST></RESPONSE></KNOWLEDGE_BASE_VULN_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	// 4,000 findings referencing only these 2 unique QIDs must still be a
	// single request — the caller is responsible for de-duplicating QIDs
	// before calling this, and this test asserts that when it does, exactly
	// one HTTP call happens no matter how many QIDs are passed in that call.
	entries, err := c.GetKnowledgeBaseEntries(context.Background(), []string{"105015", "38170"})
	if err != nil {
		t.Fatalf("GetKnowledgeBaseEntries: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want exactly 1 batched request", calls)
	}
	if form.Get("ids") != "105015,38170" {
		t.Errorf("ids = %q", form.Get("ids"))
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	byQID := map[string]*KBEntry{}
	for _, e := range entries {
		byQID[e.QID] = e
	}
	e := byQID["105015"]
	if e == nil || e.Title != "SSL Weak Cipher" || e.CVSSV2Base != 4.3 || e.CVSSV3Base != 5.9 || !e.PCIFlag || e.Patchable {
		t.Errorf("entry 105015 = %+v", e)
	}
	e2 := byQID["38170"]
	if e2 == nil || e2.CVSSV3Base != 0 || e2.PCIFlag || !e2.Patchable {
		t.Errorf("entry 38170 = %+v", e2)
	}
}

func TestGetKnowledgeBaseEntriesNoQIDsIsNoRequest(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent for an empty QID list")
	}))
	defer srv.Close()

	entries, err := c.GetKnowledgeBaseEntries(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetKnowledgeBaseEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entries = %v, want none", entries)
	}
}

// A QID with no KnowledgeBase entry (e.g. a custom, non-KB QID) is simply
// absent from the result, not an error and not a fabricated zero-value entry.
func TestGetKnowledgeBaseEntriesAbsentQIDIsOmitted(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<KNOWLEDGE_BASE_VULN_LIST_OUTPUT><RESPONSE><VULN_LIST>
		  <VULN><QID>105015</QID><TITLE>SSL Weak Cipher</TITLE></VULN>
		</VULN_LIST></RESPONSE></KNOWLEDGE_BASE_VULN_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	entries, err := c.GetKnowledgeBaseEntries(context.Background(), []string{"105015", "999999"})
	if err != nil {
		t.Fatalf("GetKnowledgeBaseEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (999999 has no KB entry)", len(entries))
	}
}

func TestGetKnowledgeBaseEntriesFollowsTruncation(t *testing.T) {
	var calls int
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = r.ParseForm()
		if r.Form.Get("id_min") == "" {
			fmt.Fprint(w, `<KNOWLEDGE_BASE_VULN_LIST_OUTPUT><RESPONSE><VULN_LIST>
			  <VULN><QID>1</QID><TITLE>First</TITLE></VULN>
			</VULN_LIST><WARNING><CODE>1980</CODE><TEXT>truncated</TEXT>
			  <URL>https://x/api/2.0/fo/knowledge_base/vuln/?action=list&amp;id_min=2</URL>
			</WARNING></RESPONSE></KNOWLEDGE_BASE_VULN_LIST_OUTPUT>`)
			return
		}
		fmt.Fprint(w, `<KNOWLEDGE_BASE_VULN_LIST_OUTPUT><RESPONSE><VULN_LIST>
		  <VULN><QID>2</QID><TITLE>Second</TITLE></VULN>
		</VULN_LIST></RESPONSE></KNOWLEDGE_BASE_VULN_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	entries, err := c.GetKnowledgeBaseEntries(context.Background(), []string{"1", "2"})
	if err != nil {
		t.Fatalf("GetKnowledgeBaseEntries: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries across both pages, want 2", len(entries))
	}
}
