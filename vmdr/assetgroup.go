package vmdr

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// AssetGroup is a named collection of scan targets.
type AssetGroup struct {
	ID       string
	Title    string
	OwnerID  string
	UnitID   string
	Comments string

	// NetworkID is fixed at creation. The edit action exposes no set_network_id,
	// so moving a group between networks is not possible through this API.
	NetworkID string

	IPs          []string
	Domains      []string
	DNSNames     []string
	NetBIOSNames []string
	ApplianceIDs []string

	DefaultApplianceID string

	Division       string
	Function       string
	Location       string
	BusinessImpact string

	CVSSEnviroCDP string
	CVSSEnviroTD  string
	CVSSEnviroCR  string
	CVSSEnviroIR  string
	CVSSEnviroAR  string

	LastUpdate string
}

// AssetGroupInput is the desired state of a group.
//
// Terraform is authoritative over the member lists, so Update sends the set_*
// form of every list. The API also offers add_*/remove_* for additive callers;
// those are intentionally unused here, since converging on declared state is the
// whole point of a Terraform resource.
type AssetGroupInput struct {
	Title    string
	Comments string

	// NetworkID applies to creation only and is ignored on update.
	NetworkID string

	IPs          []string
	Domains      []string
	DNSNames     []string
	NetBIOSNames []string
	ApplianceIDs []string

	DefaultApplianceID string

	Division       string
	Function       string
	Location       string
	BusinessImpact string

	CVSSEnviroCDP string
	CVSSEnviroTD  string
	CVSSEnviroCR  string
	CVSSEnviroIR  string
	CVSSEnviroAR  string
}

