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
// A second user-supplied excerpt (a "Create and Update Authentication
// Records" walkthrough, transcribed rather than a verbatim PDF quote) shows
// STANDARD and a separate CUSTOM form type; CUSTOM is not in the p.102
// filter list, so it may be a documentation-only distinction rather than a
// literal `type` value, or the filter list may simply be incomplete.
const (
	WASAuthFormStandard    = "STANDARD"
	WASAuthFormCustom      = "CUSTOM"
	WASAuthFormCert        = "CERT"
	WASAuthFormSelfInitial = "SELFINITIAL"
	WASAuthServerBasic     = "BASIC"
	WASAuthServerDigest    = "DIGEST"
)

// WASAuthField is one named field of a CUSTOM WAS form authentication
// record: an arbitrary login-form field/value pair. Only CUSTOM records are
// believed to need this — STANDARD records use the fixed Username/Password
// fields on WASFormRecord directly (see the WASFormRecord doc comment).
type WASAuthField struct {
	Name    string
	Value   string
	Secured bool
}

// WASFormRecord is the form-based half of a WAS authentication record.
//
// Username/Password/LoginURL are fixed fields for a STANDARD record (the
// common case: a login page with a known username and password field). A
// user-supplied "Create Authentication Record" example shows these as flat
// elements directly under formRecord, not wrapped in a generic field list —
// a correction from this package's first version, which sent every
// credential (including STANDARD username/password) through the generic
// Fields list below.
//
// Fields carries arbitrary field/value pairs for a CUSTOM record, where the
// login form's field names aren't known in advance. This is Corroborated
// (the WAS API guide's derived reference and an open-source Qualys API
// client both describe a generic fields list) but not seen in a CUSTOM
// example specifically — only in general documentation prose.
type WASFormRecord struct {
	// SubType is the record's authentication style. See the WASAuthForm*
	// constants above. Not validated client-side: STANDARD and CUSTOM are
	// corroborated by a create-example and by supported-types documentation
	// respectively, but CERT/SELFINITIAL are only corroborated via a filter
	// enum, so an invalid value is left for the API to reject at apply time.
	SubType   string
	LoginURL  string
	Username  string
	Password  string
	SSLOnly   bool
	AuthVault bool
	Fields    []WASAuthField
}

// WASServerRecord is the server-based (HTTP Basic/Digest-style) half of a
// WAS authentication record. Username/Password/Domain are flat elements
// under serverRecord — corrected from this package's first version, which
// sent them through a generic fields list borrowed from an open-source
// client's abstraction that a user-supplied example does not show.
type WASServerRecord struct {
	SubType  string
	Username string
	Password string
	Domain   string
}

// WASAuthRecord is a WAS authentication record as read back from the API.
//
// Only the non-credential fields are modelled: Qualys masks form and server
// field values on read (see the WASAuthRecordInput doc comment), so a
// faithful decode of the credential contents would be misleading even if
// attempted. Callers that need to know whether a record carries a form or
// server record should keep that in their own configuration rather than
// infer it from a read.
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
// masks form and server field values on read (confirmed — see doc 08 in the
// provider's discovery notes), so this package never attempts to decode them
// back out of a get/search response, and the provider resource never sets
// them from a read. A credential changed outside Terraform is not detected
// as drift.
//
// The wire shape is now grounded in a user-supplied "Create and Update
// Authentication Records" walkthrough — transcribed rather than a verbatim
// PDF quote, so treated as strong but not absolute evidence — corroborated
// by a genuine primary-source excerpt (WAS API User Guide, Chapter 3, p.102)
// that separately confirmed the endpoint path (/was/webauthrecord, not this
// package's original /was/webappauthrecord guess) and the record sub-type
// vocabulary. Verify against a tenant before relying on this for a Selenium
// or OAuth2 record; only form and server records are implemented.
type WASAuthRecordInput struct {
	Name     string
	Comments string
	TagIDs   []string
	Form     *WASFormRecord
	Server   *WASServerRecord
}

type wasAuthFieldWire struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

// wasAuthFieldSet is the "fields" list wrapper for a CUSTOM form record. The
// element key (WebAppAuthFormRecordField) is the one class name doc 11's
// derived WAS API reference names explicitly for this list.
type wasAuthFieldSet struct {
	WebAppAuthFormRecordField []wasAuthFieldWire `json:"WebAppAuthFormRecordField"`
}

type wasAuthFieldsWire struct {
	Set *wasAuthFieldSet `json:"set,omitempty"`
}

type wasFormRecordWire struct {
	Type      string             `json:"type,omitempty"`
	LoginURL  string             `json:"loginUrl,omitempty"`
	Username  string             `json:"username,omitempty"`
	Password  string             `json:"password,omitempty"`
	SSLOnly   *bool              `json:"sslOnly,omitempty"`
	AuthVault *bool              `json:"authVault,omitempty"`
	Fields    *wasAuthFieldsWire `json:"fields,omitempty"`
}

type wasServerRecordWire struct {
	Type      string `json:"type,omitempty"`
	Domain    string `json:"domain,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	SSLOnly   *bool  `json:"sslOnly,omitempty"`
	AuthVault *bool  `json:"authVault,omitempty"`
}

type wasAuthRecordWire struct {
	ID           json.Number          `json:"id,omitempty"`
	Name         string               `json:"name,omitempty"`
	FormRecord   *wasFormRecordWire   `json:"formRecord,omitempty"`
	ServerRecord *wasServerRecordWire `json:"serverRecord,omitempty"`
	Tags         *webAppTagList       `json:"tags,omitempty"`
	Comments     string               `json:"comments,omitempty"`
	Created      string               `json:"createdDate,omitempty"`
}

type wasAuthRecordData struct {
	WebAppAuthRecord *wasAuthRecordWire `json:"WebAppAuthRecord,omitempty"`
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
		w.FormRecord = &wasFormRecordWire{
			Type:      in.Form.SubType,
			LoginURL:  in.Form.LoginURL,
			Username:  in.Form.Username,
			Password:  in.Form.Password,
			SSLOnly:   &sslOnly,
			AuthVault: &authVault,
			Fields:    wasAuthFieldsToWire(in.Form.Fields),
		}
	}

	if in.Server != nil {
		w.ServerRecord = &wasServerRecordWire{
			Type:     in.Server.SubType,
			Domain:   in.Server.Domain,
			Username: in.Server.Username,
			Password: in.Server.Password,
		}
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
// Exactly one of in.Form or in.Server should be set; the API is not known to
// support both record types on one record (only Corroborated evidence for
// either individually), so sending both is not attempted here — the
// provider resource enforces the choice at plan time.
func (c *Client) CreateWASAuthRecord(ctx context.Context, in WASAuthRecordInput) (*WASAuthRecord, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("qualys qps: WAS authentication record name is required")
	}
	if in.Form == nil && in.Server == nil {
		return nil, fmt.Errorf("qualys qps: WAS authentication record requires a form or server record")
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
