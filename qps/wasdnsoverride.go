package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// WASDNSMapping is one hostname-to-IP override a WAS DNS override record
// applies during crawling and scanning.
type WASDNSMapping struct {
	HostName  string
	IPAddress string
}

// WASDNSOverride resolves one or more hostnames to user-defined IP
// addresses during WAS crawling and scanning — for scanning dev/UAT
// environments, private applications with no public DNS, or redirecting a
// hostname to an alternative back-end. Can be set as a web application's
// default or referenced by an individual qualys_was_scan_schedule
// (dns_override_id).
type WASDNSOverride struct {
	ID       string
	Name     string
	Mappings []WASDNSMapping
	Comments []string
	Tags     []TagRef
	Created  string
	Updated  string
}

// WASDNSOverrideInput is the desired state of a WAS DNS override record.
//
// Confirmed against a user-supplied walkthrough carrying inline citations
// to specific docs.qualys.com pages (dns_count/search_dns/get_dns_details/
// create_dns/update_dns/delete_dns.htm) — stronger provenance than a plain
// transcription, though still not a verbatim primary-source read. Two real
// deviations from every other WAS object built so far: update takes no ID
// in the URL path (the ID travels inside the request body instead), and
// delete takes no ID in the URL path either (it travels as a filter
// criterion). Mappings and Comments use the same set/add/remove idiom
// already confirmed for tag associations; this package always sends "set"
// (an authoritative replace), matching how Terraform naturally expresses
// "this is the complete desired state" rather than an incremental delta.
type WASDNSOverrideInput struct {
	Name     string
	Mappings []WASDNSMapping
	Comments []string
	TagIDs   []string
}

