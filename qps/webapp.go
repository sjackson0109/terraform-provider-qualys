package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// WebApp is a WAS web application: the scan target object that option
// profiles, auth records and scan schedules all reference.
type WebApp struct {
	ID      string
	Name    string
	URL     string
	Tags    []TagRef
	Created string
}

// WebAppInput is the desired state of a web application.
type WebAppInput struct {
	Name string
	URL  string
	// TagIDs are asset tag IDs (see package qps's Tag type) to associate with
	// this web application. Sent as a full authoritative replacement on every
	// create/update, matching the "set" idiom the tagging API documents for
	// tag-list associations, rather than an incremental add/remove.
	TagIDs []string
}

// tagSimpleList is the wire shape the portal API uses under both "set" and
// "list" for a tag association: {"TagSimple": [...]}. Named once and reused
// rather than declared inline at each use site, so a future field addition
// (a count, a cursor) only needs to change in one place.
type tagSimpleList struct {
	TagSimple []tagSimple `json:"TagSimple"`
}

type webAppTagList struct {
	Set  *tagSimpleList `json:"set,omitempty"`
	List *tagSimpleList `json:"list,omitempty"`
}

type webAppWire struct {
	ID      json.Number    `json:"id,omitempty"`
	Name    string         `json:"name,omitempty"`
	URL     string         `json:"url,omitempty"`
	Tags    *webAppTagList `json:"tags,omitempty"`
	Created string         `json:"createdDate,omitempty"`
}

type webAppData struct {
	WebApp *webAppWire `json:"WebApp,omitempty"`
}

func (w *webAppWire) toWebApp() *WebApp {
	out := &WebApp{
		ID:      w.ID.String(),
		Name:    w.Name,
		URL:     w.URL,
		Created: w.Created,
	}
	if w.Tags != nil && w.Tags.List != nil {
		for _, t := range w.Tags.List.TagSimple {
			out.Tags = append(out.Tags, TagRef{ID: t.ID.String(), Name: t.Name})
		}
	}
	return out
}

func webAppInputToWire(in WebAppInput) *webAppWire {
	w := &webAppWire{Name: in.Name, URL: in.URL}
	simples := make([]tagSimple, 0, len(in.TagIDs))
	for _, id := range in.TagIDs {
		simples = append(simples, tagSimple{ID: json.Number(id)})
	}
	w.Tags = &webAppTagList{Set: &tagSimpleList{TagSimple: simples}}
	return w
}

// CreateWebApp creates a web application and returns it.
func (c *Client) CreateWebApp(ctx context.Context, in WebAppInput) (*WebApp, error) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.URL) == "" {
		return nil, fmt.Errorf("qualys qps: web application name and url are required")
	}

	var resp ServiceResponse
	err := c.call(ctx, http.MethodPost, "/qps/rest/3.0/create/was/webapp",
		&ServiceRequest{Data: webAppData{WebApp: webAppInputToWire(in)}}, &resp, true)
	if err != nil {
		return nil, err
	}
	apps, err := decodeWebApps(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(apps) == 0 {
		return nil, fmt.Errorf("qualys qps: web application was created but the API returned no data; " +
			"import it by ID to bring it under management")
	}
	return apps[0], nil
}

// UpdateWebApp applies desired state to an existing web application.
//
// name and url are omitted when unset, like every other update encoder in
// this package: an omitted field leaves the previous value in place. tags is
// always sent, including empty, because it uses the "set" (authoritative
// replace) idiom, and an omitted set element could otherwise be read as
// leave-as-is on some qps endpoints — always sending it makes the intended
// state unambiguous.
func (c *Client) UpdateWebApp(ctx context.Context, id string, in WebAppInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: web application id is required for update")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/update/was/webapp/"+id,
		&ServiceRequest{Data: webAppData{WebApp: webAppInputToWire(in)}}, nil, false)
}

// DeleteWebApp removes a web application.
func (c *Client) DeleteWebApp(ctx context.Context, id string) error {
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/delete/was/webapp/"+id, nil, nil, true)
}

// GetWebApp returns one web application, or ErrNotFound.
func (c *Client) GetWebApp(ctx context.Context, id string) (*WebApp, error) {
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, "/qps/rest/3.0/get/was/webapp/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	apps, err := decodeWebApps(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(apps) == 0 {
		return nil, fmt.Errorf("qualys qps: web application %s: %w", id, ErrNotFound)
	}
	return apps[0], nil
}

// SearchWebApps returns web applications matching filters, following pagination.
func (c *Client) SearchWebApps(ctx context.Context, filters *Filters) ([]*WebApp, error) {
	var all []*WebApp
	err := c.SearchAll(ctx, "/qps/rest/3.0/search/was/webapp", filters, 100, 0,
		func(raw json.RawMessage) error {
			apps, err := decodeWebApps(raw)
			if err != nil {
				return err
			}
			all = append(all, apps...)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return all, nil
}

func decodeWebApps(raw json.RawMessage) ([]*WebApp, error) {
	items := decodeListOrSingle(raw)
	out := make([]*WebApp, 0, len(items))
	for _, item := range items {
		var d webAppData
		if err := json.Unmarshal(item, &d); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding web application data: %w", err)
		}
		if d.WebApp != nil {
			out = append(out, d.WebApp.toWebApp())
		}
	}
	return out, nil
}
