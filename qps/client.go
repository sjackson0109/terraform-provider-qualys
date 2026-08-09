// Package qps talks to the Qualys portal APIs: Asset Management & Tagging
// (/qps/rest/2.0/.../am/...) and Web Application Scanning
// (/qps/rest/3.0/.../was/...).
//
// This is a different API family from the VM/PA XML API in package vmdr, and
// shares nothing with it beyond credentials: different base path, envelope
// (ServiceRequest/ServiceResponse), encoding (JSON here, form/XML there),
// pagination (hasMoreRecords/lastId versus a truncation warning), and error model
// (a responseCode string versus a numeric code in an XML body).
//
// They are deliberately separate clients. Folding them together would mean one
// set of retry, pagination and error-classification rules serving two APIs that
// agree on none of those things.
package qps

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultConcurrency = 2
	defaultTimeout     = 5 * time.Minute
	maxResponseBytes   = 256 << 20
)

// Config configures a Client.
type Config struct {
	// BaseURL is the platform API server, e.g. https://qualysapi.qualys.com.
	BaseURL string

	Username string
	Password string

	UserAgent string

	// Concurrency caps in-flight requests. Zero uses 2.
	Concurrency int
	// MaxRetries bounds retries of limit-blocked requests. Zero uses 4.
	MaxRetries int

	HTTPClient *http.Client
	Logger     *log.Logger
}

// Client is a Qualys portal API client. It is safe for concurrent use.
type Client struct {
	baseURL *url.URL
	cfg     Config
	http    *http.Client
	sem     chan struct{}
}

// NewClient validates cfg and returns a Client.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("qualys qps: BaseURL is required " +
			"(the platform API server for your subscription, e.g. https://qualysapi.qualys.com)")
	}
	u, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("qualys qps: parsing BaseURL: %w", err)
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("qualys qps: BaseURL must use https, got %q", u.Scheme)
	}
	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("qualys qps: username and password are required")
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "terraform-provider-qualys"
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

// ServiceRequest is the portal API request envelope.
type ServiceRequest struct {
	Filters     *Filters     `json:"filters,omitempty"`
	Preferences *Preferences `json:"preferences,omitempty"`
	Data        interface{}  `json:"data,omitempty"`
}

// Preferences controls paging.
type Preferences struct {
	StartFromOffset int `json:"startFromOffset,omitempty"`
	StartFromID     int `json:"startFromId,omitempty"`
	LimitResults    int `json:"limitResults,omitempty"`
}

// Filters carries search criteria.
type Filters struct {
	Criteria []Criterion `json:"Criteria,omitempty"`
}

// Criterion is one search predicate. Operators documented for the portal APIs
// are EQUALS, NOT EQUALS, CONTAINS, GREATER, LESSER, IN and NONE; not every
// operator is valid on every field.
type Criterion struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// ServiceResponse is the portal API response envelope.
type ServiceResponse struct {
	ResponseCode        string `json:"responseCode"`
	Count               int    `json:"count"`
	HasMoreRecords      string `json:"hasMoreRecords"`
	LastID              int64  `json:"lastId"`
	ResponseErrorDetail *struct {
		ErrorMessage    string `json:"errorMessage"`
		ErrorResolution string `json:"errorResolution"`
	} `json:"responseErrorDetails"`

	// Data is left raw so each object type can decode its own shape.
	Data json.RawMessage `json:"data"`
}

func (r *ServiceResponse) hasMore() bool {
	return strings.EqualFold(strings.TrimSpace(r.HasMoreRecords), "true")
}

