package vmdr

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeXMLAcceptsDoctypeWithoutFetchingDTD(t *testing.T) {
	// A real VMDR response: DOCTYPE present, pointing at a remote DTD. The DTD
	// host here is a test server that records whether it was ever contacted.
	var dtdFetched bool
	dtdServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dtdFetched = true
		w.WriteHeader(http.StatusOK)
	}))
	defer dtdServer.Close()

	body := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE SIMPLE_RETURN SYSTEM "` + dtdServer.URL + `/api/2.0/simple_return.dtd">
<SIMPLE_RETURN>
  <RESPONSE>
    <DATETIME>2026-08-07T09:00:00Z</DATETIME>
    <TEXT>Asset Group Deleted Successfully</TEXT>
  </RESPONSE>
</SIMPLE_RETURN>`

	var got simpleReturn
	if err := decodeXMLBytes([]byte(body), &got); err != nil {
		t.Fatalf("decode failed on a valid response: %v", err)
	}
	if dtdFetched {
		t.Fatal("parser fetched the remote DTD; it must never do so")
	}
	if got.Response.Text != "Asset Group Deleted Successfully" {
		t.Fatalf("TEXT not decoded, got %q", got.Response.Text)
	}
}

func TestDecodeXMLRejectsExternalEntity(t *testing.T) {
	// Classic XXE: an external entity pointing at a local file.
	var fetched bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetched = true
		_, _ = w.Write([]byte("SECRET"))
	}))
	defer srv.Close()

	body := `<?xml version="1.0"?>
<!DOCTYPE SIMPLE_RETURN [
  <!ENTITY xxe SYSTEM "` + srv.URL + `/secret">
]>
<SIMPLE_RETURN><RESPONSE><TEXT>&xxe;</TEXT></RESPONSE></SIMPLE_RETURN>`

	var got simpleReturn
	err := decodeXMLBytes([]byte(body), &got)
	if err == nil && strings.Contains(got.Response.Text, "SECRET") {
		t.Fatal("external entity was expanded; XXE is reachable")
	}
	if fetched {
		t.Fatal("parser fetched an external entity over the network")
	}
}

func TestDecodeXMLRejectsEntityExpansion(t *testing.T) {
	// Billion laughs. Needs no network access, so disabling remote fetches alone
	// would not stop it — the fixed entity map is what does.
	body := `<?xml version="1.0"?>
<!DOCTYPE SIMPLE_RETURN [
  <!ENTITY a "aaaaaaaaaa">
  <!ENTITY b "&a;&a;&a;&a;&a;&a;&a;&a;&a;&a;">
  <!ENTITY c "&b;&b;&b;&b;&b;&b;&b;&b;&b;&b;">
  <!ENTITY d "&c;&c;&c;&c;&c;&c;&c;&c;&c;&c;">
]>
<SIMPLE_RETURN><RESPONSE><TEXT>&d;</TEXT></RESPONSE></SIMPLE_RETURN>`

	var got simpleReturn
	if err := decodeXMLBytes([]byte(body), &got); err == nil {
		if len(got.Response.Text) > 1000 {
			t.Fatalf("entity expansion occurred: produced %d bytes", len(got.Response.Text))
		}
	}
}

func TestDecodeXMLRejectsMalformed(t *testing.T) {
	if err := decodeXMLBytes([]byte(`<SIMPLE_RETURN><RESPONSE>`), new(simpleReturn)); err == nil {
		t.Fatal("expected an error decoding truncated XML")
	}
}
