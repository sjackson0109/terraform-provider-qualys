package vmdr

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// requestedWithHeader is mandatory on every v2 call, under both Basic and
	// session authentication. Qualys rejects requests without it.
	requestedWithHeader = "X-Requested-With"

	defaultUserAgent = "terraform-provider-qualys"

	// defaultConcurrency matches the documented per-subscription default of 2
	// concurrent calls per API. Exceeding it earns a 409, so the client
	// self-limits rather than relying on retries to absorb the overshoot.
	defaultConcurrency = 2

	defaultTimeout = 5 * time.Minute
)

// Config configures a Client.
type Config struct {
	// BaseURL is the platform API server, e.g. https://qualysapi.qualys.com.
	// It is required and must be supplied explicitly: Qualys hostnames vary by
	// POD, the published mapping could not be fully verified, and guessing one
	// would send credentials to the wrong platform.
	BaseURL string

	Username string
	Password string

	// UserAgent is sent as X-Requested-With and User-Agent.
	UserAgent string

	// Concurrency caps in-flight requests. Zero uses the documented default of 2.
	// Raise it only for subscriptions whose limits Qualys has actually raised.
	Concurrency int

	// MaxRetries bounds retries of limit-blocked requests. Zero uses 4.
	MaxRetries int

	// HTTPClient overrides the underlying client, for tests or custom transports.
	HTTPClient *http.Client

	// Logger receives operational warnings (deprecated API versions, limit waits).
	// Nil discards them.
	Logger *log.Logger
}

// Client talks to the Qualys VM/PA (VMDR) XML API.
//
// It is safe for concurrent use.
type Client struct {
	baseURL *url.URL
	cfg     Config
	http    *http.Client

	// sem bounds concurrent in-flight requests to the subscription's limit.
	sem chan struct{}

	// warnedOnce tracks per-capability deprecation warnings so a busy provider
	// logs each migration notice once rather than once per API call.
	warnedOnce sync.Map // capability -> struct{}
}

// NewClient validates cfg and returns a Client.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("qualys vmdr: BaseURL is required " +
			"(the platform API server for your subscription, e.g. https://qualysapi.qualys.com); " +
			"find it under Help > About in the Qualys UI")
	}
	u, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("qualys vmdr: parsing BaseURL: %w", err)
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("qualys vmdr: BaseURL must use https, got %q", u.Scheme)
	}
	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("qualys vmdr: username and password are required")
	}

	if cfg.UserAgent == "" {
		cfg.UserAgent = defaultUserAgent
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = defaultConcurrency
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 4
	}

	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{
		baseURL: u,
		cfg:     cfg,
		http:    hc,
		sem:     make(chan struct{}, cfg.Concurrency),
	}, nil
}

// request describes a single VMDR call.
type request struct {
	// capability selects the API version and drives deprecation warnings.
	capability capability
	// path is appended after /api/<version>/fo/, e.g. "asset/group/".
	path string
	// action is the action= parameter, e.g. "list".
	action string
	// params are additional form parameters. action is added automatically.
	params url.Values
	// method defaults to POST. Reads may use GET.
	method string
	// rawEndpoint overrides capability/path version-prefixed URL building
	// entirely, for the handful of legacy /msp/*.php endpoints that predate
	// /api/<version>/fo/ and are kept only because they are the sole surface
	// for what they return (doc 09 §18). Use legacyEndpoint to build one.
	rawEndpoint string
	// nonIdempotent marks a call whose repetition is unsafe: creates (which would
	// duplicate the object or orphan one outside state) and destructive actions
	// such as delete and purge. Such calls are never retried automatically once
	// dispatched — not on a transport error, and not on a limit response, since
	// neither proves the server did not process the request.
	//
	// Updates are idempotent — the same parameters produce the same result — so
	// they are safe to retry and are not marked.
	nonIdempotent bool
}