// Error is a failure reported by a portal API.
type Error struct {
	Status     int
	Code       string
	Message    string
	Resolution string
	Path       string
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("qualys qps: ")
	if e.Path != "" {
		fmt.Fprintf(&b, "%s: ", e.Path)
	}
	if e.Code != "" {
		fmt.Fprintf(&b, "%s", e.Code)
	} else {
		fmt.Fprintf(&b, "HTTP %d", e.Status)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	if e.Resolution != "" {
		fmt.Fprintf(&b, " (%s)", e.Resolution)
	}
	return b.String()
}

// ErrNotFound reports that an object does not exist.
//
// The portal APIs return a specific responseCode for this, so detection is exact
// here — unlike the VM/PA family, where the code table is not yet available.
//
// One caveat carried from the documentation: OBJECT_NOT_FOUND is also returned
// when the object exists but lies outside the caller's scope. A permissions
// change can therefore look identical to a deletion. Callers that would destroy
// or recreate on this signal should be sure the credentials' scope has not
// changed.
var ErrNotFound = errors.New("object not found (or outside the caller's scope)")

// ErrUnauthorized reports bad credentials or a missing permission.
var ErrUnauthorized = errors.New("unauthorized or insufficient permission")

// ErrLimitExceeded reports a rate, concurrency or storage limit.
var ErrLimitExceeded = errors.New("API or storage limit exceeded")

func (e *Error) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return e.Code == "OBJECT_NOT_FOUND"
	case ErrUnauthorized:
		return e.Code == "UNAUTHORIZED" || e.Code == "INSUFFICIENT_PERMISSION" ||
			e.Status == http.StatusUnauthorized || e.Status == http.StatusForbidden
	case ErrLimitExceeded:
		return e.Code == "LIMIT_EXCEEDED" ||
			e.Status == http.StatusConflict || e.Status == http.StatusTooManyRequests
	}
	return false
}

// call issues a single portal API request.
//
// path is the portion after the host, e.g. "/qps/rest/2.0/search/am/tag".
// body may be nil for operations that take none. out, if non-nil, receives the
// decoded ServiceResponse.
// call issues a request. nonIdempotent marks creates and deletes, which must
// never be repeated automatically: neither a transport error nor a limit
// response proves the server did not process the request, and repeating a
// create duplicates the object outside Terraform's knowledge.
func (c *Client) call(ctx context.Context, method, path string, body interface{}, out *ServiceResponse, nonIdempotent bool) error {
	var payload []byte
	if body != nil {
		// The portal API nests the request under a ServiceRequest key:
		// {"ServiceRequest": {...}}. Marshalling the body bare produces a payload
		// the real service ignores, so the wrapper is applied here, once, rather
		// than trusting every caller to remember it.
		var err error
		payload, err = json.Marshal(map[string]interface{}{"ServiceRequest": body})
		if err != nil {
			return fmt.Errorf("qualys qps: encoding request: %w", err)
		}
	}

	for attempt := 0; ; attempt++ {
		status, respBody, hdr, err := c.roundTrip(ctx, method, path, payload)
		if err != nil {
			if nonIdempotent || attempt >= c.cfg.MaxRetries {
				return fmt.Errorf("qualys qps: %s: %w", path, err)
			}
			if waitErr := sleepCtx(ctx, backoff(attempt)); waitErr != nil {
				return waitErr
			}
			continue
		}

		sr := decodeServiceResponse(respBody)

		apiErr := classify(status, sr, respBody, path)
		if apiErr == nil {
			if out != nil {
				*out = *sr
			}
			return nil
		}

		var e *Error
		if errors.As(apiErr, &e) && e.Is(ErrLimitExceeded) && attempt < c.cfg.MaxRetries && !nonIdempotent {
			wait := waitFromHeaders(hdr)
			if wait <= 0 {
				wait = backoff(attempt)
			}
			c.logf("qualys qps: limit reached on %s; waiting %s before retry (attempt %d/%d)",
				path, wait, attempt+1, c.cfg.MaxRetries)
			if waitErr := sleepCtx(ctx, wait); waitErr != nil {
				return waitErr
			}
			continue
		}
		return apiErr
	}
}

func (c *Client) roundTrip(ctx context.Context, method, path string, payload []byte) (int, []byte, http.Header, error) {
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return 0, nil, nil, ctx.Err()
	}

	var rdr io.Reader
	if payload != nil {
		rdr = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL.String()+path, rdr)
	if err != nil {
		return 0, nil, nil, err
	}

	// JSON on both sides. The portal APIs default to XML and switch to JSON only
	// when both headers are present, so both are always set — including on GET
	// and body-less POST calls, where omitting Content-Type would flip the
	// response back to XML.
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", c.cfg.UserAgent)
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.SetBasicAuth(c.cfg.Username, c.cfg.Password)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return resp.StatusCode, nil, resp.Header, err
	}
	return resp.StatusCode, b, resp.Header, nil
}

