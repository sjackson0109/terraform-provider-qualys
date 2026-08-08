package vmdr

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Provisioning a virtual scanner spans two systems, and only the first half is
// this provider's job:
//
//  1. Configure the appliance in Qualys (this API) — returns an activation code
//  2. Deploy the appliance VM (vSphere, AWS, Azure, GCP, ...) using that code
//  3. The VM personalises itself with the code and connects
//  4. Verify readiness by polling the appliance's status
//
// Steps 2 and 3 belong to an infrastructure provider. This resource covers step
// 1 and exports the activation code for step 2 to consume.

// Appliance is a scanner appliance.
type Appliance struct {
	ID       string
	UUID     string
	Name     string
	Status   string // Online or Offline
	Type     string
	Version  string
	Comments string

	NetworkID       string
	PollingInterval int

	// ActivationCode is the personalisation code used when deploying the
	// appliance VM.
	//
	// Qualys only returns it while the appliance has not yet been activated;
	// afterwards the element is absent. Callers must not treat an empty value as
	// "the code was removed" — see the resource's read logic.
	ActivationCode string

	HeartbeatsMissed string
	LastUpdated      string

	VulnSigsVersion string
	VulnSigsLatest  string
	MLVersion       string
	MLLatest        string

	RunningScanCount int
}

// UpToDate reports whether the appliance's signature and module versions match
// the latest Qualys has published. Both comparisons need the *_LATEST fields,
// which the list output provides, so this needs no extra call.
func (a *Appliance) UpToDate() bool {
	sigsOK := a.VulnSigsLatest == "" || a.VulnSigsVersion == a.VulnSigsLatest
	mlOK := a.MLLatest == "" || a.MLVersion == a.MLLatest
	return sigsOK && mlOK
}

// Online reports whether Qualys currently considers the appliance reachable.
func (a *Appliance) Online() bool {
	return strings.EqualFold(strings.TrimSpace(a.Status), "Online")
}

