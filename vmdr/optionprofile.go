package vmdr

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// VMOptionProfile is a VM scan option profile.
//
// Two API surfaces exist for option profiles: whole-XML export/import on
// /subscription/option_profile/, and field-level CRUD on
// /subscription/option_profile/vm/. This client uses the field-level surface,
// because per-field parameters produce diffs a practitioner can read, whereas
// whole-document import would make every change look like a wholesale
// replacement. Import also rewrites some fields silently (see the notes on
// non-round-tripping fields below), which would be invisible in a plan.
type VMOptionProfile struct {
	ID       string
	Title    string
	Owner    string
	Global   bool
	Default  bool
	Offline  bool
	Modified string

	ScanTCPPorts           string
	ScanTCPPortsAdditional string
	ScanUDPPorts           string
	ScanUDPPortsAdditional string

	ScanDeadHosts        bool
	CloseVulnOnDeadHosts bool
	PurgeHostData        bool

	ScanOverallPerformance string
	ScanPacketDelay        string
	ScanParallelScaling    bool

	VulnerabilityDetection string
	CustomSearchListIDs    []string

	PasswordBruteForcingSystem string

	BasicHostInformationChecks bool
	OVALChecks                 bool
	AllQRDIChecks              bool

	AdditionalCertificateDetection bool
	LoadBalancer                   bool
	EnableDissolvableAgent         bool
	TestAuthentication             bool

	Authentication []string

	// Write-only settings — three_way_handshake, scan_intensity, the scanner
	// process/thread tuning, exclude_search_list_ids and the system-auth trio —
	// have no counterpart in the documented list output, so this read-side type
	// deliberately does not carry them. Advertising fields that are always zero
	// invites callers to trust values that were never read.
}

// VMOptionProfileInput is the desired state of a profile.
type VMOptionProfileInput struct {
	Title   string
	Owner   string
	Global  bool
	Default bool
	Offline bool

	ScanTCPPorts           string
	ScanTCPPortsAdditional string
	ScanUDPPorts           string
	ScanUDPPortsAdditional string
	ThreeWayHandshake      bool

	ScanDeadHosts        bool
	CloseVulnOnDeadHosts bool
	PurgeHostData        bool

	ScanOverallPerformance string
	ScanIntensity          string
	ScanPacketDelay        string
	ScanParallelScaling    bool
	ScanExternalScanners   string
	ScanScannerAppliances  string
	ScanTotalProcess       string
	ScanHTTPProcess        string

	VulnerabilityDetection string
	CustomSearchListIDs    []string
	ExcludeSearchListIDs   []string

	PasswordBruteForcingSystem string

	BasicHostInformationChecks bool
	OVALChecks                 bool
	AllQRDIChecks              bool

	AdditionalCertificateDetection bool
	LoadBalancer                   bool
	EnableDissolvableAgent         bool
	TestAuthentication             bool

	Authentication           []string
	IncludeSystemAuth        bool
	UseSystemAuthOnDuplicate bool
	UseUserAuthOnDuplicate   bool
}

// optionProfileOutput decodes the OPTION_PROFILES document.
//
// Unlike every other list in this package, the VM option-profile list is not
// wrapped in a RESPONSE element: the documented root is OPTION_PROFILES with
// OPTION_PROFILE children, the same shape the export action returns. Both
// placements are accepted here so a nested response still decodes rather than
// silently yielding zero profiles — which would make GetVMOptionProfile report
// ErrNotFound and drop a live resource from state.
type optionProfileOutput struct {
	Response struct {
		Profiles []optionProfileEntry `xml:"OPTION_PROFILE"`
	} `xml:"RESPONSE"`

	Profiles []optionProfileEntry `xml:"OPTION_PROFILE"`
}