// decodeServiceResponse unwraps a portal API body.
//
// The documented shape nests the payload under a ServiceResponse key:
// {"ServiceResponse": {"responseCode": ...}}. The bare shape is also accepted,
// so a response that arrives unwrapped still decodes instead of presenting as
// an empty responseCode — which classify would report as a failure even on a
// 200. A body that decodes as neither is left zeroed and reported with its
// HTTP status.
func decodeServiceResponse(body []byte) *ServiceResponse {
	var outer struct {
		ServiceResponse ServiceResponse `json:"ServiceResponse"`
	}
	if err := json.Unmarshal(body, &outer); err == nil && outer.ServiceResponse.ResponseCode != "" {
		return &outer.ServiceResponse
	}
	var sr ServiceResponse
	_ = json.Unmarshal(body, &sr)
	return &sr
}

func classify(status int, sr *ServiceResponse, raw []byte, path string) error {
	code := strings.TrimSpace(sr.ResponseCode)

	if status >= 200 && status < 300 && strings.EqualFold(code, "SUCCESS") {
		return nil
	}

	e := &Error{Status: status, Code: code, Path: path}
	if sr.ResponseErrorDetail != nil {
		e.Message = sr.ResponseErrorDetail.ErrorMessage
		e.Resolution = sr.ResponseErrorDetail.ErrorResolution
	}
	if e.Code == "" && e.Message == "" {
		e.Message = excerpt(raw, 512)
	}
	return e
}

// SearchAll runs a search, following pagination until the server reports no more
// records, and calls onPage with each page's raw data.
//
// Paging uses the documented id-cursor: the next request filters on
// id GREATER lastId. maxPages bounds the walk; exceeding it is an error rather
// than a silent truncation, because a caller acting on a partial set will make
// decisions the full set would not support.
func (c *Client) SearchAll(ctx context.Context, path string, filters *Filters, pageSize int, maxPages int, onPage func(json.RawMessage) error) error {
	if maxPages <= 0 {
		maxPages = 1000
	}
	if pageSize <= 0 {
		pageSize = 100
	}

	req := &ServiceRequest{
		Filters:     filters,
		Preferences: &Preferences{LimitResults: pageSize},
	}

	for page := 0; ; page++ {
		if page >= maxPages {
			return fmt.Errorf("qualys qps: %s did not terminate after %d pages; "+
				"refusing to return partial results", path, maxPages)
		}

		var resp ServiceResponse
		if err := c.call(ctx, http.MethodPost, path, req, &resp, false); err != nil {
			return err
		}
		if len(resp.Data) > 0 {
			if err := onPage(resp.Data); err != nil {
				return err
			}
		}
		if !resp.hasMore() {
			return nil
		}

		// Advance the cursor. Copy the caller's criteria so the original filter is
		// not mutated across pages, and replace any previous cursor.
		next := &Filters{}
		if filters != nil {
			for _, cr := range filters.Criteria {
				if cr.Field == "id" && cr.Operator == "GREATER" {
					continue
				}
				next.Criteria = append(next.Criteria, cr)
			}
		}
		next.Criteria = append(next.Criteria, Criterion{
			Field: "id", Operator: "GREATER", Value: strconv.FormatInt(resp.LastID, 10),
		})
		req = &ServiceRequest{
			Filters:     next,
			Preferences: &Preferences{LimitResults: pageSize},
		}
	}
}

// decodeListOrSingle splits a portal API "data" payload into individual
// object payloads, handling the two shapes the API uses: a JSON array for
// search results, or a single bare object for get/create. Each returned
// element is valid JSON on its own, for the caller to unmarshal into its own
// typed wrapper — this only factors out the shape detection, which every
// object type in this package (Tag, WebApp, WASOptionProfile, ...) needs
// identically; the per-type unmarshalling stays with the caller since Go's
// lack of generics before this module's minimum version rules out a fully
// generic decoder.
func decodeListOrSingle(raw json.RawMessage) []json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err == nil {
		return list
	}
	return []json.RawMessage{raw}
}

func (c *Client) logf(format string, args ...interface{}) {
	if c.cfg.Logger == nil {
		return
	}
	c.cfg.Logger.Printf(format, args...)
}

func waitFromHeaders(h http.Header) time.Duration {
	if h == nil {
		return 0
	}
	for _, k := range []string{"X-RateLimit-ToWait-Sec", "Retry-After"} {
		if v := h.Get(k); v != "" {
			if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && secs > 0 {
				return time.Duration(secs) * time.Second
			}
		}
	}
	return 0
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

func excerpt(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
