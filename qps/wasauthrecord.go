package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Sub-types confirmed by a primary-source excerpt (WAS API User Guide,
// Chapter 3 "Authentication API", p.102, "Current authentication record
// count") supplied directly during this provider's discovery work: the
// `credentials` count/search filter accepts exactly these five values,
// implying they are also the record's own sub-type vocabulary — FORM records
// come in STANDARD/CERT/SELFINITIAL flavours, SERVER records in BASIC/DIGEST.
// SELENIUM is a sixth FORM sub-type, Confirmed against the official
// docs.qualys.com "Create Authentication Record" reference (a user-supplied
// "Gap Review" document quoting it, not a direct fetch — this sandbox
// cannot reach docs.qualys.com — but citing the specific page rather than
// transcribing prose, so treated as strong evidence).
const (
	WASAuthFormStandard    = "STANDARD"
	WASAuthFormCustom      = "CUSTOM"
	WASAuthFormCert        = "CERT"
	WASAuthFormSelfInitial = "SELFINITIAL"
	WASAuthFormSelenium    = "SELENIUM"
	// WASAuthFormNone is not a real form record: setting formRecord.type to
	// NONE is the documented way to switch a record from form authentication
	// to OAuth2, per the same "Gap Review" source. See WASOAuth2Record.
	WASAuthFormNone     = "NONE"
	WASAuthServerBasic  = "BASIC"
	WASAuthServerDigest = "DIGEST"
)

// OAuth2 grant types, Confirmed against the same source as WASAuthFormSelenium.
const (
	WASOAuth2GrantNone        = "NONE"
	WASOAuth2GrantAuthCode    = "AUTH_CODE"
	WASOAuth2GrantImplicit    = "IMPLICIT"
	WASOAuth2GrantPassword    = "PASSWORD"
	WASOAuth2GrantClientCreds = "CLIENT_CREDS"
)

// WASAuthField is one named field of a WAS form authentication record's
// generic field collection.
type WASAuthField struct {
	Name    string
	Value   string
	Secured bool
}

// WASSeleniumScript is a Selenium IDE script attached to a SELENIUM form
// record, or to an OAuth2 AUTH_CODE/IMPLICIT grant that needs to drive a
// login page itself.
type WASSeleniumScript struct {
	Name string
	// Data is the exported Selenium IDE script content (HTML/XML).
	Data string
	// Regex is the pattern Qualys evaluates against the resulting page to
	// confirm the script authenticated successfully.
	Regex string
}

// WASFormRecord is the form-based half of a WAS authentication record.
//
// Username/Password are a convenience for the STANDARD case, but are NOT
// sent as flat elements: a user-supplied "Gap Review" document, quoting the
// official Qualys create example verbatim, showed STANDARD authentication
// using the same generic `fields`/`set`/`WebAppAuthFormRecordField` list as
// CUSTOM records — correcting this package's previous version, which sent
// STANDARD credentials as flat `username`/`password` elements based on an
// earlier, lower-confidence transcribed walkthrough. Username/Password here
// are encoded into that same fields list under the "username"/"password"
// names on write; see wasFormFieldsToWire.
//
// SeleniumScript/SeleniumCreds are for a SELENIUM record: the crawler runs
// the script instead of filling in a login form directly. SeleniumCreds
// enables the "@@authusername@@"/"@@authpassword@@" placeholder substitution
// the source documents, letting Username/Password be rotated without
// editing the script.
type WASFormRecord struct {
	// SubType is the record's authentication style. See the WASAuthForm*
	// constants above. Not validated client-side: an invalid value is left
	// for the API to reject at apply time.
	SubType   string
	LoginURL  string
	SSLOnly   bool
	AuthVault bool
	Username  string
	Password  string
	// Fields carries additional field/value pairs for a CUSTOM record whose
	// login form field names aren't the standard username/password.
	Fields         []WASAuthField
	SeleniumScript *WASSeleniumScript
	SeleniumCreds  bool
}

// WASServerRecord is the server-based (HTTP Basic/Digest-style) half of a
// WAS authentication record. Username/Password/Domain are flat elements
// under serverRecord — unlike WASFormRecord, the "Gap Review" correction
// only concerned form records; the server record shape is unchanged from
// the evidence that established it.
type WASServerRecord struct {
	SubType  string
	Username string
	Password string
	Domain   string
}

// WASOAuth2Record is a dedicated OAuth2 authentication record — Confirmed
// as its own `oauth2Record` object with flat, grant-specific fields, NOT
// modelled through the generic CUSTOM field list. Which fields the API
// requires depends on GrantType (see the WASOAuth2Grant* constants):
//
//	AUTH_CODE:    SeleniumScript, RedirectURL required; AccessTokenURL,
//	              ClientID, ClientSecret, Scope, AccessTokenExpiredMsgPattern optional
//	IMPLICIT:     SeleniumScript, RedirectURL required
//	PASSWORD:     AccessTokenURL, Username, Password required; ClientID,
//	              ClientSecret, Scope, AccessTokenExpiredMsgPattern optional
//	CLIENT_CREDS: AccessTokenURL required; ClientID, ClientSecret, Scope optional
//
// Not validated client-side beyond requiring GrantType — the API enforces
// the per-grant requirements above at apply time. A record can be switched
// from OAuth2 back to form authentication by setting GrantType to NONE (see
// WASAuthFormNone for the reverse direction).
type WASOAuth2Record struct {
	GrantType                    string
	AccessTokenURL               string
	ClientID                     string
	ClientSecret                 string
	Scope                        string
	RedirectURL                  string
	Username                     string
	Password                     string
	SeleniumScript               *WASSeleniumScript
	AccessTokenExpiredMsgPattern string
}

// WASAuthRecord is a WAS authentication record as read back from the API.
//
// Only the non-credential fields are modelled: Qualys masks form, server and
// OAuth2 field values on read (see the WASAuthRecordInput doc comment), so a
// faithful decode of the credential contents would be misleading even if
// attempted. Callers that need to know whether a record carries a form,
// server or OAuth2 record should keep that in their own configuration
// rather than infer it from a read.
type WASAuthRecord struct {
	ID       string
	Name     string
	Comments string
	Tags     []TagRef
	Created  string
}

// WASAuthRecordInput is the desired state of a WAS authentication record.
//
// Credentials here are write-only from Terraform's point of view: Qualys
// masks form, server and OAuth2 field values on read (confirmed — see doc
// 08 in the provider's discovery notes), so this package never attempts to
// decode them back out of a get/search response, and the provider resource
// never sets them from a read. A credential changed outside Terraform is
// not detected as drift.
//
// The wire shape is grounded in two rounds of user-supplied evidence: a
// "Create and Update Authentication Records" walkthrough (transcribed, not
// verbatim) established the endpoint path and record sub-type vocabulary,
// then a later "Gap Review" document — quoting the official Qualys create
// example directly — corrected STANDARD form records to use the same
// generic fields list as CUSTOM (this package's first version had them
// flat, based on the transcribed walkthrough alone) and added SELENIUM form
// records and the dedicated OAuth2Record. Exactly one of Form, Server or
// OAuth2 should be set per record.
type WASAuthRecordInput struct {
	Name     string
	Comments string
	TagIDs   []string
	Form     *WASFormRecord
	Server   *WASServerRecord
	OAuth2   *WASOAuth2Record
}

type wasAuthFieldWire struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

// wasAuthFieldSet is the "fields" list wrapper for a form record's generic
// field collection. The element key (WebAppAuthFormRecordField) and the
// fields/set/WebAppAuthFormRecordField nesting are both Confirmed — the
// latter by a user-supplied "Gap Review" document quoting the official
// Qualys STANDARD create example verbatim, which uses this exact shape.
type wasAuthFieldSet struct {
	WebAppAuthFormRecordField []wasAuthFieldWire `json:"WebAppAuthFormRecordField"`
}

type wasAuthFieldsWire struct {
	Set *wasAuthFieldSet `json:"set,omitempty"`
}

type wasSeleniumScriptWire struct {
	Name  string `json:"name,omitempty"`
	Data  string `json:"data,omitempty"`
	Regex string `json:"regex,omitempty"`
}

type wasFormRecordWire struct {
	Type           string                 `json:"type,omitempty"`
	LoginURL       string                 `json:"loginUrl,omitempty"`
	SSLOnly        *bool                  `json:"sslOnly,omitempty"`
	AuthVault      *bool                  `json:"authVault,omitempty"`
	Fields         *wasAuthFieldsWire     `json:"fields,omitempty"`
	SeleniumScript *wasSeleniumScriptWire `json:"seleniumScript,omitempty"`
	SeleniumCreds  *bool                  `json:"seleniumCreds,omitempty"`
}

type wasServerRecordWire struct {
	Type      string `json:"type,omitempty"`
	Domain    string `json:"domain,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	SSLOnly   *bool  `json:"sslOnly,omitempty"`
	AuthVault *bool  `json:"authVault,omitempty"`
}

type wasOAuth2RecordWire struct {
	GrantType                    string                 `json:"grantType,omitempty"`
	AccessTokenURL               string                 `json:"accessTokenUrl,omitempty"`
	ClientID                     string                 `json:"clientId,omitempty"`
	ClientSecret                 string                 `json:"clientSecret,omitempty"`
	Scope                        string                 `json:"scope,omitempty"`
	RedirectURL                  string                 `json:"redirectUrl,omitempty"`
	Username                     string                 `json:"username,omitempty"`
	Password                     string                 `json:"password,omitempty"`
	SeleniumScript               *wasSeleniumScriptWire `json:"seleniumScript,omitempty"`
	AccessTokenExpiredMsgPattern string                 `json:"accessTokenExpiredMsgPattern,omitempty"`
}

type wasAuthRecordWire struct {
	ID           json.Number          `json:"id,omitempty"`
	Name         string               `json:"name,omitempty"`
	FormRecord   *wasFormRecordWire   `json:"formRecord,omitempty"`
	ServerRecord *wasServerRecordWire `json:"serverRecord,omitempty"`
	OAuth2Record *wasOAuth2RecordWire `json:"oauth2Record,omitempty"`
	Tags         *webAppTagList       `json:"tags,omitempty"`
	Comments     string               `json:"comments,omitempty"`
	Created      string               `json:"createdDate,omitempty"`
}

type wasAuthRecordData struct {
	WebAppAuthRecord *wasAuthRecordWire `json:"WebAppAuthRecord,omitempty"`
}

// wasFormFieldsToWire builds a form record's generic fields list from the
// convenience Username/Password plus any explicit extra fields. Used for
// STANDARD and CUSTOM alike, per the STANDARD correction described on
// WASFormRecord.
func wasFormFieldsToWire(username, password string, extra []WASAuthField) *wasAuthFieldsWire {
	var fields []WASAuthField
	if username != "" {
		fields = append(fields, WASAuthField{Name: "username", Value: username})
	}
	if password != "" {
		fields = append(fields, WASAuthField{Name: "password", Value: password, Secured: true})
	}
	fields = append(fields, extra...)
	return wasAuthFieldsToWire(fields)
}

func wasAuthFieldsToWire(fields []WASAuthField) *wasAuthFieldsWire {
	if len(fields) == 0 {
		return nil
	}
	wire := make([]wasAuthFieldWire, 0, len(fields))
	for _, f := range fields {
		wire = append(wire, wasAuthFieldWire{Name: f.Name, Value: f.Value, Secured: f.Secured})
	}
	return &wasAuthFieldsWire{Set: &wasAuthFieldSet{WebAppAuthFormRecordField: wire}}
}

func wasSeleniumScriptToWire(s *WASSeleniumScript) *wasSeleniumScriptWire {
	if s == nil {
		return nil
	}
	return &wasSeleniumScriptWire{Name: s.Name, Data: s.Data, Regex: s.Regex}
}

func wasAuthRecordInputToWire(in WASAuthRecordInput) *wasAuthRecordWire {
	w := &wasAuthRecordWire{Name: in.Name, Comments: in.Comments}

	if len(in.TagIDs) > 0 {
		simples := make([]tagSimple, 0, len(in.TagIDs))
		for _, id := range in.TagIDs {
			simples = append(simples, tagSimple{ID: json.Number(id)})
		}
		w.Tags = &webAppTagList{Set: &tagSimpleList{TagSimple: simples}}
	}

	if in.Form != nil {
		sslOnly, authVault := in.Form.SSLOnly, in.Form.AuthVault
		formWire := &wasFormRecordWire{
			Type:      in.Form.SubType,
			LoginURL:  in.Form.LoginURL,
			SSLOnly:   &sslOnly,
			AuthVault: &authVault,
			Fields:    wasFormFieldsToWire(in.Form.Username, in.Form.Password, in.Form.Fields),
		}
		if in.Form.SeleniumScript != nil {
			formWire.SeleniumScript = wasSeleniumScriptToWire(in.Form.SeleniumScript)
			creds := in.Form.SeleniumCreds
			formWire.SeleniumCreds = &creds
		}
		w.FormRecord = formWire
	}

	if in.Server != nil {
		w.ServerRecord = &wasServerRecordWire{
			Type:     in.Server.SubType,
			Domain:   in.Server.Domain,
			Username: in.Server.Username,
			Password: in.Server.Password,
		}
	}

	if in.OAuth2 != nil {
		o := in.OAuth2
		oauthWire := &wasOAuth2RecordWire{
			GrantType:                    o.GrantType,
			AccessTokenURL:               o.AccessTokenURL,
			ClientID:                     o.ClientID,
			ClientSecret:                 o.ClientSecret,
			Scope:                        o.Scope,
			RedirectURL:                  o.RedirectURL,
			Username:                     o.Username,
			Password:                     o.Password,
			SeleniumScript:               wasSeleniumScriptToWire(o.SeleniumScript),
			AccessTokenExpiredMsgPattern: o.AccessTokenExpiredMsgPattern,
		}
		w.OAuth2Record = oauthWire
	}

	return w
}

func (w *wasAuthRecordWire) toWASAuthRecord() *WASAuthRecord {
	out := &WASAuthRecord{
		ID:       w.ID.String(),
		Name:     w.Name,
		Comments: w.Comments,
		Created:  w.Created,
	}
	if w.Tags != nil && w.Tags.List != nil {
		for _, t := range w.Tags.List.TagSimple {
			out.Tags = append(out.Tags, TagRef{ID: t.ID.String(), Name: t.Name})
		}
	}
	return out
}

// CreateWASAuthRecord creates a WAS authentication record and returns it.
//
// Exactly one of in.Form, in.Server or in.OAuth2 should be set; the API is
// not known to support more than one on a record at a time (only
// Corroborated evidence per record type individually), so sending more than
// one is not attempted here — the provider resource enforces the choice at
// plan time.
func (c *Client) CreateWASAuthRecord(ctx context.Context, in WASAuthRecordInput) (*WASAuthRecord, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("qualys qps: WAS authentication record name is required")
	}
	if in.Form == nil && in.Server == nil && in.OAuth2 == nil {
		return nil, fmt.Errorf("qualys qps: WAS authentication record requires a form, server or oauth2 record")
	}

	var resp ServiceResponse
	err := c.call(ctx, http.MethodPost, "/qps/rest/3.0/create/was/webauthrecord",
		&ServiceRequest{Data: wasAuthRecordData{WebAppAuthRecord: wasAuthRecordInputToWire(in)}}, &resp, true)
	if err != nil {
		return nil, err
	}
	recs, err := decodeWASAuthRecords(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS authentication record was created but the API returned no data; " +
			"import it by ID to bring it under management")
	}
	return recs[0], nil
}

// UpdateWASAuthRecord applies desired state to an existing WAS authentication record.
func (c *Client) UpdateWASAuthRecord(ctx context.Context, id string, in WASAuthRecordInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS authentication record id is required for update")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/update/was/webauthrecord/"+id,
		&ServiceRequest{Data: wasAuthRecordData{WebAppAuthRecord: wasAuthRecordInputToWire(in)}}, nil, false)
}

// DeleteWASAuthRecord removes a WAS authentication record.
func (c *Client) DeleteWASAuthRecord(ctx context.Context, id string) error {
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/delete/was/webauthrecord/"+id, nil, nil, true)
}

// GetWASAuthRecord returns one WAS authentication record, or ErrNotFound.
//
// Only id, name, comments, tags and the creation date are populated — see
// the WASAuthRecord doc comment for why credential contents are not
// decoded.
func (c *Client) GetWASAuthRecord(ctx context.Context, id string) (*WASAuthRecord, error) {
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, "/qps/rest/3.0/get/was/webauthrecord/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	recs, err := decodeWASAuthRecords(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS authentication record %s: %w", id, ErrNotFound)
	}
	return recs[0], nil
}

// SearchWASAuthRecords returns WAS authentication records matching filters, following pagination.
func (c *Client) SearchWASAuthRecords(ctx context.Context, filters *Filters) ([]*WASAuthRecord, error) {
	var all []*WASAuthRecord
	err := c.SearchAll(ctx, "/qps/rest/3.0/search/was/webauthrecord", filters, 100, 0,
		func(raw json.RawMessage) error {
			recs, err := decodeWASAuthRecords(raw)
			if err != nil {
				return err
			}
			all = append(all, recs...)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return all, nil
}

func decodeWASAuthRecords(raw json.RawMessage) ([]*WASAuthRecord, error) {
	items := decodeListOrSingle(raw)
	out := make([]*WASAuthRecord, 0, len(items))
	for _, item := range items {
		var d wasAuthRecordData
		if err := json.Unmarshal(item, &d); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding WAS authentication record data: %w", err)
		}
		if d.WebAppAuthRecord != nil {
			out = append(out, d.WebAppAuthRecord.toWASAuthRecord())
		}
	}
	return out, nil
}
