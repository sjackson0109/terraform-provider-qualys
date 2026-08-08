package vmdr

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// HostDetection is a host as reported by the host detection list API
// (asset/host/vm/detection/), which is the input to a stale-asset review: it
// carries a host's last VM scan time, unlike the plain asset/host/ list.
//
// The response also carries a per-vulnerability DETECTION_LIST (QID, type,
// severity, first/last-found timestamps, results). Those field names are not
// confirmed against official documentation for this specific endpoint (see
// doc 08), so this client deliberately reads only the host-level summary —
// enough to identify stale hosts — rather than guess a per-detection schema.
type HostDetection struct {
	ID               string
	IP               string
	TrackingMethod   string
	DNS              string
	NetBIOS          string
	OS               string
	LastScanDatetime string
}

type hostDetectionListOutput struct {
	Response struct {
		Hosts []struct {
			ID               string `xml:"ID"`
			IP               string `xml:"IP"`
			TrackingMethod   string `xml:"TRACKING_METHOD"`
			DNS              string `xml:"DNS"`
			NetBIOS          string `xml:"NETBIOS"`
			OS               string `xml:"OS"`
			LastScanDatetime string `xml:"LAST_SCAN_DATETIME"`
		} `xml:"HOST_LIST>HOST"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *hostDetectionListOutput) warning() *warning { return o.Response.Warning }

func (o *hostDetectionListOutput) detections() []*HostDetection {
	out := make([]*HostDetection, 0, len(o.Response.Hosts))
	for i := range o.Response.Hosts {
		h := &o.Response.Hosts[i]
		out = append(out, &HostDetection{
			ID:               strings.TrimSpace(h.ID),
			IP:               strings.TrimSpace(h.IP),
			TrackingMethod:   h.TrackingMethod,
			DNS:              h.DNS,
			NetBIOS:          h.NetBIOS,
			OS:               h.OS,
			LastScanDatetime: h.LastScanDatetime,
		})
	}
	return out
}

// HostDetectionFilter narrows a host detection list request.
type HostDetectionFilter struct {
	IPs []string
	IDs []string

	// VMScanDateBefore/After filter on the host's last VM scan time
	// (YYYY-MM-DD[THH:MM:SSZ]), which is how a stale-asset sweep is scoped.
	VMScanDateBefore string
	VMScanDateAfter  string

	TruncationLimit int
}

// ListHostDetections returns hosts with their last scan time, following
// truncation. This is a read-only lookup: there is no corresponding
// create/update/delete API, and purge (asset/host/?action=purge, a different
// endpoint) is the imperative operation that acts on what this identifies.
func (c *Client) ListHostDetections(ctx context.Context, filter HostDetectionFilter) ([]*HostDetection, error) {
	params := url.Values{}
	if len(filter.IPs) > 0 {
		ips, err := FormatIPSet(filter.IPs)
		if err != nil {
			return nil, err
		}
		params.Set("ips", ips)
	}
	setListIfNotEmpty(params, "ids", filter.IDs)
	setIfNotEmpty(params, "vm_scan_date_before", filter.VMScanDateBefore)
	setIfNotEmpty(params, "vm_scan_date_after", filter.VMScanDateAfter)
	if filter.TruncationLimit > 0 {
		params.Set("truncation_limit", strconv.Itoa(filter.TruncationLimit))
	}

	var all []*HostDetection
	err := c.listAll(ctx, request{
		capability: capHostDetection,
		path:       "asset/host/vm/detection/",
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(hostDetectionListOutput) },
		func(p paginated) error {
			all = append(all, p.(*hostDetectionListOutput).detections()...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}