// do executes req and decodes a successful response into out (which may be nil).
//
// Retries are limited to limit-blocked responses (409/429), and only for requests
// that are safe to repeat. A destructive call that has been dispatched is never
// retried: on an ambiguous failure the caller must re-read state to establish
// whether the operation applied, because a blind retry could purge or delete twice.
func (c *Client) do(ctx context.Context, req request, out interface{}) error {
	c.warnIfDeprecated(req.capability)

	endpoint := req.rawEndpoint
	if endpoint == "" {
		endpoint = c.endpoint(req.capability, req.path)
	}

	form := url.Values{}
	for k, vs := range req.params {
		for _, v := range vs {
			// A repeated parameter keeps only its last instance server-side, so
			// emitting duplicates silently drops values. Params is a url.Values
			// and callers Set rather than Add; this preserves any deliberate
			// repetition while keeping the encoding explicit.
			form.Add(k, v)
		}
	}
	if req.action != "" {
		form.Set("action", req.action)
	}

	method := req.method
	if method == "" {
		method = http.MethodPost
	}

	for attempt := 0; ; attempt++ {
		status, body, hdr, err := c.roundTrip(ctx, method, endpoint, form)
		if err != nil {
			// Transport failure. Safe to retry only for idempotent calls, since we
			// cannot tell whether the server processed the request.
			if req.nonIdempotent || attempt >= c.cfg.MaxRetries {
				return fmt.Errorf("qualys vmdr: %s %s: %w", endpoint, req.action, err)
			}
			if waitErr := sleepCtx(ctx, backoff(attempt)); waitErr != nil {
				return waitErr
			}
			continue
		}

		apiErr := classifyResponse(status, body, endpoint, req.action)
		if apiErr == nil {
			if out == nil {
				return nil
			}
			return decodeXMLBytes(body, out)
		}

		// Limit conditions are the only automatically retried failures.
		var e *Error
		if asError(apiErr, &e) && e.isRateLimited() && attempt < c.cfg.MaxRetries {
			// Non-idempotent calls are not retried even here: a 409 does not prove
			// the call was rejected rather than merely blocked late.
			if !req.nonIdempotent {
				wait := waitFromHeaders(hdr)
				if wait <= 0 {
					wait = backoff(attempt)
				}
				c.logf("qualys vmdr: %s limit reached on %s; waiting %s before retry (attempt %d/%d)",
					limitKind(hdr, e), endpoint, wait, attempt+1, c.cfg.MaxRetries)
				if waitErr := sleepCtx(ctx, wait); waitErr != nil {
					return waitErr
				}
				continue
			}
		}
		return apiErr
	}
}

func (c *Client) roundTrip(ctx context.Context, method, endpoint string, form url.Values) (int, []byte, http.Header, error) {
	// Bound concurrency to the subscription limit before issuing the request.
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return 0, nil, nil, ctx.Err()
	}

	var (
		httpReq *http.Request
		err     error
	)
	if method == http.MethodGet {
		httpReq, err = http.NewRequestWithContext(ctx, method, endpoint+"?"+form.Encode(), nil)
	} else {
		httpReq, err = http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(form.Encode()))
		if err == nil {
			httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if err != nil {
		return 0, nil, nil, err
	}

	httpReq.Header.Set(requestedWithHeader, c.cfg.UserAgent)
	httpReq.Header.Set("User-Agent", c.cfg.UserAgent)
	httpReq.SetBasicAuth(c.cfg.Username, c.cfg.Password)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return resp.StatusCode, nil, resp.Header, err
	}
	return resp.StatusCode, body, resp.Header, nil
}

// endpoint builds the full URL for a capability and path.
func (c *Client) endpoint(cap capability, path string) string {
	v := versionFor(cap)
	return fmt.Sprintf("%s/api/%s/fo/%s", c.baseURL.String(), v, strings.TrimLeft(path, "/"))
}

// legacyEndpoint builds the full URL for a V1 /msp/*.php script. These predate
// the versioned /api/<version>/fo/ family entirely and are not subject to its
// version selection or deprecation warnings.
func (c *Client) legacyEndpoint(script string) string {
	return fmt.Sprintf("%s/msp/%s", c.baseURL.String(), strings.TrimLeft(script, "/"))
}

func (c *Client) warnIfDeprecated(cap capability) {
	newer, ok := deprecatedCapabilities[cap]
	if !ok {
		return
	}
	if _, loaded := c.warnedOnce.LoadOrStore(cap, struct{}{}); loaded {
		return
	}
	c.logf("qualys vmdr: %s is being used at API version %s; Qualys has published version %s "+
		"for this capability. Superseded versions reach End of Support 6 months after their "+
		"replacement ships and End of Life 6 months later. Plan a migration.",
		cap, versionFor(cap), newer)
}

func (c *Client) logf(format string, args ...interface{}) {
	if c.cfg.Logger == nil {
		return
	}
	c.cfg.Logger.Printf(format, args...)
}

// waitFromHeaders reads the documented wait hint.
//
// X-RateLimit-ToWait-Sec is not always present on the first blocked response —
// Qualys sometimes only includes it once the call has been retried — so a missing
// header means "fall back to backoff", not "wait zero".
func waitFromHeaders(h http.Header) time.Duration {
	if h == nil {
		return 0
	}
	if v := h.Get("X-RateLimit-ToWait-Sec"); v != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	if v := h.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 0
}

// limitKind distinguishes concurrency from rate blocking for the log line.
func limitKind(h http.Header, e *Error) string {
	if h != nil {
		running := h.Get("X-Concurrency-Limit-Running")
		limit := h.Get("X-Concurrency-Limit-Limit")
		if running != "" && limit != "" && running == limit {
			return "concurrency"
		}
		if h.Get("X-RateLimit-Remaining") == "0" {
			return "rate"
		}
	}
	if e != nil && strings.Contains(strings.ToLower(e.Text), "concurrent") {
		return "concurrency"
	}
	return "rate"
}

func backoff(attempt int) time.Duration {
	d := time.Duration(1<<uint(attempt)) * time.Second
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// asError is errors.As specialised to *Error, kept local to avoid importing
// errors in the hot path of do().
func asError(err error, target **Error) bool {
	if e, ok := err.(*Error); ok {
		*target = e
		return true
	}
	return false
}
