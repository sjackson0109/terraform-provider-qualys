package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// PortalVersion is the tenant capability probe result: whatever the portal
// version endpoint reports about the installed platform and per-module
// versions (WAS, VM, CA, FIM, ...). Doc 11's "Useful for the tenant
// capability probe" section names `GET /qps/rest/portal/version` as
// returning this information and recommends it over cheap-list probing,
// but this session found no field-level schema for its response — not
// even a DTD or class name, unlike every other qps object in this package.
//
// Raw therefore carries the decoded top-level JSON object verbatim,
// flattened to strings (nested values are re-encoded as their own JSON
// text rather than dropped), instead of this package asserting field names
// it has no evidence for. This is a deliberate generic passthrough, not a
// typed schema — inspect Raw's keys against your own tenant's response
// before depending on any particular one in Terraform configuration.
type PortalVersion struct {
	Raw map[string]string
}

// GetPortalVersion calls the tenant capability probe endpoint and returns
// its response as a flattened key/value map.
//
// This bypasses the ServiceRequest/ServiceResponse envelope every other
// call in this package uses: that envelope shape ({"responseCode": ...,
// "data": ...}) is Confirmed for the versioned /qps/rest/<ver>/... object
// APIs, but /qps/rest/portal/version does not follow that path shape (no
// version segment, no operation verb, no module/object), so assuming the
// same envelope here would itself be an unconfirmed guess. Decoding
// generically avoids that assumption regardless of which shape the tenant
// actually returns.
func (c *Client) GetPortalVersion(ctx context.Context) (*PortalVersion, error) {
	status, body, _, err := c.roundTrip(ctx, http.MethodGet, "/qps/rest/portal/version", nil)
	if err != nil {
		return nil, fmt.Errorf("qualys qps: portal/version: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("qualys qps: portal/version: HTTP %d: %s", status, excerpt(body, 512))
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("qualys qps: portal/version: decoding response: %w", err)
	}

	flat := make(map[string]string, len(raw))
	for k, v := range raw {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			flat[k] = s
			continue
		}
		// Not a bare JSON string (a nested object, number, or array) — keep
		// its own JSON text rather than lose the value.
		flat[k] = string(v)
	}

	return &PortalVersion{Raw: flat}, nil
}
