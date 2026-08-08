package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testList models a truncated list response.
type testList struct {
	Response struct {
		IDs     []string `xml:"ID_SET>ID"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (l *testList) warning() *warning { return l.Response.Warning }

func TestListAllFollowsTruncationWarning(t *testing.T) {
	var seenIDMin []string
	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		idMin := r.Form.Get("id_min")
		seenIDMin = append(seenIDMin, idMin)

		switch idMin {
		case "":
			// First page, truncated, with a continuation URL.
			fmt.Fprintf(w, `<HOST_LIST_OUTPUT><RESPONSE>
			  <ID_SET><ID>1</ID><ID>2</ID></ID_SET>
			  <WARNING>
			    <CODE>1980</CODE>
			    <TEXT>2 record limit exceeded. Use URL to get next batch of results.</TEXT>
			    <URL>https://qualysapi.example.com/api/2.0/fo/asset/host/?action=list&amp;id_min=3</URL>
			  </WARNING>
			</RESPONSE></HOST_LIST_OUTPUT>`)
		case "3":
			// Final page, no warning.
			fmt.Fprint(w, `<HOST_LIST_OUTPUT><RESPONSE>
			  <ID_SET><ID>3</ID></ID_SET>
			</RESPONSE></HOST_LIST_OUTPUT>`)
		default:
			t.Errorf("unexpected id_min %q", idMin)
		}
	})

	c, srv := testClient(t, mux)
	defer srv.Close()

	var all []string
	err := c.listAll(context.Background(),
		request{capability: capAssetHost, path: "asset/host/", action: "list"},
		func() paginated { return new(testList) },
		func(p paginated) error {
			all = append(all, p.(*testList).Response.IDs...)
			return nil
		}, 10)
	if err != nil {
		t.Fatalf("listAll: %v", err)
	}

	if got, want := strings.Join(all, ","), "1,2,3"; got != want {
		t.Errorf("collected %q, want %q", got, want)
	}
	if len(seenIDMin) != 2 {
		t.Errorf("made %d requests, want 2", len(seenIDMin))
	}
}

// The continuation URL names a host. The client must reuse its own base URL and
// take only the query from it, so a response cannot redirect credentials elsewhere.
func TestContinuationDoesNotFollowForeignHost(t *testing.T) {
	var foreignHit bool
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		foreignHit = true
	}))
	defer foreign.Close()

	var pages int
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		if pages == 1 {
			fmt.Fprintf(w, `<HOST_LIST_OUTPUT><RESPONSE>
			  <ID_SET><ID>1</ID></ID_SET>
			  <WARNING><CODE>1980</CODE><URL>%s/api/2.0/fo/asset/host/?action=list&amp;id_min=2</URL></WARNING>
			</RESPONSE></HOST_LIST_OUTPUT>`, foreign.URL)
			return
		}
		fmt.Fprint(w, `<HOST_LIST_OUTPUT><RESPONSE><ID_SET><ID>2</ID></ID_SET></RESPONSE></HOST_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	err := c.listAll(context.Background(),
		request{capability: capAssetHost, path: "asset/host/", action: "list"},
		func() paginated { return new(testList) },
		func(paginated) error { return nil }, 10)
	if err != nil {
		t.Fatalf("listAll: %v", err)
	}
	if foreignHit {
		t.Fatal("client followed a continuation URL to a foreign host, leaking credentials")
	}
	if pages != 2 {
		t.Errorf("pages = %d, want 2", pages)
	}
}

// A server that always signals truncation must not loop forever, and must not
// silently return partial data either.
func TestListAllRefusesPartialResultsOnRunaway(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<HOST_LIST_OUTPUT><RESPONSE>
		  <ID_SET><ID>1</ID></ID_SET>
		  <WARNING><CODE>1980</CODE><URL>https://x/api/2.0/fo/asset/host/?action=list&amp;id_min=1</URL></WARNING>
		</RESPONSE></HOST_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	err := c.listAll(context.Background(),
		request{capability: capAssetHost, path: "asset/host/", action: "list"},
		func() paginated { return new(testList) },
		func(paginated) error { return nil }, 3)
	if err == nil {
		t.Fatal("expected an error rather than silently truncated results")
	}
	if !strings.Contains(err.Error(), "refusing to return partial results") {
		t.Errorf("unexpected error: %v", err)
	}
}
