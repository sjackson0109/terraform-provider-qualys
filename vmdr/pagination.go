package vmdr

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// truncationWarningCode is the WARNING/CODE value VMDR uses to signal that a list
// response was truncated and a further request is needed. It is a warning, not an
// error: the records already returned are valid.
const truncationWarningCode = 1980

// warning is the continuation marker embedded in truncated list responses.
//
// The URL element carries a fully-formed next request, including the cursor
// (typically id_min). Following that URL is the documented mechanism; the client
// does not attempt to reconstruct the cursor itself, because the parameter differs
// between endpoints and a hand-built cursor risks silently skipping records.
type warning struct {
	Code int    `xml:"CODE"`
	Text string `xml:"TEXT"`
	URL  string `xml:"URL"`
}

// truncated reports whether this warning is a continuation marker.
func (w *warning) truncated() bool {
	return w != nil && w.Code == truncationWarningCode && strings.TrimSpace(w.URL) != ""
}

// paginated is implemented by list response types that can be truncated.
type paginated interface {
	// warning returns the response's WARNING element, or nil when absent.
	warning() *warning
}

// listAll fetches every page of a truncated list.
//
// next is called with each decoded page; it should accumulate results. The page
// value passed to next is reused between iterations only if newPage returns the
// same pointer, so newPage should allocate a fresh value per page.
//
// maxPages bounds the walk. A server that always returns a continuation marker
// would otherwise loop forever; hitting the bound is reported as an error rather
// than silently returning partial data, because a caller that believes it has the
// full set will make wrong decisions with it (for example, deciding an asset is
// absent and purging it).
func (c *Client) listAll(
	ctx context.Context,
	req request,
	newPage func() paginated,
	onPage func(paginated) error,
	maxPages int,
) error {
	if maxPages <= 0 {
		maxPages = 1000
	}

	page := newPage()
	if err := c.do(ctx, req, page); err != nil {
		return err
	}
	if err := onPage(page); err != nil {
		return err
	}

	for i := 1; ; i++ {
		w := page.warning()
		if !w.truncated() {
			return nil
		}
		if i >= maxPages {
			return fmt.Errorf("qualys vmdr: list did not terminate after %d pages; "+
				"refusing to return partial results", maxPages)
		}

		nextReq, err := continuationRequest(req, w.URL)
		if err != nil {
			return err
		}

		page = newPage()
		if err := c.do(ctx, nextReq, page); err != nil {
			return err
		}
		if err := onPage(page); err != nil {
			return err
		}
	}
}

// continuationRequest rebuilds req to fetch the next page described by rawURL.
//
// Only the query parameters are taken from the continuation URL. The host is
// deliberately ignored and the client's configured base URL is reused, so a
// response cannot redirect the client — with its credentials — to another host.
func continuationRequest(req request, rawURL string) (request, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return request{}, fmt.Errorf("qualys vmdr: parsing continuation URL: %w", err)
	}

	q := u.Query()
	next := req
	next.params = url.Values{}
	for k, vs := range q {
		if strings.EqualFold(k, "action") {
			continue // carried on the request itself
		}
		for _, v := range vs {
			next.params.Add(k, v)
		}
	}
	// A continuation is always a read.
	next.destructive = false
	return next, nil
}
