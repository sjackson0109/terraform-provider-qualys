package vmdr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(t *testing.T, h http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	c, err := NewClient(Config{
		BaseURL:    srv.URL,
		Username:   "u",
		Password:   "p",
		HTTPClient: srv.Client(),
		MaxRetries: 2,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, srv
}

func TestNewClientRequiresExplicitHTTPSBaseURL(t *testing.T) {
	if _, err := NewClient(Config{Username: "u", Password: "p"}); err == nil {
		t.Fatal("expected an error when BaseURL is empty; the POD hostname must never be guessed")
	}
	if _, err := NewClient(Config{BaseURL: "http://qualysapi.qualys.com", Username: "u", Password: "p"}); err == nil {
		t.Fatal("expected an error for a plaintext BaseURL")
	}
	if _, err := NewClient(Config{BaseURL: "https://qualysapi.qualys.com"}); err == nil {
		t.Fatal("expected an error when credentials are missing")
	}
}

func TestRequestSendsMandatoryHeaderAndAuth(t *testing.T) {
	var gotHeader, gotUser string
	var gotAction, gotContentType string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get(requestedWithHeader)
		gotContentType = r.Header.Get("Content-Type")
		u, _, _ := r.BasicAuth()
		gotUser = u
		_ = r.ParseForm()
		gotAction = r.Form.Get("action")
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	err := c.do(context.Background(), request{
		capability: capAssetGroup, path: "asset/group/", action: "list",
	}, nil)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if gotHeader == "" {
		t.Error("X-Requested-With was not sent; Qualys rejects v2 calls without it")
	}
	if gotUser != "u" {
		t.Errorf("basic auth user = %q, want %q", gotUser, "u")
	}
	if gotAction != "list" {
		t.Errorf("action = %q, want list", gotAction)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
}

func TestEndpointUsesPerCapabilityVersion(t *testing.T) {
	var gotPath string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	_ = c.do(context.Background(), request{capability: capAssetGroup, path: "asset/group/", action: "list"}, nil)
	if want := "/api/2.0/fo/asset/group/"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

// A 200 response carrying an error CODE is a failure. Branching on HTTP status
// alone would treat this as success.
func TestErrorInsideHTTP200IsAFailure(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><CODE>1905</CODE><TEXT>Id not found</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	err := c.do(context.Background(), request{capability: capAssetGroup, path: "asset/group/", action: "edit"}, nil)
	if err == nil {
		t.Fatal("expected an error from a 200 response carrying an error code")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestEULAFailureIsIdentified(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>User has not accepted the Qualys EULA</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	err := c.do(context.Background(), request{capability: capAssetIP, path: "asset/ip/", action: "list"}, nil)
	if !errors.Is(err, ErrEULANotAccepted) {
		t.Fatalf("expected ErrEULANotAccepted, got %v", err)
	}
}

func TestRateLimitRetriesAfterDocumentedWait(t *testing.T) {
	var calls int32
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("X-RateLimit-ToWait-Sec", "1")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.WriteHeader(http.StatusConflict)
			fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>This API cannot be run again for another 1 seconds.</TEXT></RESPONSE></SIMPLE_RETURN>`)
			return
		}
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	start := time.Now()
	err := c.do(context.Background(), request{capability: capAssetIP, path: "asset/ip/", action: "list"}, nil)
	if err != nil {
		t.Fatalf("expected the retry to succeed, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("call count = %d, want 2", got)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Errorf("retried after %s; must honour X-RateLimit-ToWait-Sec", elapsed)
	}
}

// A destructive call must not be repeated automatically: purge and delete are not
// idempotent in a way that makes a blind retry safe.
func TestDestructiveCallIsNotRetried(t *testing.T) {
	var calls int32
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("X-RateLimit-ToWait-Sec", "1")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>This API cannot be run again for another 1 seconds.</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	err := c.do(context.Background(), request{
		capability: capAssetHost, path: "asset/host/", action: "purge", nonIdempotent: true,
	}, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("destructive call was issued %d times; it must be issued once", got)
	}
}

func TestConcurrencyIsBounded(t *testing.T) {
	var inFlight, peak int32
	var mu sync.Mutex
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&inFlight, 1)
		mu.Lock()
		if n > peak {
			peak = n
		}
		mu.Unlock()
		time.Sleep(30 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.do(context.Background(), request{capability: capAssetIP, path: "asset/ip/", action: "list"}, nil)
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if peak > defaultConcurrency {
		t.Errorf("peak concurrency %d exceeded the subscription default of %d", peak, defaultConcurrency)
	}
}

func TestContextCancellationIsHonoured(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := c.do(ctx, request{capability: capAssetIP, path: "asset/ip/", action: "list"}, nil); err == nil {
		t.Fatal("expected the cancelled context to abort the call")
	}
}

// A create must never be re-sent after a transport failure. The server may have
// processed the first attempt, so retrying would duplicate the object — and the
// duplicate would be invisible to Terraform, which only records one ID.
func TestCreateIsNotRetriedOnTransportError(t *testing.T) {
	var calls int32
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		// Hang up mid-response to simulate a lost reply to a processed request.
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("test server does not support hijacking")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		conn.Close()
	}))
	defer srv.Close()

	err := c.do(context.Background(), request{
		capability: capAssetGroup, path: "asset/group/", action: "add",
		nonIdempotent: true,
	}, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("create was sent %d times; a lost response must not cause a re-send", got)
	}
}

// Idempotent calls may still be retried: repeating a list or an update with the
// same parameters cannot create anything.
func TestIdempotentCallIsRetriedOnTransportError(t *testing.T) {
	var calls int32
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>ok</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	if err := c.do(context.Background(), request{
		capability: capAssetGroup, path: "asset/group/", action: "edit",
	}, nil); err != nil {
		t.Fatalf("an idempotent call should recover: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("call count = %d, want 2", got)
	}
}