type assetGroupListOutput struct {
	Response struct {
		AssetGroups []struct {
			ID       string `xml:"ID"`
			Title    string `xml:"TITLE"`
			OwnerID  string `xml:"OWNER_ID"`
			UnitID   string `xml:"UNIT_ID"`
			Comments string `xml:"COMMENTS"`

			NetworkIDs string `xml:"NETWORK_IDS"`

			IPSet        *ipSet   `xml:"IP_SET"`
			DomainList   []string `xml:"DOMAIN_LIST>DOMAIN"`
			DNSList      []string `xml:"DNS_LIST>DNS"`
			NetBIOSList  []string `xml:"NETBIOS_LIST>NETBIOS"`
			ApplianceIDs string   `xml:"APPLIANCE_IDS"`

			DefaultApplianceID string `xml:"DEFAULT_APPLIANCE_ID"`

			Division       string `xml:"DIVISION"`
			Function       string `xml:"FUNCTION"`
			Location       string `xml:"LOCATION"`
			BusinessImpact string `xml:"BUSINESS_IMPACT"`

			CVSSEnviro struct {
				CDP string `xml:"CDP"`
				TD  string `xml:"TD"`
				CR  string `xml:"CR"`
				IR  string `xml:"IR"`
				AR  string `xml:"AR"`
			} `xml:"CVSS"`

			LastUpdate string `xml:"LAST_UPDATE"`
		} `xml:"ASSET_GROUP_LIST>ASSET_GROUP"`

		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *assetGroupListOutput) warning() *warning { return o.Response.Warning }

func (o *assetGroupListOutput) groups() []*AssetGroup {
	out := make([]*AssetGroup, 0, len(o.Response.AssetGroups))
	for i := range o.Response.AssetGroups {
		g := &o.Response.AssetGroups[i]
		out = append(out, &AssetGroup{
			ID:                 g.ID,
			Title:              g.Title,
			OwnerID:            g.OwnerID,
			UnitID:             g.UnitID,
			Comments:           g.Comments,
			NetworkID:          firstCSV(g.NetworkIDs),
			IPs:                g.IPSet.entries(),
			Domains:            trimAll(g.DomainList),
			DNSNames:           trimAll(g.DNSList),
			NetBIOSNames:       trimAll(g.NetBIOSList),
			ApplianceIDs:       splitCSV(g.ApplianceIDs),
			DefaultApplianceID: strings.TrimSpace(g.DefaultApplianceID),
			Division:           g.Division,
			Function:           g.Function,
			Location:           g.Location,
			BusinessImpact:     g.BusinessImpact,
			CVSSEnviroCDP:      g.CVSSEnviro.CDP,
			CVSSEnviroTD:       g.CVSSEnviro.TD,
			CVSSEnviroCR:       g.CVSSEnviro.CR,
			CVSSEnviroIR:       g.CVSSEnviro.IR,
			CVSSEnviroAR:       g.CVSSEnviro.AR,
			LastUpdate:         g.LastUpdate,
		})
	}
	return out
}

// CreateAssetGroup creates a group and returns its ID.
//
// The add action takes bare parameter names (ips, domains, title). The edit
// action takes prefixed ones (set_ips, set_title). They are not the same
// parameter set under a different action, so the two are encoded separately.
func (c *Client) CreateAssetGroup(ctx context.Context, in AssetGroupInput) (string, error) {
	if strings.TrimSpace(in.Title) == "" {
		return "", fmt.Errorf("qualys vmdr: asset group title is required")
	}

	params := url.Values{}
	params.Set("title", in.Title)
	setIfNotEmpty(params, "network_id", in.NetworkID)
	setIfNotEmpty(params, "comments", in.Comments)
	setIfNotEmpty(params, "division", in.Division)
	setIfNotEmpty(params, "function", in.Function)
	setIfNotEmpty(params, "location", in.Location)
	setIfNotEmpty(params, "business_impact", in.BusinessImpact)
	setIfNotEmpty(params, "default_appliance_id", in.DefaultApplianceID)
	setIfNotEmpty(params, "cvss_enviro_cdp", in.CVSSEnviroCDP)
	setIfNotEmpty(params, "cvss_enviro_td", in.CVSSEnviroTD)
	setIfNotEmpty(params, "cvss_enviro_cr", in.CVSSEnviroCR)
	setIfNotEmpty(params, "cvss_enviro_ir", in.CVSSEnviroIR)
	setIfNotEmpty(params, "cvss_enviro_ar", in.CVSSEnviroAR)

	if len(in.IPs) > 0 {
		ips, err := FormatIPSet(in.IPs)
		if err != nil {
			return "", err
		}
		params.Set("ips", ips)
	}
	setListIfNotEmpty(params, "domains", in.Domains)
	setListIfNotEmpty(params, "dns_names", in.DNSNames)
	setListIfNotEmpty(params, "netbios_names", in.NetBIOSNames)
	setListIfNotEmpty(params, "appliance_ids", in.ApplianceIDs)

	var out simpleReturn
	err := c.do(ctx, request{
		capability: capAssetGroup,
		path:       "asset/group/",
		action:     "add",
		params:     params,
	}, &out)
	if err != nil {
		return "", err
	}

	id, ok := out.itemValue("ID")
	if !ok || strings.TrimSpace(id) == "" {
		// Without an ID we cannot record the resource, and the group may well have
		// been created — say so rather than implying nothing happened.
		return "", fmt.Errorf("qualys vmdr: asset group was created but the API returned no ID "+
			"(response: %q); import it by title to bring it under management", out.Response.Text)
	}
	return strings.TrimSpace(id), nil
}

// UpdateAssetGroup applies desired state to an existing group.
//
// Every member list is sent in its set_* (authoritative) form, including when
// empty: a list the practitioner has emptied must be emptied server-side, which
// the API supports and documents as able to leave the group with no members.
func (c *Client) UpdateAssetGroup(ctx context.Context, id string, in AssetGroupInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys vmdr: asset group id is required for update")
	}

	params := url.Values{}
	params.Set("id", id)
	params.Set("set_title", in.Title)
	params.Set("set_comments", in.Comments)
	params.Set("set_division", in.Division)
	params.Set("set_function", in.Function)
	params.Set("set_location", in.Location)
	setIfNotEmpty(params, "set_business_impact", in.BusinessImpact)
	params.Set("set_default_appliance_id", in.DefaultApplianceID)
	setIfNotEmpty(params, "set_cvss_enviro_cdp", in.CVSSEnviroCDP)
	setIfNotEmpty(params, "set_cvss_enviro_td", in.CVSSEnviroTD)
	setIfNotEmpty(params, "set_cvss_enviro_cr", in.CVSSEnviroCR)
	setIfNotEmpty(params, "set_cvss_enviro_ir", in.CVSSEnviroIR)
	setIfNotEmpty(params, "set_cvss_enviro_ar", in.CVSSEnviroAR)

	ips, err := FormatIPSet(in.IPs)
	if err != nil {
		return err
	}
	params.Set("set_ips", ips)
	params.Set("set_domains", strings.Join(trimAll(in.Domains), ","))
	params.Set("set_dns_names", strings.Join(trimAll(in.DNSNames), ","))
	params.Set("set_netbios_names", strings.Join(trimAll(in.NetBIOSNames), ","))
	params.Set("set_appliance_ids", strings.Join(trimAll(in.ApplianceIDs), ","))

	return c.do(ctx, request{
		capability: capAssetGroup,
		path:       "asset/group/",
		action:     "edit",
		params:     params,
	}, nil)
}