type optionProfileEntry struct {
	BasicInfo struct {
		ID        string `xml:"ID"`
		GroupName string `xml:"GROUP_NAME"`
		GroupType string `xml:"GROUP_TYPE"`
		UserID    string `xml:"USER_ID"`
		IsDefault string `xml:"IS_DEFAULT"`
		IsGlobal  string `xml:"IS_GLOBAL"`
		IsOffline string `xml:"IS_OFFLINE_SYNCABLE"`
		UpdatedAt string `xml:"UPDATE_DATE"`
	} `xml:"BASIC_INFO"`

	Scan struct {
		Ports struct {
			TCPPorts struct {
				Type       string `xml:"TCP_PORTS_TYPE"`
				Additional string `xml:"TCP_PORTS_ADDITIONAL"`
			} `xml:"TCP_PORTS"`
			UDPPorts struct {
				Type       string `xml:"UDP_PORTS_TYPE"`
				Additional string `xml:"UDP_PORTS_ADDITIONAL"`
			} `xml:"UDP_PORTS"`
			ThreeWayHandshake string `xml:"STANDARD_SCAN"`
		} `xml:"PORTS"`

		ScanDeadHosts        string `xml:"SCAN_DEAD_HOSTS"`
		CloseVulnOnDeadHosts string `xml:"CLOSE_VULNERABILITIES>CLOSE_VULN_ON_DEAD_HOSTS"`
		PurgeHostData        string `xml:"CLOSE_VULNERABILITIES>PURGE_HOST_DATA"`

		Performance struct {
			OverallPerformance string `xml:"OVERALL_PERFORMANCE"`
			PacketDelay        string `xml:"PACKET_DELAY"`
			ParallelScaling    string `xml:"PARALLEL_SCALING"`
		} `xml:"PERFORMANCE"`

		VulnDetection struct {
			Type              string   `xml:"CUSTOM_LIST>TYPE"`
			CustomListIDs     []string `xml:"CUSTOM_LIST>CUSTOM>ID"`
			DetectionType     string   `xml:"DETECTION_TYPE"`
			BasicHostInfo     string   `xml:"BASIC_HOST_INFO_CHECKS"`
			OVALChecks        string   `xml:"OVAL_CHECKS"`
			AllQRDIChecks     string   `xml:"ALL_QRDI_CHECKS"`
			PasswordBruteType string   `xml:"PASSWORD_BRUTE_FORCING>SYSTEM"`
		} `xml:"VULNERABILITY_DETECTION"`

		Authentication       []string `xml:"AUTHENTICATION>TYPE"`
		AdditionalCertDetect string   `xml:"ADDITIONAL_CERT_DETECTION"`
		DissolvableAgent     string   `xml:"DISSOLVABLE_AGENT>DISSOLVABLE_AGENT_ENABLE"`
		LoadBalancer         string   `xml:"LOAD_BALANCER_DETECTION"`
		TestAuthentication   string   `xml:"TEST_AUTHENTICATION"`
	} `xml:"SCAN"`
}

// entries returns the profiles from whichever placement the response used.
func (o *optionProfileOutput) entries() []optionProfileEntry {
	if len(o.Profiles) > 0 {
		return o.Profiles
	}
	return o.Response.Profiles
}