type dnsMappingWire struct {
	HostName  string `json:"hostName,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

type dnsMappingSetWire struct {
	DnsMapping []dnsMappingWire `json:"DnsMapping,omitempty"`
}

type dnsMappingsWire struct {
	Set  *dnsMappingSetWire `json:"set,omitempty"`
	List *dnsMappingSetWire `json:"list,omitempty"`
}

type dnsCommentsWire struct {
	// Set deliberately has no omitempty: an empty (but non-nil, always
	// allocated by wasDNSOverrideInputToWire) slice must still render as
	// "set":[] on the wire, or clearing comments back to empty could never
	// reach the API — see that function's doc comment. List keeps
	// omitempty; it is decode-only (a response never needs to distinguish
	// "server sent an empty list" from "server sent nothing").
	Set  []string `json:"set"`
	List []string `json:"list,omitempty"`
}

type wasDNSOverrideWire struct {
	ID       json.Number      `json:"id,omitempty"`
	Name     string           `json:"name,omitempty"`
	Mappings *dnsMappingsWire `json:"mappings,omitempty"`
	Comments *dnsCommentsWire `json:"comments,omitempty"`
	Tags     *webAppTagList   `json:"tags,omitempty"`
	Created  string           `json:"createdDate,omitempty"`
	Updated  string           `json:"updatedDate,omitempty"`
}

// wasDNSOverrideData is the "data" envelope. The wrapper key DnsOverride is
// Confirmed — seen verbatim in a user-supplied create/update example.
type wasDNSOverrideData struct {
	DnsOverride *wasDNSOverrideWire `json:"DnsOverride,omitempty"`
}

func dnsMappingsToWire(mappings []WASDNSMapping) *dnsMappingsWire {
	if len(mappings) == 0 {
		return &dnsMappingsWire{Set: &dnsMappingSetWire{}}
	}
	wire := make([]dnsMappingWire, 0, len(mappings))
	for _, m := range mappings {
		wire = append(wire, dnsMappingWire(m))
	}
	return &dnsMappingsWire{Set: &dnsMappingSetWire{DnsMapping: wire}}
}

func wasDNSOverrideInputToWire(in WASDNSOverrideInput) *wasDNSOverrideWire {
	w := &wasDNSOverrideWire{
		Name:     in.Name,
		Mappings: dnsMappingsToWire(in.Mappings),
	}

	// Comments and Tags are always sent, even empty — the same "clearing
	// this back to empty must actually reach the API" reasoning as
	// dnsMappingsToWire's own unconditional allocation just above.
	comments := in.Comments
	if comments == nil {
		comments = []string{}
	}
	w.Comments = &dnsCommentsWire{Set: comments}

	simples := make([]tagSimple, 0, len(in.TagIDs))
	for _, id := range in.TagIDs {
		simples = append(simples, tagSimple{ID: json.Number(id)})
	}
	w.Tags = &webAppTagList{Set: &tagSimpleList{TagSimple: simples}}

	return w
}

func (w *wasDNSOverrideWire) toWASDNSOverride() *WASDNSOverride {
	out := &WASDNSOverride{
		ID:      w.ID.String(),
		Name:    w.Name,
		Created: w.Created,
		Updated: w.Updated,
	}
	if w.Mappings != nil {
		set := w.Mappings.List
		if set == nil {
			set = w.Mappings.Set
		}
		if set != nil {
			for _, m := range set.DnsMapping {
				out.Mappings = append(out.Mappings, WASDNSMapping(m))
			}
		}
	}
	if w.Comments != nil {
		if len(w.Comments.List) > 0 {
			out.Comments = w.Comments.List
		} else {
			out.Comments = w.Comments.Set
		}
	}
	if w.Tags != nil && w.Tags.List != nil {
		for _, t := range w.Tags.List.TagSimple {
			out.Tags = append(out.Tags, TagRef{ID: t.ID.String(), Name: t.Name})
		}
	}
	return out
}

// CreateWASDNSOverride creates a WAS DNS override record and returns it.
func (c *Client) CreateWASDNSOverride(ctx context.Context, in WASDNSOverrideInput) (*WASDNSOverride, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("qualys qps: WAS DNS override name is required")
	}
	if len(in.Mappings) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS DNS override requires at least one hostname/IP mapping")
	}

	var resp ServiceResponse
	err := c.call(ctx, http.MethodPost, "/qps/rest/3.0/create/was/dnsoverride",
		&ServiceRequest{Data: wasDNSOverrideData{DnsOverride: wasDNSOverrideInputToWire(in)}}, &resp, true)
	if err != nil {
		return nil, err
	}
	overrides, err := decodeWASDNSOverrides(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(overrides) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS DNS override was created but the API returned no data; " +
			"import it by ID to bring it under management")
	}
	return overrides[0], nil
}

// UpdateWASDNSOverride applies desired state to an existing WAS DNS
// override record. Unlike every other WAS object in this package, the
// update endpoint takes no ID in the URL path — the ID travels inside the
// request body instead, per the confirming source.
func (c *Client) UpdateWASDNSOverride(ctx context.Context, id string, in WASDNSOverrideInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS DNS override id is required for update")
	}
	wire := wasDNSOverrideInputToWire(in)
	wire.ID = json.Number(id)
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/update/was/dnsoverride",
		&ServiceRequest{Data: wasDNSOverrideData{DnsOverride: wire}}, nil, false)
}

// DeleteWASDNSOverride removes a WAS DNS override record. Like update, the
// delete endpoint takes no ID in the URL path — the ID travels as a filter
// criterion instead, per the confirming source.
func (c *Client) DeleteWASDNSOverride(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS DNS override id is required for delete")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/delete/was/dnsoverride",
		&ServiceRequest{Filters: &Filters{Criteria: []Criterion{{Field: "id", Operator: "EQUALS", Value: id}}}},
		nil, true)
}

// GetWASDNSOverride returns one WAS DNS override record, or ErrNotFound.
func (c *Client) GetWASDNSOverride(ctx context.Context, id string) (*WASDNSOverride, error) {
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, "/qps/rest/3.0/get/was/dnsoverride/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	overrides, err := decodeWASDNSOverrides(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(overrides) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS DNS override %s: %w", id, ErrNotFound)
	}
	return overrides[0], nil
}

// SearchWASDNSOverrides returns WAS DNS override records matching filters, following pagination.
func (c *Client) SearchWASDNSOverrides(ctx context.Context, filters *Filters) ([]*WASDNSOverride, error) {
	var all []*WASDNSOverride
	err := c.SearchAll(ctx, "/qps/rest/3.0/search/was/dnsoverride", filters, 100, 0,
		func(raw json.RawMessage) error {
			overrides, err := decodeWASDNSOverrides(raw)
			if err != nil {
				return err
			}
			all = append(all, overrides...)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return all, nil
}

func decodeWASDNSOverrides(raw json.RawMessage) ([]*WASDNSOverride, error) {
	items := decodeListOrSingle(raw)
	out := make([]*WASDNSOverride, 0, len(items))
	for _, item := range items {
		var d wasDNSOverrideData
		if err := json.Unmarshal(item, &d); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding WAS DNS override data: %w", err)
		}
		if d.DnsOverride != nil {
			out = append(out, d.DnsOverride.toWASDNSOverride())
		}
	}
	return out, nil
}