// DeleteAssetGroup removes a group. Member hosts are not affected.
func (c *Client) DeleteAssetGroup(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("id", id)
	return c.do(ctx, request{
		capability:  capAssetGroup,
		path:        "asset/group/",
		action:      "delete",
		params:      params,
		destructive: true,
	}, nil)
}

// GetAssetGroup returns a single group, or ErrNotFound.
//
// An empty result is the unambiguous signal that the group is gone, so it is
// preferred over matching an error code whose numeric value is not yet verified.
func (c *Client) GetAssetGroup(ctx context.Context, id string) (*AssetGroup, error) {
	groups, err := c.ListAssetGroups(ctx, AssetGroupFilter{IDs: []string{id}})
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if g.ID == id {
			return g, nil
		}
	}
	return nil, fmt.Errorf("qualys vmdr: asset group %s: %w", id, ErrNotFound)
}

// AssetGroupFilter narrows a list request.
type AssetGroupFilter struct {
	IDs        []string
	NetworkIDs []string
	UnitID     string
	UserID     string
	// TruncationLimit caps records per page; zero uses the server default.
	// Every page is followed regardless, so this only tunes page size.
	TruncationLimit int
}

// ListAssetGroups returns all groups matching filter, following truncation.
func (c *Client) ListAssetGroups(ctx context.Context, filter AssetGroupFilter) ([]*AssetGroup, error) {
	params := url.Values{}
	setListIfNotEmpty(params, "ids", filter.IDs)
	setListIfNotEmpty(params, "network_ids", filter.NetworkIDs)
	setIfNotEmpty(params, "unit_id", filter.UnitID)
	setIfNotEmpty(params, "user_id", filter.UserID)
	if filter.TruncationLimit > 0 {
		params.Set("truncation_limit", strconv.Itoa(filter.TruncationLimit))
	}
	// Request every attribute; the resource reads most of them and a second
	// round-trip to fetch the rest would be wasteful.
	params.Set("show_attributes", "All")

	var all []*AssetGroup
	err := c.listAll(ctx, request{
		capability: capAssetGroup,
		path:       "asset/group/",
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(assetGroupListOutput) },
		func(p paginated) error {
			all = append(all, p.(*assetGroupListOutput).groups()...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}

func setIfNotEmpty(v url.Values, key, val string) {
	if strings.TrimSpace(val) != "" {
		v.Set(key, val)
	}
}

func setListIfNotEmpty(v url.Values, key string, vals []string) {
	t := trimAll(vals)
	if len(t) > 0 {
		v.Set(key, strings.Join(t, ","))
	}
}

func trimAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return trimAll(strings.Split(s, ","))
}

func firstCSV(s string) string {
	parts := splitCSV(s)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
