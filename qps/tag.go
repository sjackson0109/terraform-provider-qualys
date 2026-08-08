package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Tag is an asset tag.
//
// Tags live in a single subscription-wide tree managed by AssetView/CSAM. The
// same tag is referenced by VM scan and report targeting and by WAS web
// applications, so one tag resource serves all three modules rather than each
// module needing its own.
type Tag struct {
	ID       string
	Name     string
	Parent   string
	RuleType string
	RuleText string
	Color    string
	Created  string
	Modified string
	Children []TagRef
}

// TagRef identifies a tag by ID and name.
type TagRef struct {
	ID   string
	Name string
}

// Rule types accepted by the tagging API. A tag with no rule type is static:
// membership is assigned explicitly rather than evaluated.
const (
	RuleTypeStatic            = "STATIC"
	RuleTypeGroovy            = "GROOVY"
	RuleTypeOSRegex           = "OS_REGEX"
	RuleTypeNetworkRange      = "NETWORK_RANGE"
	RuleTypeNameContains      = "NAME_CONTAINS"
	RuleTypeInstalledSoftware = "INSTALLED_SOFTWARE"
	RuleTypeOpenPorts         = "OPEN_PORTS"
	RuleTypeVulnExist         = "VULN_EXIST"
	RuleTypeAssetSearch       = "ASSET_SEARCH"
	RuleTypeGlobalAssetView   = "GLOBAL_ASSET_VIEW"
)

// TagInput is the desired state of a tag.
type TagInput struct {
	Name     string
	Parent   string
	RuleType string
	RuleText string
	Color    string
}

// wire types. The portal API nests everything under data.Tag and represents
// collections as {"set"/"add"/"remove": {"TagSimple": [...]}}.
type tagWire struct {
	ID       json.Number `json:"id,omitempty"`
	Name     string      `json:"name,omitempty"`
	Parent   json.Number `json:"parentTagId,omitempty"`
	RuleType string      `json:"ruleType,omitempty"`
	RuleText string      `json:"ruleText,omitempty"`
	Color    string      `json:"color,omitempty"`
	Created  string      `json:"created,omitempty"`
	Modified string      `json:"modified,omitempty"`
	Children *struct {
		List []tagSimple `json:"list"`
	} `json:"children,omitempty"`
}

type tagSimple struct {
	ID   json.Number `json:"id,omitempty"`
	Name string      `json:"name,omitempty"`
}

type tagData struct {
	Tag *tagWire `json:"Tag,omitempty"`
}

func (t *tagWire) toTag() *Tag {
	out := &Tag{
		ID:       t.ID.String(),
		Name:     t.Name,
		Parent:   t.Parent.String(),
		RuleType: t.RuleType,
		RuleText: t.RuleText,
		Color:    t.Color,
		Created:  t.Created,
		Modified: t.Modified,
	}
	if out.Parent == "0" {
		// The API reports a root tag's parent as 0; surface that as "no parent"
		// so it round-trips against an unset configuration attribute.
		out.Parent = ""
	}
	if t.Children != nil {
		for _, c := range t.Children.List {
			out.Children = append(out.Children, TagRef{ID: c.ID.String(), Name: c.Name})
		}
	}
	return out
}

func tagInputToWire(in TagInput) *tagWire {
	w := &tagWire{
		Name:     in.Name,
		RuleType: in.RuleType,
		RuleText: in.RuleText,
		Color:    in.Color,
	}
	if strings.TrimSpace(in.Parent) != "" {
		w.Parent = json.Number(in.Parent)
	}
	return w
}

// CreateTag creates a tag and returns it.
func (c *Client) CreateTag(ctx context.Context, in TagInput) (*Tag, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("qualys qps: tag name is required")
	}

	var resp ServiceResponse
	err := c.call(ctx, http.MethodPost, "/qps/rest/2.0/create/am/tag",
		&ServiceRequest{Data: tagData{Tag: tagInputToWire(in)}}, &resp, true)
	if err != nil {
		return nil, err
	}

	tags, err := decodeTags(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("qualys qps: tag was created but the API returned no tag data; " +
			"import it by ID to bring it under management")
	}
	return tags[0], nil
}

// UpdateTag applies desired state to an existing tag.
func (c *Client) UpdateTag(ctx context.Context, id string, in TagInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: tag id is required for update")
	}
	// Update sends the managed fields unconditionally, including empty ones.
	// The create encoder omits empty values (omitempty), which is right for
	// create but wrong here: an omitted field leaves the old value in place, so
	// clearing an attribute in configuration would never take effect and the
	// resource would diff forever.
	//
	// Two exceptions:
	//
	// ruleType — an empty rule type is not meaningful (a static tag simply has
	// none) and the API rejects it, so it is omitted when unset.
	//
	// parentTagId — omitting it keeps the old parent, and no payload for
	// "detach from parent" has been verified against the API, so an update
	// cannot express re-rooting at all. Rather than accept input it would
	// silently ignore, UpdateTag does not carry the parent; the resource marks
	// parent_tag_id ForceNew and moving a tag recreates it.
	tag := map[string]interface{}{
		"name":     in.Name,
		"ruleText": in.RuleText,
		"color":    in.Color,
	}
	if strings.TrimSpace(in.RuleType) != "" {
		tag["ruleType"] = in.RuleType
	}

	return c.call(ctx, http.MethodPost, "/qps/rest/2.0/update/am/tag/"+id,
		&ServiceRequest{Data: map[string]interface{}{"Tag": tag}}, nil, false)
}

// DeleteTag removes a tag. Assets keep existing; only the association is lost.
func (c *Client) DeleteTag(ctx context.Context, id string) error {
	return c.call(ctx, http.MethodPost, "/qps/rest/2.0/delete/am/tag/"+id, nil, nil, true)
}

// GetTag returns one tag, or ErrNotFound.
func (c *Client) GetTag(ctx context.Context, id string) (*Tag, error) {
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, "/qps/rest/2.0/get/am/tag/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	tags, err := decodeTags(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("qualys qps: tag %s: %w", id, ErrNotFound)
	}
	return tags[0], nil
}

// SearchTags returns tags matching filters, following pagination.
func (c *Client) SearchTags(ctx context.Context, filters *Filters) ([]*Tag, error) {
	var all []*Tag
	err := c.SearchAll(ctx, "/qps/rest/2.0/search/am/tag", filters, 100, 0,
		func(raw json.RawMessage) error {
			tags, err := decodeTags(raw)
			if err != nil {
				return err
			}
			all = append(all, tags...)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return all, nil
}

// decodeTags handles the two shapes the portal API uses for `data`: a single
// object for get/create, and a list for search.
func decodeTags(raw json.RawMessage) ([]*Tag, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	// Try the list shape first: [{"Tag": {...}}, ...]
	var list []tagData
	if err := json.Unmarshal(raw, &list); err == nil {
		out := make([]*Tag, 0, len(list))
		for _, d := range list {
			if d.Tag != nil {
				out = append(out, d.Tag.toTag())
			}
		}
		return out, nil
	}

	// Fall back to the single-object shape: {"Tag": {...}}
	var one tagData
	if err := json.Unmarshal(raw, &one); err != nil {
		return nil, fmt.Errorf("qualys qps: decoding tag data: %w", err)
	}
	if one.Tag == nil {
		return nil, nil
	}
	return []*Tag{one.Tag.toTag()}, nil
}