func (o *optionProfileOutput) profiles() []*VMOptionProfile {
	entries := o.entries()
	out := make([]*VMOptionProfile, 0, len(entries))
	for i := range entries {
		p := &entries[i]
		out = append(out, &VMOptionProfile{
			ID:       strings.TrimSpace(p.BasicInfo.ID),
			Title:    p.BasicInfo.GroupName,
			Owner:    p.BasicInfo.UserID,
			Global:   isTrue(p.BasicInfo.IsGlobal),
			Default:  isTrue(p.BasicInfo.IsDefault),
			Offline:  isTrue(p.BasicInfo.IsOffline),
			Modified: p.BasicInfo.UpdatedAt,

			ScanTCPPorts:           p.Scan.Ports.TCPPorts.Type,
			ScanTCPPortsAdditional: p.Scan.Ports.TCPPorts.Additional,
			ScanUDPPorts:           p.Scan.Ports.UDPPorts.Type,
			ScanUDPPortsAdditional: p.Scan.Ports.UDPPorts.Additional,

			ScanDeadHosts:        isTrue(p.Scan.ScanDeadHosts),
			CloseVulnOnDeadHosts: isTrue(p.Scan.CloseVulnOnDeadHosts),
			PurgeHostData:        isTrue(p.Scan.PurgeHostData),

			ScanOverallPerformance: p.Scan.Performance.OverallPerformance,
			ScanPacketDelay:        p.Scan.Performance.PacketDelay,
			ScanParallelScaling:    isTrue(p.Scan.Performance.ParallelScaling),

			VulnerabilityDetection: firstNonEmpty(p.Scan.VulnDetection.Type, p.Scan.VulnDetection.DetectionType),
			CustomSearchListIDs:    trimAll(p.Scan.VulnDetection.CustomListIDs),

			PasswordBruteForcingSystem: p.Scan.VulnDetection.PasswordBruteType,
			BasicHostInformationChecks: isTrue(p.Scan.VulnDetection.BasicHostInfo),
			OVALChecks:                 isTrue(p.Scan.VulnDetection.OVALChecks),
			AllQRDIChecks:              isTrue(p.Scan.VulnDetection.AllQRDIChecks),

			AdditionalCertificateDetection: isTrue(p.Scan.AdditionalCertDetect),
			LoadBalancer:                   isTrue(p.Scan.LoadBalancer),
			EnableDissolvableAgent:         isTrue(p.Scan.DissolvableAgent),
			TestAuthentication:             isTrue(p.Scan.TestAuthentication),
			Authentication:                 trimAll(p.Scan.Authentication),
		})
	}
	return out
}

// optionProfileParams encodes the fields shared by create and update.
//
// The documented update action accepts the same parameters as create ("For other
// parameters see Create VM Option Profile"), so one encoder serves both.
func optionProfileParams(in VMOptionProfileInput) (url.Values, error) {
	p := url.Values{}
	p.Set("title", in.Title)
	setIfNotEmpty(p, "owner", in.Owner)
	p.Set("global", boolParam(in.Global))
	p.Set("default", boolParam(in.Default))
	p.Set("offline_scanner", boolParam(in.Offline))

	setIfNotEmpty(p, "scan_tcp_ports", in.ScanTCPPorts)
	setIfNotEmpty(p, "scan_tcp_ports_additional", in.ScanTCPPortsAdditional)
	setIfNotEmpty(p, "scan_udp_ports", in.ScanUDPPorts)
	setIfNotEmpty(p, "scan_udp_ports_additional", in.ScanUDPPortsAdditional)
	p.Set("3_way_handshake", boolParam(in.ThreeWayHandshake))

	p.Set("scan_dead_hosts", boolParam(in.ScanDeadHosts))
	p.Set("close_vuln_on_dead_hosts", boolParam(in.CloseVulnOnDeadHosts))
	p.Set("purge_host_data", boolParam(in.PurgeHostData))

	setIfNotEmpty(p, "scan_overall_performance", in.ScanOverallPerformance)
	setIfNotEmpty(p, "scan_intensity", in.ScanIntensity)
	setIfNotEmpty(p, "scan_packet_delay", in.ScanPacketDelay)
	p.Set("scan_parallel_scaling", boolParam(in.ScanParallelScaling))
	setIfNotEmpty(p, "scan_external_scanners", in.ScanExternalScanners)
	setIfNotEmpty(p, "scan_scanner_appliances", in.ScanScannerAppliances)
	setIfNotEmpty(p, "scan_total_process", in.ScanTotalProcess)
	setIfNotEmpty(p, "scan_http_process", in.ScanHTTPProcess)

	if in.VulnerabilityDetection != "" {
		p.Set("vulnerability_detection", in.VulnerabilityDetection)
		if strings.EqualFold(in.VulnerabilityDetection, "custom") && len(in.CustomSearchListIDs) == 0 {
			return nil, fmt.Errorf("qualys vmdr: vulnerability_detection is \"custom\" but no " +
				"custom search list IDs were supplied; Qualys would silently fall back to " +
				"complete detection")
		}
	}
	setListIfNotEmpty(p, "custom_search_list_ids", in.CustomSearchListIDs)
	setListIfNotEmpty(p, "exclude_search_list_ids", in.ExcludeSearchListIDs)

	setIfNotEmpty(p, "password_brute_forcing_system", in.PasswordBruteForcingSystem)

	p.Set("basic_host_information_checks", boolParam(in.BasicHostInformationChecks))
	p.Set("oval_checks", boolParam(in.OVALChecks))
	p.Set("all_qrdi_checks", boolParam(in.AllQRDIChecks))

	p.Set("enable_additional_certificate_detection", boolParam(in.AdditionalCertificateDetection))
	p.Set("load_balancer", boolParam(in.LoadBalancer))
	p.Set("enable_dissolvable_agent", boolParam(in.EnableDissolvableAgent))
	p.Set("test_authentication", boolParam(in.TestAuthentication))

	setListIfNotEmpty(p, "authentication", in.Authentication)
	p.Set("include_system_auth", boolParam(in.IncludeSystemAuth))
	p.Set("use_system_auth_on_duplicate", boolParam(in.UseSystemAuthOnDuplicate))
	p.Set("use_user_auth_on_duplicate", boolParam(in.UseUserAuthOnDuplicate))

	return p, nil
}

