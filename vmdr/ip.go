package vmdr

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Host assets are registered by adding IPs to the subscription and enabling them
// for a module (VM, PC, and others).
//
// There is no delete. Qualys documents no API action for removing IPs from a
// subscription — support state the omission is deliberate, to avoid an API call
// that could empty an account. The only API-side reduction is purge, which
// removes vulnerability data and drops the asset from AssetView but is a
// different operation with different consequences.
//
// The update action splits selectors from setters, and the distinction is easy
// to get wrong:
//
//	selectors: ips, ids, ag_ids, ag_titles, network_id, tracking_method,
//	           host_dns, host_netbios  — these choose which hosts to act on
//	setters:   new_tracking_method, new_owner, new_ud1..3, new_comment
//
// Sending `owner` instead of `new_owner` silently filters the selection rather
// than setting anything, so this client only ever writes through the new_* form.

// HostAssetUpdate is the set of attributes an update can change.
type HostAssetUpdate struct {
	TrackingMethod string // IP, DNS or NETBIOS. EC2 and AGENT cannot be set.
	Owner          string
	UD1, UD2, UD3  string
	Comment        string
}

// AddIPsInput registers IPs with the subscription.
type AddIPsInput struct {
	// IPs accepts single addresses, hyphenated ranges and CIDR blocks.
	IPs []string

	EnableVM bool
	EnablePC bool

	// TrackingMethod is IP, DNS or NETBIOS.
	TrackingMethod string

	Owner         string
	UD1, UD2, UD3 string
	Comment       string

	// AssetGroupTitle adds the IPs to an existing group as they are registered.
	AssetGroupTitle string
}

// AddIPs registers IPs with the subscription.
func (c *Client) AddIPs(ctx context.Context, in AddIPsInput) error {
	ips, err := FormatIPSet(in.IPs)
	if err != nil {
		return err
	}
	if ips == "" {
		return fmt.Errorf("qualys vmdr: at least one IP is required")
	}
	if !in.EnableVM && !in.EnablePC {
		return fmt.Errorf("qualys vmdr: enable at least one module (VM or PC) when adding IPs; " +
			"an IP registered for no module cannot be scanned")
	}

	params := url.Values{}
	params.Set("ips", ips)
	if in.EnableVM {
		params.Set("enable_vm", "1")
	}
	if in.EnablePC {
		params.Set("enable_pc", "1")
	}
	setIfNotEmpty(params, "tracking_method", in.TrackingMethod)
	setIfNotEmpty(params, "owner", in.Owner)
	setIfNotEmpty(params, "ud1", in.UD1)
	setIfNotEmpty(params, "ud2", in.UD2)
	setIfNotEmpty(params, "ud3", in.UD3)
	setIfNotEmpty(params, "comment", in.Comment)
	setIfNotEmpty(params, "ag_title", in.AssetGroupTitle)

	return c.do(ctx, request{
		capability: capAssetIP,
		path:       "asset/ip/",
		action:     "add",
		params:     params,
	}, nil)
}

// UpdateIPs changes attributes on the hosts selected by ips (and optionally
// network).
//
// Every setter is applied to every matched host, so a broad selection rewrites
// broadly. The caller is expected to pass exactly the addresses it manages.
func (c *Client) UpdateIPs(ctx context.Context, ips []string, networkID string, up HostAssetUpdate) error {
	formatted, err := FormatIPSet(ips)
	if err != nil {
		return err
	}
	if formatted == "" {
		return fmt.Errorf("qualys vmdr: at least one IP is required for update")
	}

	params := url.Values{}
	params.Set("ips", formatted)
	// network_id scopes the selection; it cannot move an IP between networks.
	setIfNotEmpty(params, "network_id", networkID)

	// Setters use the new_* spelling. The bare names are selectors.
	setIfNotEmpty(params, "new_tracking_method", up.TrackingMethod)
	setIfNotEmpty(params, "new_owner", up.Owner)
	setIfNotEmpty(params, "new_ud1", up.UD1)
	setIfNotEmpty(params, "new_ud2", up.UD2)
	setIfNotEmpty(params, "new_ud3", up.UD3)
	setIfNotEmpty(params, "new_comment", up.Comment)

	return c.do(ctx, request{
		capability: capAssetIP,
		path:       "asset/ip/",
		action:     "update",
		params:     params,
	}, nil)
}

// HostAsset is a registered host.
type HostAsset struct {
	ID             string
	IP             string
	TrackingMethod string
	DNS            string
	NetBIOS        string
	OS             string
	Owner          string
	Comments       string
	LastVMScan     string
	NetworkID      string
}

