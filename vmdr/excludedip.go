package vmdr

import (
	"context"
	"fmt"
	"net/url"
)

// Excluded IPs are addresses the subscription holds but that scans skip —
// for example a host that must never be scanned even though it is in an
// asset group's range. Confirmed by doc 03 §1 (lines 45-48):
//
//	list        GET/POST  ips
//	add         POST      ips, comment, expiry_days, dg_names, network_id
//	remove      POST      ips, comment, network_id
//	remove_all  POST      comment, network_id ("ips is invalid for remove_all")
//
// The list response's own XML element names are, deliberately, not decoded
// by this file: doc 03 confirms the endpoint and its request parameters,
// but this session found no DTD filename or field-name evidence for the
// "excluded host list XML" response shape it names — unlike, say,
// VMFinding's DETECTION_LIST, which at least had two of its field names
// independently confirmed elsewhere. Guessing element names here would be
// exactly the kind of unsupported inference this project's evidence
// discipline exists to prevent. ListExcludedIPs is therefore not
// implemented; the qualys_excluded_ips resource manages its configured set
// write-only, via idempotent add/remove calls, and does not attempt to
// read back what Qualys currently has excluded. See that resource's doc
// comment for the consequence (drift is not detected).

// AddExcludedIPsInput registers IPs as excluded from scanning.
type AddExcludedIPsInput struct {
	IPs []string

	Comment string
	// ExpiryDays automatically re-includes the IPs in scans after this many
	// days. Zero leaves the exclusion open-ended.
	ExpiryDays int
	// AssetGroupNames scopes the exclusion to these asset groups.
	AssetGroupNames []string
	NetworkID       string
}

// AddExcludedIPs excludes IPs from scanning. Confirmed idempotent by the
// generic SIMPLE_RETURN envelope every write action in this API family
// uses — re-adding an already-excluded IP is not documented as an error,
// consistent with every other additive action (`add`) in this package.
func (c *Client) AddExcludedIPs(ctx context.Context, in AddExcludedIPsInput) error {
	ips, err := FormatIPSet(in.IPs)
	if err != nil {
		return err
	}
	if ips == "" {
		return fmt.Errorf("qualys vmdr: at least one IP is required")
	}

	params := url.Values{}
	params.Set("ips", ips)
	setIfNotEmpty(params, "comment", in.Comment)
	if in.ExpiryDays > 0 {
		params.Set("expiry_days", fmt.Sprint(in.ExpiryDays))
	}
	setListIfNotEmpty(params, "dg_names", in.AssetGroupNames)
	setIfNotEmpty(params, "network_id", in.NetworkID)

	return c.do(ctx, request{
		capability: capAssetExcluded,
		path:       "asset/excluded_ip/",
		action:     "add",
		params:     params,
	}, nil)
}

// RemoveExcludedIPs re-enables scanning for IPs previously excluded.
func (c *Client) RemoveExcludedIPs(ctx context.Context, ips []string, comment, networkID string) error {
	formatted, err := FormatIPSet(ips)
	if err != nil {
		return err
	}
	if formatted == "" {
		return fmt.Errorf("qualys vmdr: at least one IP is required")
	}

	params := url.Values{}
	params.Set("ips", formatted)
	setIfNotEmpty(params, "comment", comment)
	setIfNotEmpty(params, "network_id", networkID)

	return c.do(ctx, request{
		capability: capAssetExcluded,
		path:       "asset/excluded_ip/",
		action:     "remove",
		params:     params,
	}, nil)
}

// RemoveAllExcludedIPs clears every exclusion (optionally scoped to a
// network). Confirmed that `ips` is invalid on this specific action — it
// selects everything by construction, not by an IP list.
func (c *Client) RemoveAllExcludedIPs(ctx context.Context, comment, networkID string) error {
	params := url.Values{}
	setIfNotEmpty(params, "comment", comment)
	setIfNotEmpty(params, "network_id", networkID)

	return c.do(ctx, request{
		capability:    capAssetExcluded,
		path:          "asset/excluded_ip/",
		action:        "remove_all",
		params:        params,
		nonIdempotent: true, // clears every exclusion subscription- or network-wide
	}, nil)
}
