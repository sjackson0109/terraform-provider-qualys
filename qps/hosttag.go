package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// HostAsset is an AssetView/CSAM host asset — the AM-module view of a host,
// distinct from vmdr.HostAsset (the VM/PA XML API's own host object). The
// two are related (they describe the same underlying host) but come from
// different API families with different identifiers and fields; this
// package does not attempt to reconcile them.
//
// Evidence tier: Corroborated. Doc 03 §14 confirms tag assignment
// (`POST /qps/rest/2.0/update/am/hostasset` with tags.add/tags.remove) and
// the search filter field names (qwebHostId, id, name, created, updated,
// tagName, tagId, lastVulnScan, lastComplianceScan,
// informationGatheredUpdated, os, dnsHostName, netbiosName,
// netbiosNetworkID, trackingMethod, port, installedSoftware) as Confirmed,
// but not a response field list for the object itself. The fields below
// reuse the search filter names verbatim as JSON field names, following
// the same convention this codebase already confirmed holds for every
// other qps object (Tag's ruleType/ruleText, WebApp's url/tags/...): the
// criteria field name and the response JSON field name match in every
// object this package has decoded so far. Unverified against a tenant.
type HostAsset struct {
	ID             string
	Name           string
	DNSHostName    string
	NetbiosName    string
	TrackingMethod string
	OS             string
	Created        string
	Updated        string
	Tags           []TagRef
}

type hostAssetWire struct {
	ID             json.Number    `json:"id,omitempty"`
	Name           string         `json:"name,omitempty"`
	DNSHostName    string         `json:"dnsHostName,omitempty"`
	NetbiosName    string         `json:"netbiosName,omitempty"`
	TrackingMethod string         `json:"trackingMethod,omitempty"`
	OS             string         `json:"os,omitempty"`
	Created        string         `json:"created,omitempty"`
	Updated        string         `json:"updated,omitempty"`
	Tags           *webAppTagList `json:"tags,omitempty"`
}

type hostAssetData struct {
	HostAsset *hostAssetWire `json:"HostAsset,omitempty"`
}

func (h *hostAssetWire) toHostAsset() *HostAsset {
	out := &HostAsset{
		ID:             h.ID.String(),
		Name:           h.Name,
		DNSHostName:    h.DNSHostName,
		NetbiosName:    h.NetbiosName,
		TrackingMethod: h.TrackingMethod,
		OS:             h.OS,
		Created:        h.Created,
		Updated:        h.Updated,
	}
	if h.Tags != nil {
		set := h.Tags.List
		if set == nil {
			set = h.Tags.Set
		}
		if set != nil {
			for _, t := range set.TagSimple {
				out.Tags = append(out.Tags, TagRef{ID: t.ID.String(), Name: t.Name})
			}
		}
	}
	return out
}

// UpdateHostAssetTags adds and/or removes tag associations on a host
// asset. Confirmed as an incremental add/remove idiom (not the "set"
// authoritative-replace idiom Tags and WebApps use elsewhere in this
// package) — doc 03 §14 shows only tags.add.TagSimple{id} /
// tags.remove.TagSimple{id}.
func (c *Client) UpdateHostAssetTags(ctx context.Context, hostAssetID string, add, remove []string) error {
	if strings.TrimSpace(hostAssetID) == "" {
		return fmt.Errorf("qualys qps: host asset id is required")
	}
	if len(add) == 0 && len(remove) == 0 {
		return nil
	}

	tagsPayload := map[string]interface{}{}
	if len(add) > 0 {
		tagsPayload["add"] = tagSimpleList{TagSimple: tagSimplesFor(add)}
	}
	if len(remove) > 0 {
		tagsPayload["remove"] = tagSimpleList{TagSimple: tagSimplesFor(remove)}
	}

	return c.call(ctx, http.MethodPost, "/qps/rest/2.0/update/am/hostasset/"+hostAssetID,
		&ServiceRequest{Data: map[string]interface{}{
			"HostAsset": map[string]interface{}{"tags": tagsPayload},
		}}, nil, false)
}

func tagSimplesFor(ids []string) []tagSimple {
	out := make([]tagSimple, 0, len(ids))
	for _, id := range ids {
		out = append(out, tagSimple{ID: json.Number(id)})
	}
	return out
}

// SearchHostAssets returns host assets matching filters, following
// pagination — most commonly used to look up hosts by tag (Criteria
// "tagName" or "tagId" EQUALS, per doc 03 §14).
func (c *Client) SearchHostAssets(ctx context.Context, filters *Filters) ([]*HostAsset, error) {
	var all []*HostAsset
	err := c.SearchAll(ctx, "/qps/rest/2.0/search/am/hostasset", filters, 100, 0,
		func(raw json.RawMessage) error {
			assets, err := decodeHostAssets(raw)
			if err != nil {
				return err
			}
			all = append(all, assets...)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return all, nil
}

func decodeHostAssets(raw json.RawMessage) ([]*HostAsset, error) {
	items := decodeListOrSingle(raw)
	out := make([]*HostAsset, 0, len(items))
	for _, item := range items {
		var d hostAssetData
		if err := json.Unmarshal(item, &d); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding host asset data: %w", err)
		}
		if d.HostAsset != nil {
			out = append(out, d.HostAsset.toHostAsset())
		}
	}
	return out, nil
}
