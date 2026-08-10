package vmdr

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// VMFinding is one Qualys VM vulnerability finding: one affected host, one
// QID, one detection instance — distinguished by port/protocol where Qualys
// itself reports them as distinct detections. This is deliberately NOT a
// per-host summary (see HostDetection for that, which this file's client
// leaves untouched) but an individual vulnerability instance: "one affected
// host + one QID + one vulnerability instance," per the requirement this
// type was built against. Findings are never aggregated by host or by QID.
//
// Evidence tier: the DETECTION_LIST schema this file decodes (QID, TYPE,
// SEVERITY, PORT, PROTOCOL, SSL, RESULTS, STATUS, FIRST_FOUND_DATETIME,
// LAST_FOUND_DATETIME, LAST_TEST_DATETIME, LAST_UPDATE_DATETIME,
// LAST_FIXED_DATETIME, TIMES_FOUND, IS_IGNORED, IS_DISABLED) is Corroborated,
// not Confirmed: it reflects Qualys's long-published Host List Detection
// API v2 schema (host_list_vm_detection_output.dtd), which this session
// could not fetch directly (docs.qualys.com is blocked in this sandbox) or
// verify against a captured tenant response. FIRST_FOUND_DATETIME and
// LAST_FOUND_DATETIME specifically are two of the three field names doc 03
// already records as Confirmed for this endpoint (the third,
// LAST_SCAN_DATETIME, is the host-level field HostDetection already uses),
// which corroborates this file's per-detection date fields more than an
// unsupported guess would carry — but this has not been verified against a
// tenant. hostdetection.go's own doc comment already flags DETECTION_LIST
// as unconfirmed for exactly this reason; this file resolves that gap using
// the best evidence available, not a fetched or user-supplied source, and
// should be verified against a tenant or a captured API response before
// being relied on in production. Two fields this session found no
// confident evidence for were deliberately left out rather than guessed:
// a per-detection FQDN and a SERVICE element.
type VMFinding struct {
	// Asset fields (host-level; the same set HostDetection already exposes).
	HostID         string
	IP             string
	DNS            string
	NetBIOS        string
	OS             string
	TrackingMethod string

	// Finding identity.
	QID string
	// Port and Protocol identify the detection instance for a port-based
	// finding. HasPort is false when Qualys reports no port for this
	// detection (a host-based, not port-based, vulnerability) — Port is then
	// meaningless and excluded from FindingKey.
	Port     int
	HasPort  bool
	Protocol string

	// Vulnerability fields.
	Severity int
	Status   string
	Type     string
	SSL      bool
	// Results may contain technical detail from the target (banners,
	// response fragments); treat as potentially sensitive. See the
	// data source's documentation for handling guidance.
	Results string

	// Detection lifecycle fields.
	FirstFoundDatetime string
	LastFoundDatetime  string
	LastTestDatetime   string
	LastUpdateDatetime string
	LastFixedDatetime  string
	TimesFound         int

	IsIgnored  bool
	IsDisabled bool
}

// FindingKey is a stable, deterministic finding identity:
// "<host_id>:<qid>:<PROTOCOL>:<port>", or "<host_id>:<qid>" when this
// detection carries no port/protocol. Never derived from list position or
// randomness, so it is stable across repeated reads of unchanged data.
func (f *VMFinding) FindingKey() string {
	if f.HasPort || strings.TrimSpace(f.Protocol) != "" {
		return fmt.Sprintf("%s:%s:%s:%d", f.HostID, f.QID, strings.ToUpper(f.Protocol), f.Port)
	}
	return fmt.Sprintf("%s:%s", f.HostID, f.QID)
}

// VMFindingConflict records two findings that decoded to the same
// FindingKey but disagree on their other fields — a signal that the
// identity scheme has a gap, rather than a true duplicate. Exact duplicates
// (identical in every field) are collapsed silently; this is only raised
// for a genuine conflict, so a caller can surface it rather than silently
// picking one side.
type VMFindingConflict struct {
	Key   string
	First *VMFinding
	Other *VMFinding
}

type vmDetectionWire struct {
	QID                string `xml:"QID"`
	Type               string `xml:"TYPE"`
	Severity           string `xml:"SEVERITY"`
	Port               string `xml:"PORT"`
	Protocol           string `xml:"PROTOCOL"`
	SSL                string `xml:"SSL"`
	Results            string `xml:"RESULTS"`
	Status             string `xml:"STATUS"`
	FirstFoundDatetime string `xml:"FIRST_FOUND_DATETIME"`
	LastFoundDatetime  string `xml:"LAST_FOUND_DATETIME"`
	LastTestDatetime   string `xml:"LAST_TEST_DATETIME"`
	LastUpdateDatetime string `xml:"LAST_UPDATE_DATETIME"`
	LastFixedDatetime  string `xml:"LAST_FIXED_DATETIME"`
	TimesFound         string `xml:"TIMES_FOUND"`
	IsIgnored          string `xml:"IS_IGNORED"`
	IsDisabled         string `xml:"IS_DISABLED"`
}

