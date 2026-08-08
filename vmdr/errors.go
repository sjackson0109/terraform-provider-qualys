package vmdr

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// simpleReturn is the envelope VMDR returns from mutating actions, and from most
// failures regardless of the HTTP status. DTD: /api/2.0/simple_return.dtd
type simpleReturn struct {
	Response struct {
		DateTime string `xml:"DATETIME"`
		Code     int    `xml:"CODE"`
		Text     string `xml:"TEXT"`
		ItemList struct {
			Items []struct {
				Key   string `xml:"KEY"`
				Value string `xml:"VALUE"`
			} `xml:"ITEM"`
		} `xml:"ITEM_LIST"`
	} `xml:"RESPONSE"`
}

// itemValue returns the first ITEM_LIST value for key, e.g. "ID" on a create.
func (s *simpleReturn) itemValue(key string) (string, bool) {
	for _, it := range s.Response.ItemList.Items {
		if strings.EqualFold(it.Key, key) {
			return it.Value, true
		}
	}
	return "", false
}

// Error is a failure reported by the VMDR API.
//
// HTTP status is deliberately not the primary signal. Qualys returns 200 with an
// error body on some calls, and conversely returns 409 for limit conditions whose
// meaning is only in the body text. Callers should test behaviour with the Is*
// helpers rather than inspecting Status.
type Error struct {
	// Status is the HTTP status code, recorded for diagnostics only.
	Status int
	// Code is the Qualys numeric code from SIMPLE_RETURN/RESPONSE/CODE, or 0 when
	// the response carried no code.
	Code int
	// Text is the human-readable message from SIMPLE_RETURN/RESPONSE/TEXT.
	Text string
	// Endpoint and Action identify the call that failed.
	Endpoint string
	Action   string
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("qualys vmdr: ")
	if e.Action != "" {
		fmt.Fprintf(&b, "%s %s: ", e.Endpoint, e.Action)
	} else if e.Endpoint != "" {
		fmt.Fprintf(&b, "%s: ", e.Endpoint)
	}
	if e.Text != "" {
		b.WriteString(e.Text)
	} else {
		fmt.Fprintf(&b, "HTTP %d", e.Status)
	}
	if e.Code != 0 {
		fmt.Fprintf(&b, " (code %d)", e.Code)
	}
	return b.String()
}

// ErrNotFound reports that the target object does not exist.
//
// The VM/PA error-code table is in an appendix of the API user guide that this
// project has not yet obtained, so there is no verified numeric code to match on.
// Rather than guess a code and silently mis-handle deletions, detection is
// text-based and deliberately conservative: an ambiguous failure is reported as an
// ordinary error, which surfaces to the user, instead of being treated as "already
// gone", which would silently drop a resource from state.
//
// Callers that must be certain an object is absent should re-read it (a list by ID
// returning an empty set is unambiguous) rather than relying on this.
var ErrNotFound = errors.New("object not found")

// ErrEULANotAccepted reports an account that has not accepted the Qualys EULA.
//
// An API-only account (no GUI access) stays inactive until the EULA is accepted,
// and every call fails until then. It is the most likely cause of a brand-new
// service account failing everything, and it presents as a generic authorisation
// failure, so it is called out explicitly.
var ErrEULANotAccepted = errors.New("account has not accepted the Qualys EULA; " +
	"an API-only account remains inactive until the EULA is accepted and all API calls will fail")

// ErrRateLimited reports that a rate or concurrency limit blocked the call.
var ErrRateLimited = errors.New("API limit reached")

func (e *Error) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return e.isNotFound()
	case ErrEULANotAccepted:
		return e.isEULA()
	case ErrRateLimited:
		return e.isRateLimited()
	}
	return false
}

// notFoundPhrases are matched case-insensitively against TEXT. Kept narrow on
// purpose: a false positive here removes a live resource from Terraform state.
var notFoundPhrases = []string{
	"not found",
	"does not exist",
	"no such",
	"unknown id",
	"invalid id",
}

func (e *Error) isNotFound() bool {
	t := strings.ToLower(e.Text)
	for _, p := range notFoundPhrases {
		if strings.Contains(t, p) {
			return true
		}
	}
	return false
}

func (e *Error) isEULA() bool {
	t := strings.ToLower(e.Text)
	return strings.Contains(t, "eula") ||
		strings.Contains(t, "license agreement") ||
		strings.Contains(t, "licence agreement")
}

func (e *Error) isRateLimited() bool {
	if e.Status == http.StatusConflict || e.Status == http.StatusTooManyRequests {
		return true
	}
	t := strings.ToLower(e.Text)
	// A 409 whose meaning lives only in the body.
	return strings.Contains(t, "cannot be run again for another") ||
		strings.Contains(t, "concurrent") && strings.Contains(t, "limit")
}

// classifyResponse turns a completed HTTP exchange into an error, or nil when the
// call succeeded.
//
// The body is parsed before the status is considered, because a 200 can carry a
// failure and a 409 can be a limit condition rather than an application error.
func classifyResponse(status int, body []byte, endpoint, action string) error {
	e := &Error{Status: status, Endpoint: endpoint, Action: action}

	// A SIMPLE_RETURN body is authoritative when present, whatever the status.
	var sr simpleReturn
	if err := decodeXMLBytes(body, &sr); err == nil {
		e.Code = sr.Response.Code
		e.Text = strings.TrimSpace(sr.Response.Text)
	}

	// Limit conditions are reported even on an otherwise-parseable body so the
	// retry layer can act on them.
	if status == http.StatusConflict || status == http.StatusTooManyRequests {
		return e
	}

	if status >= 200 && status < 300 {
		// 2xx with an error code set is still a failure. Code 0 means the
		// response carried no CODE element, which is the success shape.
		if e.Code != 0 {
			return e
		}
		return nil
	}

	if e.Text == "" {
		// Non-XML error body (Qualys serves HTML for some failures). Keep a
		// bounded excerpt for diagnostics rather than the whole page.
		e.Text = excerpt(body, 512)
	}
	return e
}

func excerpt(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
