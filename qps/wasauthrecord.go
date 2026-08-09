package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// WASAuthField is one named credential field of a WAS authentication record:
// a login-form field/value pair, or (for a server record) one of the
// synthetic "username"/"password"/"domain" fields the API represents server
// credentials as.
type WASAuthField struct {
	Name    string
	Value   string
	Secured bool
}

// WASFormRecord is the form-based half of a WAS authentication record: the
// crawler fills these field/value pairs into a login page.
type WASFormRecord struct {
	// SubType is the record's authentication style (Qualys examples include
	// STANDARD and SELENIUM). Not validated client-side: the confirmed
	// values are not corroborated well enough to enumerate; an invalid
	// value is rejected by the API at apply time.
	SubType   string
	SSLOnly   bool
	AuthVault bool
	Fields    []WASAuthField
}

// WASServerRecord is the server-based (HTTP Basic/NTLM/Digest-style) half of
// a WAS authentication record.
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
	ID      string
	Name    string
	Tags    []TagRef
	Created string
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
// The wire shape for formRecord/serverRecord (a "type" sub-type string, an
// sslOnly/authVault flag pair, and a "fields" list of {name, value, secured}
// entries under the "set" idiom already used for tag associations) is
// corroborated from two independent, non-official sources — the WAS API
// guide's derived reference and an open-source Qualys API client — rather
// than read directly from the official WAS API User Guide PDF, which was
// unreachable (network egress to docs.qualys.com/cdn2.qualys.com is blocked
// in the environment this was built in). Verify against a tenant before
// relying on it for a Selenium or OAuth2 record; only form and server
// records are implemented.
type WASAuthRecordInput struct {
	Name   string
	TagIDs []string
	Form   *WASFormRecord
	Server *WASServerRecord
}

type wasAuthFieldWire struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

// wasAuthFieldSet is the "fields" list wrapper. The element key
// (WebAppAuthFormRecordField) is the one class name doc 11's derived WAS API
// reference names explicitly for this list, reused here for both form and
// server records since the API represents server credentials as fields too.
type wasAuthFieldSet struct {
	WebAppAuthFormRecordField []wasAuthFieldWire `json:"WebAppAuthFormRecordField"`
}

type wasAuthFieldsWire struct {
	Set *wasAuthFieldSet `json:"set,omitempty"`
}

type wasAuthSubRecordWire struct {
	Type      string             `json:"type,omitempty"`
	SSLOnly   *bool              `json:"sslOnly,omitempty"`
	AuthVault *bool              `json:"authVault,omitempty"`
	Fields    *wasAuthFieldsWire `json:"fields,omitempty"`
}

type wasAuthRecordWire struct {
	ID           json.Number           `json:"id,omitempty"`
	Name         string                `json:"name,omitempty"`
	FormRecord   *wasAuthSubRecordWire `json:"formRecord,omitempty"`
	ServerRecord *wasAuthSubRecordWire `json:"serverRecord,omitempty"`
	Tags         *webAppTagList        `json:"tags,omitempty"`
	Created      string                `json:"createdDate,omitempty"`
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
	w := &wasAuthRecordWire{Name: in.Name}

	if len(in.TagIDs) > 0 {
		simples := make([]tagSimple, 0, len(in.TagIDs))
		for _, id := range in.TagIDs {
			simples = append(simples, tagSimple{ID: json.Number(id)})
		}
		w.Tags = &webAppTagList{Set: &tagSimpleList{TagSimple: simples}}
	}

	if in.Form != nil {
		sslOnly, authVault := in.Form.SSLOnly, in.Form.AuthVault
		w.FormRecord = &wasAuthSubRecordWire{
			Type:      in.Form.SubType,
			SSLOnly:   &sslOnly,
			AuthVault: &authVault,
			Fields:    wasAuthFieldsToWire(in.Form.Fields),
		}
	}

	if in.Server != nil {
		fields := []WASAuthField{{Name: "username", Value: in.Server.Username}}
		if strings.TrimSpace(in.Server.Domain) != "" {
			fields = append(fields, WASAuthField{Name: "domain", Value: in.Server.Domain})
		}
		fields = append(fields, WASAuthField{Name: "password", Value: in.Server.Password, Secured: true})
		w.ServerRecord = &wasAuthSubRecordWire{
			Type:   in.Server.SubType,
			Fields: wasAuthFieldsToWire(fields),
		}
	}

	return w
}

func (w *wasAuthRecordWire) toWASAuthRecord() *WASAuthRecord {
	out := &WASAuthRecord{
		ID:      w.ID.String(),
		Name:    w.Name,
		Created: w.Created,
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
	err := c.call(ctx, http.MethodPost, "/qps/rest/3.0/create/was/webappauthrecord",
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
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/update/was/webappauthrecord/"+id,
		&ServiceRequest{Data: wasAuthRecordData{WebAppAuthRecord: wasAuthRecordInputToWire(in)}}, nil, false)
}

// DeleteWASAuthRecord removes a WAS authentication record.
func (c *Client) DeleteWASAuthRecord(ctx context.Context, id string) error {
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/delete/was/webappauthrecord/"+id, nil, nil, true)
}

// GetWASAuthRecord returns one WAS authentication record, or ErrNotFound.
//
// Only id, name, tags and the creation date are populated — see the
// WASAuthRecord doc comment for why credential contents are not decoded.
func (c *Client) GetWASAuthRecord(ctx context.Context, id string) (*WASAuthRecord, error) {
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, "/qps/rest/3.0/get/was/webappauthrecord/"+id, nil, &resp, false)
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
	err := c.SearchAll(ctx, "/qps/rest/3.0/search/was/webappauthrecord", filters, 100, 0,
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