type vmFindingListOutput struct {
	Response struct {
		Hosts []struct {
			ID             string            `xml:"ID"`
			IP             string            `xml:"IP"`
			TrackingMethod string            `xml:"TRACKING_METHOD"`
			DNS            string            `xml:"DNS"`
			NetBIOS        string            `xml:"NETBIOS"`
			OS             string            `xml:"OS"`
			Detections     []vmDetectionWire `xml:"DETECTION_LIST>DETECTION"`
		} `xml:"HOST_LIST>HOST"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *vmFindingListOutput) warning() *warning { return o.Response.Warning }

// atoiSafe parses a Qualys XML integer element permissively: an absent or
// empty element is 0, not an error, since Go's XML decoder already leaves a
// genuinely absent element at its zero value and Qualys detections
// legitimately omit fields like PORT for host-based findings.
func atoiSafe(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// boolFromFlag interprets a Qualys "0"/"1" flag element. Anything other
// than "1" (including absent) is false.
func boolFromFlag(s string) bool { return strings.TrimSpace(s) == "1" }

func (o *vmFindingListOutput) findings() []*VMFinding {
	var out []*VMFinding
	for i := range o.Response.Hosts {
		h := &o.Response.Hosts[i]
		hostID := strings.TrimSpace(h.ID)
		for j := range h.Detections {
			d := &h.Detections[j]
			port := atoiSafe(d.Port)
			out = append(out, &VMFinding{
				HostID:         hostID,
				IP:             strings.TrimSpace(h.IP),
				DNS:            h.DNS,
				NetBIOS:        h.NetBIOS,
				OS:             h.OS,
				TrackingMethod: h.TrackingMethod,

				QID:      strings.TrimSpace(d.QID),
				Port:     port,
				HasPort:  port > 0,
				Protocol: strings.TrimSpace(d.Protocol),

				Severity: atoiSafe(d.Severity),
				Status:   strings.TrimSpace(d.Status),
				Type:     strings.TrimSpace(d.Type),
				SSL:      boolFromFlag(d.SSL),
				Results:  d.Results,

				FirstFoundDatetime: strings.TrimSpace(d.FirstFoundDatetime),
				LastFoundDatetime:  strings.TrimSpace(d.LastFoundDatetime),
				LastTestDatetime:   strings.TrimSpace(d.LastTestDatetime),
				LastUpdateDatetime: strings.TrimSpace(d.LastUpdateDatetime),
				LastFixedDatetime:  strings.TrimSpace(d.LastFixedDatetime),
				TimesFound:         atoiSafe(d.TimesFound),

				IsIgnored:  boolFromFlag(d.IsIgnored),
				IsDisabled: boolFromFlag(d.IsDisabled),
			})
		}
	}
	return out
}

// VMFindingFilter narrows a VM findings list request.
//
// HostIDs, IPs and Status are sent as server-side query parameters: ids and
// ips are already Confirmed/used by ListHostDetections against this same
// endpoint, and status is one of doc 03's Confirmed params for this
// endpoint. QIDs and the severity range are deliberately NOT sent as query
// parameters — this session found no confirmed evidence for their exact
// parameter names on this specific endpoint (conventions like "qids"/
// "severities" are well-known elsewhere in the Qualys VM API, but asserting
// one here without confirmation risks a silently-wrong filter if the name
// is off, the class of mistake this provider's discovery notes explicitly
// warn against repeating). Callers apply QID and severity filtering
// themselves after this call returns (see the qualys_vm_findings data
// source for the reference implementation) — always correct regardless of
// the exact wire parameter name, at the cost of transferring more data than
// a confirmed server-side filter would. Asset-group scoping is not a field
// here either, for the same reason; resolve it to IPs via GetAssetGroup/
// ListAssetGroups (already Confirmed) before calling this.
type VMFindingFilter struct {
	HostIDs []string
	IPs     []string
	// Status restricts to these Qualys detection statuses (e.g. "New",
	// "Active", "Fixed", "Re-Opened") — sent verbatim, not validated or
	// translated client-side.
	Status []string

	// TruncationLimit caps records per page; zero uses the server default.
	// A client-level knob only, like the identical field on
	// HostDetectionFilter — not exposed in a Terraform schema.
	TruncationLimit int
}

// ListVMFindings returns individual VM vulnerability findings, following
// truncation across the full result set. Findings decoded from an earlier
// page that are byte-for-byte identical (by FindingKey and every other
// field) to one decoded from a later page are collapsed; a FindingKey
// collision between findings that disagree on other fields is reported in
// the returned conflicts slice rather than silently resolved, so a caller
// can decide how to surface it instead of one side silently winning.
//
// This is read-only: there is no VM finding create/update/delete API. To
// change a finding's state, use the Qualys UI or a dedicated action API
// this package does not (yet) implement.
func (c *Client) ListVMFindings(ctx context.Context, filter VMFindingFilter) ([]*VMFinding, []VMFindingConflict, error) {
	params := url.Values{}
	if len(filter.IPs) > 0 {
		ips, err := FormatIPSet(filter.IPs)
		if err != nil {
			return nil, nil, err
		}
		params.Set("ips", ips)
	}
	setListIfNotEmpty(params, "ids", filter.HostIDs)
	setListIfNotEmpty(params, "status", filter.Status)
	if filter.TruncationLimit > 0 {
		params.Set("truncation_limit", strconv.Itoa(filter.TruncationLimit))
	}

	seen := make(map[string]*VMFinding)
	var ordered []*VMFinding
	var conflicts []VMFindingConflict

	err := c.listAll(ctx, request{
		capability: capHostDetection,
		path:       "asset/host/vm/detection/",
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(vmFindingListOutput) },
		func(p paginated) error {
			for _, f := range p.(*vmFindingListOutput).findings() {
				key := f.FindingKey()
				existing, dup := seen[key]
				if !dup {
					seen[key] = f
					ordered = append(ordered, f)
					continue
				}
				if !sameFinding(existing, f) {
					conflicts = append(conflicts, VMFindingConflict{Key: key, First: existing, Other: f})
				}
				// Exact duplicates are silently collapsed; conflicting ones
				// keep the first-seen record, consistent with "detect...
				// collapse exact duplicates, diagnose real conflicts"
				// rather than either side winning arbitrarily.
			}
			return nil
		}, 0)
	if err != nil {
		return nil, nil, err
	}
	return ordered, conflicts, nil
}

func sameFinding(a, b *VMFinding) bool {
	return *a == *b
}
