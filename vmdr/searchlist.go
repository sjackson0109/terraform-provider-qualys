package vmdr

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Search lists name a set of vulnerability QIDs. Option profiles reference them
// by ID (custom_search_list_ids), so a profile with custom vulnerability
// detection cannot be expressed without them.
//
// Two kinds exist, on separate endpoints:
//
//	static  - an explicit list of QIDs
//	dynamic - criteria evaluated against the KnowledgeBase, so membership changes
//	          as Qualys publishes new detections
//
// They share a shape but not an endpoint, and their update semantics differ
// (static accepts additive qid parameters, dynamic does not), so they are
// modelled as separate types rather than one type with a kind switch.

// StaticSearchList is an explicit set of QIDs.
type StaticSearchList struct {
	ID       string
	Title    string
	Global   bool
	Comments string
	QIDs     []string
	Created  string
	Modified string
}

// StaticSearchListInput is the desired state of a static search list.
type StaticSearchListInput struct {
	Title    string
	Global   bool
	Comments string
	QIDs     []string
}

type searchListOutput struct {
	Response struct {
		Lists []struct {
			ID       string   `xml:"ID"`
			Title    string   `xml:"TITLE"`
			Global   string   `xml:"GLOBAL"`
			Comments string   `xml:"COMMENTS"`
			Created  string   `xml:"CREATED>DATETIME"`
			Modified string   `xml:"MODIFIED>DATETIME"`
			QIDList  []string `xml:"QIDS>QID"`
		} `xml:"STATIC_LISTS>STATIC_LIST"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *searchListOutput) warning() *warning { return o.Response.Warning }

func (o *searchListOutput) lists() []*StaticSearchList {
	out := make([]*StaticSearchList, 0, len(o.Response.Lists))
	for i := range o.Response.Lists {
		l := &o.Response.Lists[i]
		out = append(out, &StaticSearchList{
			ID:       strings.TrimSpace(l.ID),
			Title:    l.Title,
			Global:   isTrue(l.Global),
			Comments: l.Comments,
			QIDs:     trimAll(l.QIDList),
			Created:  l.Created,
			Modified: l.Modified,
		})
	}
	return out
}

// CreateStaticSearchList creates a list and returns its ID.
func (c *Client) CreateStaticSearchList(ctx context.Context, in StaticSearchListInput) (string, error) {
	if strings.TrimSpace(in.Title) == "" {
		return "", fmt.Errorf("qualys vmdr: search list title is required")
	}

	params := url.Values{}
	params.Set("title", in.Title)
	params.Set("global", boolParam(in.Global))
	setIfNotEmpty(params, "comments", in.Comments)
	setListIfNotEmpty(params, "qids", in.QIDs)

	var out simpleReturn
	if err := c.do(ctx, request{
		capability:    capSearchList,
		path:          "qid/search_list/static/",
		action:        "create",
		params:        params,
		nonIdempotent: true, // creating twice would duplicate the object
	}, &out); err != nil {
		return "", err
	}

	id, ok := out.itemValue("ID")
	if !ok || strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("qualys vmdr: search list was created but the API returned no ID "+
			"(response: %q); import it to bring it under management", out.Response.Text)
	}
	return strings.TrimSpace(id), nil
}

// UpdateStaticSearchList applies desired state.
//
// The QID list is sent whole via `qids` rather than through add_qids/remove_qids.
// Terraform is authoritative over the list, and the whole-list form is what makes
// a removed QID actually disappear.
func (c *Client) UpdateStaticSearchList(ctx context.Context, id string, in StaticSearchListInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys vmdr: search list id is required for update")
	}

	params := url.Values{}
	params.Set("id", id)
	params.Set("title", in.Title)
	params.Set("global", boolParam(in.Global))
	params.Set("comments", in.Comments)
	params.Set("qids", strings.Join(trimAll(in.QIDs), ","))

	return c.do(ctx, request{
		capability: capSearchList,
		path:       "qid/search_list/static/",
		action:     "update",
		params:     params,
	}, nil)
}

// DeleteStaticSearchList removes a list.
//
// Option profiles that reference it are not updated by Qualys, so a profile can
// be left pointing at a list that no longer exists. Order the removal so the
// referencing profile changes first.
func (c *Client) DeleteStaticSearchList(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("id", id)
	return c.do(ctx, request{
		capability:    capSearchList,
		path:          "qid/search_list/static/",
		action:        "delete",
		params:        params,
		nonIdempotent: true,
	}, nil)
}

// GetStaticSearchList returns one list, or ErrNotFound.
func (c *Client) GetStaticSearchList(ctx context.Context, id string) (*StaticSearchList, error) {
	lists, err := c.ListStaticSearchLists(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	for _, l := range lists {
		if l.ID == id {
			return l, nil
		}
	}
	return nil, fmt.Errorf("qualys vmdr: static search list %s: %w", id, ErrNotFound)
}

// ListStaticSearchLists returns lists, optionally filtered by ID.
func (c *Client) ListStaticSearchLists(ctx context.Context, ids []string) ([]*StaticSearchList, error) {
	params := url.Values{}
	setListIfNotEmpty(params, "ids", ids)

	var all []*StaticSearchList
	err := c.listAll(ctx, request{
		capability: capSearchList,
		path:       "qid/search_list/static/",
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(searchListOutput) },
		func(p paginated) error {
			all = append(all, p.(*searchListOutput).lists()...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}

func boolParam(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// isTrue reads the several shapes Qualys uses for booleans across endpoints.
func isTrue(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes":
		return true
	}
	return false
}