type hostListOutput struct {
	Response struct {
		Hosts []struct {
			ID             string `xml:"ID"`
			IP             string `xml:"IP"`
			TrackingMethod string `xml:"TRACKING_METHOD"`
			DNS            string `xml:"DNS"`
			NetBIOS        string `xml:"NETBIOS"`
			OS             string `xml:"OS"`
			Owner          string `xml:"OWNER"`
			Comments       string `xml:"COMMENTS"`
			LastVMScan     string `xml:"LAST_VM_SCANNED_DATE"`
			NetworkID      string `xml:"NETWORK_ID"`
		} `xml:"HOST_LIST>HOST"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *hostListOutput) warning() *warning { return o.Response.Warning }

func (o *hostListOutput) hosts() []*HostAsset {
	out := make([]*HostAsset, 0, len(o.Response.Hosts))
	for i := range o.Response.Hosts {
		h := &o.Response.Hosts[i]
		out = append(out, &HostAsset{
			ID:             strings.TrimSpace(h.ID),
			IP:             strings.TrimSpace(h.IP),
			TrackingMethod: h.TrackingMethod,
			DNS:            h.DNS,
			NetBIOS:        h.NetBIOS,
			OS:             h.OS,
			Owner:          h.Owner,
			Comments:       h.Comments,
			LastVMScan:     h.LastVMScan,
			NetworkID:      strings.TrimSpace(h.NetworkID),
		})
	}
	return out
}

// HostFilter narrows a host list request.
type HostFilter struct {
	IPs       []string
	IDs       []string
	NetworkID string

	// NoVMScanSince returns hosts not scanned since this date (YYYY-MM-DD),
	// which is how stale assets are identified.
	NoVMScanSince string
	// VMScanSince returns hosts scanned since this date.
	VMScanSince string

	TruncationLimit int
}

// ListHosts returns registered hosts matching filter, following truncation.
func (c *Client) ListHosts(ctx context.Context, filter HostFilter) ([]*HostAsset, error) {
	params := url.Values{}
	if len(filter.IPs) > 0 {
		ips, err := FormatIPSet(filter.IPs)
		if err != nil {
			return nil, err
		}
		params.Set("ips", ips)
	}
	setListIfNotEmpty(params, "ids", filter.IDs)
	setIfNotEmpty(params, "network_id", filter.NetworkID)
	setIfNotEmpty(params, "no_vm_scan_since", filter.NoVMScanSince)
	setIfNotEmpty(params, "vm_scan_since", filter.VMScanSince)
	if filter.TruncationLimit > 0 {
		params.Set("truncation_limit", fmt.Sprint(filter.TruncationLimit))
	}

	var all []*HostAsset
	err := c.listAll(ctx, request{
		capability: capAssetHost,
		path:       "asset/host/",
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(hostListOutput) },
		func(p paginated) error {
			all = append(all, p.(*hostListOutput).hosts()...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}

// PurgeInput selects hosts to purge.
type PurgeInput struct {
	IPs           []string
	IDs           []string
	AssetGroupIDs []string
	NetworkIDs    []string

	// DataScope is "vm", "pc", or "vm,pc". Empty uses the API default.
	//
	// Note: if ComplianceEnabled is set alongside DataScope, Qualys purges both
	// vulnerability and compliance data regardless of what DataScope says.
	DataScope string

	ComplianceEnabled bool

	// NoVMScanSince purges only hosts not scanned since this date, which is how
	// a stale-asset sweep is scoped.
	NoVMScanSince string

	OSPattern string
}

// PurgeHosts deletes host data.
//
// This is destructive and irreversible: it removes vulnerability and compliance
// data, tickets and exceptions, and drops the asset from AssetView. Scan results
// themselves are retained.
//
// It is also asynchronous. The response lists the hosts *queued* for purging, so
// a read immediately afterwards may still return them; verification must poll
// rather than assume completion.
//
// The call is marked destructive, so the client will not retry it after
// dispatch. On an ambiguous failure, re-read rather than calling again.
func (c *Client) PurgeHosts(ctx context.Context, in PurgeInput) error {
	params := url.Values{}
	if len(in.IPs) > 0 {
		ips, err := FormatIPSet(in.IPs)
		if err != nil {
			return err
		}
		params.Set("ips", ips)
	}
	setListIfNotEmpty(params, "ids", in.IDs)
	setListIfNotEmpty(params, "ag_ids", in.AssetGroupIDs)
	setListIfNotEmpty(params, "network_ids", in.NetworkIDs)
	setIfNotEmpty(params, "data_scope", in.DataScope)
	setIfNotEmpty(params, "no_vm_scan_since", in.NoVMScanSince)
	setIfNotEmpty(params, "os_pattern", in.OSPattern)
	if in.ComplianceEnabled {
		params.Set("compliance_enabled", "1")
	}

	if len(params) == 0 {
		return fmt.Errorf("qualys vmdr: purge requires a selector; " +
			"an unselected purge would target the whole subscription")
	}

	return c.do(ctx, request{
		capability:  capAssetHost,
		path:        "asset/host/",
		action:      "purge",
		params:      params,
		destructive: true,
	}, nil)
}