// CreateVMOptionProfile creates a profile and returns its ID.
func (c *Client) CreateVMOptionProfile(ctx context.Context, in VMOptionProfileInput) (string, error) {
	if strings.TrimSpace(in.Title) == "" {
		return "", fmt.Errorf("qualys vmdr: option profile title is required")
	}
	params, err := optionProfileParams(in)
	if err != nil {
		return "", err
	}

	var out simpleReturn
	if err := c.do(ctx, request{
		capability:    capOptionProfile,
		path:          "subscription/option_profile/vm/",
		action:        "create",
		params:        params,
		nonIdempotent: true, // creating twice would duplicate the object
	}, &out); err != nil {
		return "", err
	}

	id, ok := out.itemValue("ID")
	if !ok || strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("qualys vmdr: option profile was created but the API returned no ID "+
			"(response: %q); import it to bring it under management", out.Response.Text)
	}
	return strings.TrimSpace(id), nil
}

// UpdateVMOptionProfile applies desired state.
func (c *Client) UpdateVMOptionProfile(ctx context.Context, id string, in VMOptionProfileInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys vmdr: option profile id is required for update")
	}
	params, err := optionProfileParams(in)
	if err != nil {
		return err
	}
	params.Set("id", id)

	return c.do(ctx, request{
		capability: capOptionProfile,
		path:       "subscription/option_profile/vm/",
		action:     "update",
		params:     params,
	}, nil)
}

// DeleteVMOptionProfile removes a profile.
//
// Scans and schedules referencing it are not updated by Qualys, so deleting a
// profile still in use leaves those configurations broken.
func (c *Client) DeleteVMOptionProfile(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("id", id)
	return c.do(ctx, request{
		capability:    capOptionProfile,
		path:          "subscription/option_profile/vm/",
		action:        "delete",
		params:        params,
		nonIdempotent: true,
	}, nil)
}

// GetVMOptionProfile returns one profile, or ErrNotFound.
func (c *Client) GetVMOptionProfile(ctx context.Context, id string) (*VMOptionProfile, error) {
	profiles, err := c.ListVMOptionProfiles(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range profiles {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, fmt.Errorf("qualys vmdr: option profile %s: %w", id, ErrNotFound)
}

// ListVMOptionProfiles returns all VM option profiles.
//
// The list action takes no ID filter, so a single-profile read fetches all and
// selects. Profiles are few enough per subscription that this is cheaper than
// the round-trips a filtered API would save.
func (c *Client) ListVMOptionProfiles(ctx context.Context) ([]*VMOptionProfile, error) {
	var out optionProfileOutput
	if err := c.do(ctx, request{
		capability: capOptionProfile,
		path:       "subscription/option_profile/vm/",
		action:     "list",
		method:     "GET",
	}, &out); err != nil {
		return nil, err
	}
	return out.profiles(), nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
