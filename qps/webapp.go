package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// WAS malware scheduling occurrence types are documented (ONCE, HOURLY,
// DAILY, WEEKLY, MONTHLY, per a user-supplied "Gap Review" document), but
// the exact recurrence request/response structure is not: the same document
// says it "should be modelled from the corresponding WebApp API
// examples/XSD rather than inferred" — so malwareScheduling itself is
// deliberately not modelled here, only the flat malwareMonitoring/
// malwareNotification booleans that gate it.

// WebAppAttribute is one custom key/value attribute on a web application.
// Confirmed by a user-supplied "Gap Review" document quoting the official
// Qualys reference verbatim: attributes/set/Attribute/{name,value}.
type WebAppAttribute struct {
	Name  string
	Value string
}

// WebAppFile is a base64-encoded file attached to a web application
// (swaggerFile or one of postmanCollection's three slots). Content is
// base64-encoded file content — pass the output of Terraform's
// `filebase64()` rather than raw text, consistent with how other providers
// model file uploads.
type WebAppFile struct {
	Name    string
	Content string
}

// WebAppPostmanCollection is a Postman collection import. Confirmed by the
// same "Gap Review" document: collection/environmentVariable/globalVariable
// each carry a base64-encoded {name, content} file, and are mutually
// exclusive with SwaggerFile on WebAppInput (Confirmed).
type WebAppPostmanCollection struct {
	Collection          *WebAppFile
	EnvironmentVariable *WebAppFile
	GlobalVariable      *WebAppFile
}

// WebAppCrawlingScript is a Selenium crawling script attached to a web
// application, as read back from a GET. Confirmed by the same "Gap Review"
// document (id/name/data/requiresAuthentication/startingUrl/
// startingUrlRegex) — this package does not support writing these back
// (no create/update example was supplied), only reading them.
type WebAppCrawlingScript struct {
	ID                     string
	Name                   string
	Data                   string
	RequiresAuthentication bool
	StartingURL            string
	StartingURLRegex       bool
}

// WebApp is a WAS web application: the scan target object that option
// profiles, auth records and scan schedules all reference.
//
// AuthRecordIDs is decoded on the same list/set pattern this package already
// confirmed for Tags (a "list" sub-key of {id, name} refs on read). A
// user-supplied "Gap Review" document later confirmed that `authRecords` is
// indeed the web application's authentication-record collection containing
// `WebAppAuthRecord` objects, but did not confirm whether it serialises as
// count/list or another wrapper shape in every platform version — treat the
// list decode here as corroborated, not fully confirmed, and verify against
// a tenant.
type WebApp struct {
	ID            string
	Name          string
	URL           string
	Tags          []TagRef
	AuthRecordIDs []string
	Attributes    []WebAppAttribute

	CancelScansAt          string
	CancelScansAfterNHours int
	DefaultDNSOverrideID   string
	DNSOverrideIDs         []string

	MalwareMonitoring   bool
	MalwareNotification bool

	CrawlingScripts []WebAppCrawlingScript

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

	// Attributes are custom key/value attributes, sent as a full
	// authoritative replacement (the "set" idiom) on every create/update.
	Attributes []WebAppAttribute

	// CancelScansAt (RFC3339) and CancelScansAfterNHours are mutually
	// exclusive; set at most one. Confirmed by a user-supplied "Gap Review"
	// document, which also documents a genuine landmine this package
	// reproduces faithfully rather than working around: if a config element
	// is sent at all (which happens whenever any of these three fields, or
	// DefaultDNSOverrideID, is set) and neither cancellation field is
	// populated, Qualys removes any existing default cancellation setting —
	// including one configured outside Terraform. Leaving all of
	// CancelScansAt/CancelScansAfterNHours/DefaultDNSOverrideID unset avoids
	// sending config at all, which the same source confirms leaves the
	// existing setting untouched.
	CancelScansAt          string
	CancelScansAfterNHours int
	// DefaultDNSOverrideID selects which of DNSOverrideIDs (or a DNS
	// override not otherwise managed by this provider) crawls/scans resolve
	// through by default. Confirmed as config.defaultDnsOverride.id.
	DefaultDNSOverrideID string
	// DNSOverrideIDs is the web application's DNS override collection,
	// distinct from DefaultDNSOverrideID (which one of these is the
	// default). Sent as a full authoritative replacement (the "set" idiom).
	// Confirmed as dnsOverrides/set/DnsOverride/{id}.
	DNSOverrideIDs []string

	// SwaggerFile and PostmanCollection are mutually exclusive (Confirmed).
	// Setting SwaggerFile to a zero-value *WebAppFile (non-nil, empty
	// fields) requests removal — the source shows an empty XML element for
	// this; this package's JSON translation of that (an empty JSON object)
	// is its own inference, not itself confirmed, so verify against a
	// tenant before relying on file removal specifically.
	SwaggerFile       *WebAppFile
	PostmanCollection *WebAppPostmanCollection

	MalwareMonitoring   bool
	MalwareNotification bool
}