type applianceListOutput struct {
	Response struct {
		Appliances []struct {
			ID               string `xml:"ID"`
			UUID             string `xml:"UUID"`
			Name             string `xml:"NAME"`
			Status           string `xml:"STATUS"`
			Type             string `xml:"TYPE"`
			SoftwareVersion  string `xml:"SOFTWARE_VERSION"`
			Comments         string `xml:"COMMENTS"`
			NetworkID        string `xml:"NETWORK_ID"`
			PollingInterval  string `xml:"POLLING_INTERVAL"`
			ActivationCode   string `xml:"ACTIVATION_CODE"`
			HeartbeatsMissed string `xml:"HEARTBEATS_MISSED"`
			LastUpdated      string `xml:"LAST_UPDATED_DATE"`
			VulnSigsVersion  string `xml:"VULNSIGS_VERSION"`
			VulnSigsLatest   string `xml:"VULNSIGS_LATEST"`
			MLVersion        string `xml:"ML_VERSION"`
			MLLatest         string `xml:"ML_LATEST"`
			RunningScanCount string `xml:"RUNNING_SCAN_COUNT"`
		} `xml:"APPLIANCE_LIST>APPLIANCE"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *applianceListOutput) warning() *warning { return o.Response.Warning }

func (o *applianceListOutput) appliances() []*Appliance {
	out := make([]*Appliance, 0, len(o.Response.Appliances))
	for i := range o.Response.Appliances {
		a := &o.Response.Appliances[i]
		app := &Appliance{
			ID:               strings.TrimSpace(a.ID),
			UUID:             strings.TrimSpace(a.UUID),
			Name:             a.Name,
			Status:           strings.TrimSpace(a.Status),
			Type:             a.Type,
			Version:          a.SoftwareVersion,
			Comments:         a.Comments,
			NetworkID:        strings.TrimSpace(a.NetworkID),
			ActivationCode:   strings.TrimSpace(a.ActivationCode),
			HeartbeatsMissed: strings.TrimSpace(a.HeartbeatsMissed),
			LastUpdated:      a.LastUpdated,
			VulnSigsVersion:  a.VulnSigsVersion,
			VulnSigsLatest:   a.VulnSigsLatest,
			MLVersion:        a.MLVersion,
			MLLatest:         a.MLLatest,
		}
		app.PollingInterval, _ = strconv.Atoi(strings.TrimSpace(a.PollingInterval))
		app.RunningScanCount, _ = strconv.Atoi(strings.TrimSpace(a.RunningScanCount))
		out = append(out, app)
	}
	return out
}

// applianceCreateOutput carries the activation code returned at creation.
type applianceCreateOutput struct {
	Response struct {
		Text           string `xml:"TEXT"`
		ID             string `xml:"ID"`
		Name           string `xml:"NAME"`
		ActivationCode string `xml:"ACTIVATION_CODE"`
		Remaining      string `xml:"REMAINING_QVSA_LICENSES"`
	} `xml:"RESPONSE"`
}

// VirtualScannerInput is the desired state of a virtual scanner.
type VirtualScannerInput struct {
	Name string
	// PollingInterval in seconds, 60–360. Zero uses the Qualys default of 180.
	PollingInterval int
	// AssetGroupID is required for Unit Manager and Scanner accounts, and must be
	// omitted for Manager accounts — Qualys rejects it from a Manager.
	AssetGroupID string
	Comments     string
}

// VirtualScannerCreated is the result of creating a virtual scanner.
type VirtualScannerCreated struct {
	ID   string
	Name string
	// ActivationCode is the personalisation code for deploying the appliance VM.
	// It is returned here and, until the appliance activates, by the list action.
	ActivationCode    string
	RemainingLicenses string
}

// CreateVirtualScanner registers a virtual scanner and returns its activation
// code.
//
// Creating one consumes a QVSA licence, which is why the call is treated as
// non-idempotent and never retried automatically after dispatch.
func (c *Client) CreateVirtualScanner(ctx context.Context, in VirtualScannerInput) (*VirtualScannerCreated, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("qualys vmdr: virtual scanner name is required")
	}
	if in.PollingInterval != 0 && (in.PollingInterval < 60 || in.PollingInterval > 360) {
		return nil, fmt.Errorf("qualys vmdr: polling_interval must be between 60 and 360 seconds, got %d",
			in.PollingInterval)
	}

	params := url.Values{}
	params.Set("name", in.Name)
	if in.PollingInterval > 0 {
		params.Set("polling_interval", strconv.Itoa(in.PollingInterval))
	}
	setIfNotEmpty(params, "asset_group_id", in.AssetGroupID)
	setIfNotEmpty(params, "comment", in.Comments)

	var out applianceCreateOutput
	if err := c.do(ctx, request{
		capability:    capAppliance,
		path:          "appliance/",
		action:        "create",
		params:        params,
		nonIdempotent: true, // consumes a QVSA licence; must not be repeated blindly
	}, &out); err != nil {
		return nil, err
	}

	id := strings.TrimSpace(out.Response.ID)
	if id == "" {
		return nil, fmt.Errorf("qualys vmdr: virtual scanner was created but the API returned "+
			"no ID (response: %q); import it to bring it under management", out.Response.Text)
	}
	return &VirtualScannerCreated{
		ID:                id,
		Name:              strings.TrimSpace(out.Response.Name),
		ActivationCode:    strings.TrimSpace(out.Response.ActivationCode),
		RemainingLicenses: strings.TrimSpace(out.Response.Remaining),
	}, nil
}

// ApplianceUpdate is the mutable configuration of an appliance.
type ApplianceUpdate struct {
	Name            string
	Comments        string
	PollingInterval int

	// VLANs and Routes replace the existing lists in full. Passing an empty slice
	// clears them; the API has no add/remove variants for these.
	VLANs  []VLAN
	Routes []StaticRoute

	// SetVLANs and SetRoutes control whether those lists are sent at all, so a
	// caller that does not manage them leaves them untouched.
	SetVLANs  bool
	SetRoutes bool
}

// VLAN is a VLAN configured on an appliance.
type VLAN struct {
	ID      string
	IP      string
	Netmask string
	Name    string
}

func (v VLAN) validate() error {
	return checkDelimiters("vlan", v.ID, v.IP, v.Netmask, v.Name)
}

func (v VLAN) encode() string {
	return strings.Join([]string{v.ID, v.IP, v.Netmask, v.Name}, "|")
}

// StaticRoute is a static route configured on an appliance.
type StaticRoute struct {
	IP      string
	Netmask string
	Gateway string
	Name    string
}

func (r StaticRoute) validate() error {
	return checkDelimiters("static route", r.IP, r.Netmask, r.Gateway, r.Name)
}

func (r StaticRoute) encode() string {
	return strings.Join([]string{r.IP, r.Netmask, r.Gateway, r.Name}, "|")
}

// checkDelimiters rejects field values containing the wire format's own
// separators. The API takes VLANs and routes as pipe-delimited fields joined by
// commas; a name containing either character would split into extra fields and
// misconfigure the appliance's networking rather than fail cleanly.
func checkDelimiters(kind string, fields ...string) error {
	for _, f := range fields {
		if strings.ContainsAny(f, ",|") {
			return fmt.Errorf("qualys vmdr: %s field %q must not contain ',' or '|'; "+
				"the API uses those as field separators", kind, f)
		}
	}
	return nil
}

// UpdateAppliance applies configuration to an appliance.
func (c *Client) UpdateAppliance(ctx context.Context, id string, up ApplianceUpdate) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys vmdr: appliance id is required for update")
	}
	if up.PollingInterval != 0 && (up.PollingInterval < 60 || up.PollingInterval > 360) {
		return fmt.Errorf("qualys vmdr: polling_interval must be between 60 and 360 seconds, got %d",
			up.PollingInterval)
	}

	params := url.Values{}
	params.Set("id", id)
	setIfNotEmpty(params, "name", up.Name)
	setIfNotEmpty(params, "comment", up.Comments)
	if up.PollingInterval > 0 {
		params.Set("polling_interval", strconv.Itoa(up.PollingInterval))
	}

	// set_vlans and set_routes replace the whole list; "" clears it. They are
	// only sent when the caller manages them, so an appliance whose VLANs are
	// configured elsewhere is not silently wiped.
	if up.SetVLANs {
		encoded := make([]string, 0, len(up.VLANs))
		for _, v := range up.VLANs {
			if err := v.validate(); err != nil {
				return err
			}
			encoded = append(encoded, v.encode())
		}
		params.Set("set_vlans", strings.Join(encoded, ","))
	}
	if up.SetRoutes {
		encoded := make([]string, 0, len(up.Routes))
		for _, r := range up.Routes {
			if err := r.validate(); err != nil {
				return err
			}
			encoded = append(encoded, r.encode())
		}
		params.Set("set_routes", strings.Join(encoded, ","))
	}

	return c.do(ctx, request{
		capability: capAppliance,
		path:       "appliance/",
		action:     "update",
		params:     params,
	}, nil)
}

// DeleteVirtualScanner removes a virtual scanner.
//
// Only virtual scanners can be deleted, and not while they are running scans.
// Deletion has side effects Qualys applies silently: the appliance is removed
// from any asset groups, and schedules that referenced it are deactivated.
func (c *Client) DeleteVirtualScanner(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("id", id)
	return c.do(ctx, request{
		capability:    capAppliance,
		path:          "appliance/",
		action:        "delete",
		params:        params,
		nonIdempotent: true,
	}, nil)
}

// AssignApplianceToNetwork moves an appliance to a network.
//
// This is a separate action, not a parameter on create or update, and requires
// Manager privileges. An appliance belongs to exactly one network.
func (c *Client) AssignApplianceToNetwork(ctx context.Context, applianceID, networkID string) error {
	params := url.Values{}
	params.Set("appliance_id", applianceID)
	params.Set("network_id", networkID)
	return c.do(ctx, request{
		capability: capAppliance,
		path:       "appliance/",
		action:     "assign_network_id",
		params:     params,
	}, nil)
}

// GetAppliance returns one appliance, or ErrNotFound.
func (c *Client) GetAppliance(ctx context.Context, id string) (*Appliance, error) {
	appliances, err := c.ListAppliances(ctx, ApplianceFilter{IDs: []string{id}})
	if err != nil {
		return nil, err
	}
	for _, a := range appliances {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, fmt.Errorf("qualys vmdr: appliance %s: %w", id, ErrNotFound)
}

// ApplianceFilter narrows an appliance list request.
type ApplianceFilter struct {
	IDs       []string
	Name      string
	NetworkID string
	// Type is physical, virtual or offline.
	Type string
}

// ListAppliances returns appliances matching filter, with full detail.
func (c *Client) ListAppliances(ctx context.Context, filter ApplianceFilter) ([]*Appliance, error) {
	params := url.Values{}
	setListIfNotEmpty(params, "ids", filter.IDs)
	setIfNotEmpty(params, "name", filter.Name)
	setIfNotEmpty(params, "network_id", filter.NetworkID)
	setIfNotEmpty(params, "type", filter.Type)
	// Full output carries the activation code, version and heartbeat fields the
	// resource needs; the brief form would require a second call.
	params.Set("output_mode", "full")

	var all []*Appliance
	err := c.listAll(ctx, request{
		capability: capAppliance,
		path:       "appliance/",
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(applianceListOutput) },
		func(p paginated) error {
			all = append(all, p.(*applianceListOutput).appliances()...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}