type webAppAuthRecordRef struct {
	ID json.Number `json:"id,omitempty"`
}

type webAppAuthRecordRefList struct {
	WebAppAuthRecord []webAppAuthRecordRef `json:"WebAppAuthRecord"`
}

// webAppAuthRecordsWire is the "authRecords" element. A user-supplied
// example shows only add/remove (an incremental idiom, unlike tags' "set"
// authoritative replace); "list" is this package's own addition for
// decoding a GET response, corroborated (not fully confirmed) by a later
// "Gap Review" document — see the WebApp doc comment.
type webAppAuthRecordsWire struct {
	Add    *webAppAuthRecordRefList `json:"add,omitempty"`
	Remove *webAppAuthRecordRefList `json:"remove,omitempty"`
	List   *webAppAuthRecordRefList `json:"list,omitempty"`
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

type webAppAttributeWire struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type webAppAttributeSet struct {
	Attribute []webAppAttributeWire `json:"Attribute"`
}

type webAppAttributesWire struct {
	Set  *webAppAttributeSet `json:"set,omitempty"`
	List *webAppAttributeSet `json:"list,omitempty"`
}

type webAppDNSOverrideRef struct {
	ID json.Number `json:"id,omitempty"`
}

type webAppDNSOverrideRefList struct {
	DnsOverride []webAppDNSOverrideRef `json:"DnsOverride"`
}

type webAppDNSOverridesWire struct {
	Set  *webAppDNSOverrideRefList `json:"set,omitempty"`
	List *webAppDNSOverrideRefList `json:"list,omitempty"`
}

type webAppConfigWire struct {
	CancelScansAt          string        `json:"cancelScansAt,omitempty"`
	CancelScansAfterNHours int           `json:"cancelScansAfterNHours,omitempty"`
	DefaultDNSOverride     *wasIDRefWire `json:"defaultDnsOverride,omitempty"`
}

type webAppFileWire struct {
	Name    string `json:"name,omitempty"`
	Content string `json:"content,omitempty"`
}

type webAppPostmanCollectionWire struct {
	Collection          *webAppFileWire `json:"collection,omitempty"`
	EnvironmentVariable *webAppFileWire `json:"environmentVariable,omitempty"`
	GlobalVariable      *webAppFileWire `json:"globalVariable,omitempty"`
}

type webAppCrawlingScriptWire struct {
	ID                     json.Number `json:"id,omitempty"`
	Name                   string      `json:"name,omitempty"`
	Data                   string      `json:"data,omitempty"`
	RequiresAuthentication bool        `json:"requiresAuthentication,omitempty"`
	StartingURL            string      `json:"startingUrl,omitempty"`
	StartingURLRegex       bool        `json:"startingUrlRegex,omitempty"`
}

type webAppCrawlingScriptList struct {
	SeleniumScript []webAppCrawlingScriptWire `json:"SeleniumScript"`
}

type webAppCrawlingScriptsWire struct {
	Count int                       `json:"count,omitempty"`
	List  *webAppCrawlingScriptList `json:"list,omitempty"`
}

type webAppWire struct {
	ID          json.Number            `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Tags        *webAppTagList         `json:"tags,omitempty"`
	AuthRecords *webAppAuthRecordsWire `json:"authRecords,omitempty"`
	Attributes  *webAppAttributesWire  `json:"attributes,omitempty"`

	Config       *webAppConfigWire       `json:"config,omitempty"`
	DNSOverrides *webAppDNSOverridesWire `json:"dnsOverrides,omitempty"`

	SwaggerFile       *webAppFileWire              `json:"swaggerFile,omitempty"`
	PostmanCollection *webAppPostmanCollectionWire `json:"postmanCollection,omitempty"`

	MalwareMonitoring   *bool `json:"malwareMonitoring,omitempty"`
	MalwareNotification *bool `json:"malwareNotification,omitempty"`

	CrawlingScripts *webAppCrawlingScriptsWire `json:"crawlingScripts,omitempty"`

	Created string `json:"createdDate,omitempty"`
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
	if w.AuthRecords != nil && w.AuthRecords.List != nil {
		for _, r := range w.AuthRecords.List.WebAppAuthRecord {
			out.AuthRecordIDs = append(out.AuthRecordIDs, r.ID.String())
		}
	}
	if w.Attributes != nil {
		set := w.Attributes.List
		if set == nil {
			set = w.Attributes.Set
		}
		if set != nil {
			for _, a := range set.Attribute {
				out.Attributes = append(out.Attributes, WebAppAttribute{Name: a.Name, Value: a.Value})
			}
		}
	}
	if w.Config != nil {
		out.CancelScansAt = w.Config.CancelScansAt
		out.CancelScansAfterNHours = w.Config.CancelScansAfterNHours
		if w.Config.DefaultDNSOverride != nil {
			out.DefaultDNSOverrideID = w.Config.DefaultDNSOverride.ID.String()
		}
	}
	if w.DNSOverrides != nil {
		set := w.DNSOverrides.List
		if set == nil {
			set = w.DNSOverrides.Set
		}
		if set != nil {
			for _, d := range set.DnsOverride {
				out.DNSOverrideIDs = append(out.DNSOverrideIDs, d.ID.String())
			}
		}
	}
	if w.MalwareMonitoring != nil {
		out.MalwareMonitoring = *w.MalwareMonitoring
	}
	if w.MalwareNotification != nil {
		out.MalwareNotification = *w.MalwareNotification
	}
	if w.CrawlingScripts != nil && w.CrawlingScripts.List != nil {
		for _, s := range w.CrawlingScripts.List.SeleniumScript {
			out.CrawlingScripts = append(out.CrawlingScripts, WebAppCrawlingScript{
				ID:                     s.ID.String(),
				Name:                   s.Name,
				Data:                   s.Data,
				RequiresAuthentication: s.RequiresAuthentication,
				StartingURL:            s.StartingURL,
				StartingURLRegex:       s.StartingURLRegex,
			})
		}
	}
	return out
}

func webAppFileToWire(f *WebAppFile) *webAppFileWire {
	if f == nil {
		return nil
	}
	return &webAppFileWire{Name: f.Name, Content: f.Content}
}

func webAppInputToWire(in WebAppInput) *webAppWire {
	w := &webAppWire{Name: in.Name, URL: in.URL}
	simples := make([]tagSimple, 0, len(in.TagIDs))
	for _, id := range in.TagIDs {
		simples = append(simples, tagSimple{ID: json.Number(id)})
	}
	w.Tags = &webAppTagList{Set: &tagSimpleList{TagSimple: simples}}

	if len(in.Attributes) > 0 {
		attrs := make([]webAppAttributeWire, 0, len(in.Attributes))
		for _, a := range in.Attributes {
			attrs = append(attrs, webAppAttributeWire{Name: a.Name, Value: a.Value})
		}
		w.Attributes = &webAppAttributesWire{Set: &webAppAttributeSet{Attribute: attrs}}
	}

	if in.CancelScansAt != "" || in.CancelScansAfterNHours > 0 || strings.TrimSpace(in.DefaultDNSOverrideID) != "" {
		cfg := &webAppConfigWire{
			CancelScansAt:          in.CancelScansAt,
			CancelScansAfterNHours: in.CancelScansAfterNHours,
		}
		if strings.TrimSpace(in.DefaultDNSOverrideID) != "" {
			cfg.DefaultDNSOverride = &wasIDRefWire{ID: json.Number(in.DefaultDNSOverrideID)}
		}
		w.Config = cfg
	}

	if in.DNSOverrideIDs != nil {
		refs := make([]webAppDNSOverrideRef, 0, len(in.DNSOverrideIDs))
		for _, id := range in.DNSOverrideIDs {
			refs = append(refs, webAppDNSOverrideRef{ID: json.Number(id)})
		}
		w.DNSOverrides = &webAppDNSOverridesWire{Set: &webAppDNSOverrideRefList{DnsOverride: refs}}
	}

	if in.SwaggerFile != nil {
		w.SwaggerFile = webAppFileToWire(in.SwaggerFile)
	} else if in.PostmanCollection != nil {
		w.PostmanCollection = &webAppPostmanCollectionWire{
			Collection:          webAppFileToWire(in.PostmanCollection.Collection),
			EnvironmentVariable: webAppFileToWire(in.PostmanCollection.EnvironmentVariable),
			GlobalVariable:      webAppFileToWire(in.PostmanCollection.GlobalVariable),
		}
	}

	if in.MalwareMonitoring {
		w.MalwareMonitoring = boolPtr(true)
	}
	if in.MalwareNotification {
		w.MalwareNotification = boolPtr(true)
	}

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
// this package: an omitted field leaves the previous value in place. tags
// and attributes are always sent, including empty, because they use the
// "set" (authoritative replace) idiom, and an omitted set element could
// otherwise be read as leave-as-is on some qps endpoints — always sending
// them makes the intended state unambiguous. config is the one deliberate
// exception: it is only sent when at least one of
// CancelScansAt/CancelScansAfterNHours/DefaultDNSOverrideID is set, because
// Qualys itself treats a present-but-empty config as "clear the
// cancellation setting" — see the WebAppInput doc comment.
func (c *Client) UpdateWebApp(ctx context.Context, id string, in WebAppInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: web application id is required for update")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/update/was/webapp/"+id,
		&ServiceRequest{Data: webAppData{WebApp: webAppInputToWire(in)}}, nil, false)
}

// UpdateWebAppAuthRecordAssociations adds and/or removes authentication
// record associations on a web application.
//
// A user-supplied example shows this as its own update call — POST
// update/was/webapp/<id> with data.WebApp.authRecords.add/.remove, each a
// list of {id} refs — using an incremental add/remove idiom, distinct from
// the "set" (authoritative replace) idiom UpdateWebApp uses for tags. The
// same example states that creating an authentication record does not
// associate it with any web application; the association is always this
// separate step, including on first creation of a web application that
// should reference existing records.
func (c *Client) UpdateWebAppAuthRecordAssociations(ctx context.Context, webAppID string, add, remove []string) error {
	if strings.TrimSpace(webAppID) == "" {
		return fmt.Errorf("qualys qps: web application id is required")
	}
	if len(add) == 0 && len(remove) == 0 {
		return nil
	}

	authRecords := &webAppAuthRecordsWire{}
	if len(add) > 0 {
		authRecords.Add = &webAppAuthRecordRefList{WebAppAuthRecord: webAppAuthRecordRefsFor(add)}
	}
	if len(remove) > 0 {
		authRecords.Remove = &webAppAuthRecordRefList{WebAppAuthRecord: webAppAuthRecordRefsFor(remove)}
	}

	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/update/was/webapp/"+webAppID,
		&ServiceRequest{Data: webAppData{WebApp: &webAppWire{AuthRecords: authRecords}}}, nil, false)
}

func webAppAuthRecordRefsFor(ids []string) []webAppAuthRecordRef {
	out := make([]webAppAuthRecordRef, 0, len(ids))
	for _, id := range ids {
		out = append(out, webAppAuthRecordRef{ID: json.Number(id)})
	}
	return out
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
