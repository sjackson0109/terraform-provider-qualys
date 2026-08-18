package qps

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Connector v3 (doc 12). The AWS/Azure/GCP asset data connectors live under
// /qps/rest/3.0/<verb>/am/<cloud>assetdataconnector and use the same
// ServiceRequest/ServiceResponse envelope as every other object in this
// package. They replace the deprecated CloudView v1 connector API
// (/cloudview-api/rest/v1/...); v3 objects expose the v1 connector's UUID as
// cloudviewUuid, which is what lets pre-migration Terraform state find its
// connector under the new numeric id.
//
// Wire quirks confirmed by the official docs (doc 12):
//
//   - Request collections nest under "set"; response collections nest under
//     "list". The response-side JSON nesting is only shown for XML in the
//     docs, so response collection decoding here is shape-tolerant
//     (Corroborated, not Confirmed) rather than asserting one layout.
//   - Response booleans arrive as strings ("disabled": "false") while
//     requests take real booleans; flexBool absorbs both.
//   - Update responds with the connector id only, so callers must re-Get to
//     observe the resulting state.

// Connector activation modules documented for the v3 connector APIs.
const (
	ActivationVM        = "VM"
	ActivationCertView  = "CERTVIEW"
	ActivationCloudView = "CLOUDVIEW"
	ActivationSCA       = "SCA"
	ActivationCSA       = "CSA"
)

// ConnectorBaseInput carries the desired state common to all three clouds.
type ConnectorBaseInput struct {
	Name        string
	Description string

	// RunFrequency is the polling interval in minutes. Zero omits the field
	// (the GCP API documents it as required; the resource layer supplies a
	// default there).
	RunFrequency int

	// Tri-state booleans: nil omits the field, leaving the server default
	// (create) or current value (update) in place.
	Disabled             *bool
	IsRemediationEnabled *bool

	// ActivationModules lists the Qualys modules the connector feeds
	// (ActivationVM, ActivationCertView, ...). Empty omits the field.
	ActivationModules []string

	// DefaultTagIDs are tag ids applied to discovered assets. Empty omits
	// the field.
	DefaultTagIDs []string
}

// ConnectorBase is the read-side state common to all three clouds.
type ConnectorBase struct {
	ID          string
	Name        string
	Description string

	// State is the connector's lifecycle state (QUEUED, RUNNING,
	// FINISHED_SUCCESS, ...).
	State string

	Disabled             bool
	RunFrequency         int
	IsRemediationEnabled bool

	LastSync  string
	NextSync  string
	LastError string

	// CloudViewUUID is the deprecated CloudView v1 API's UUID for the same
	// connector — the identity bridge for state written by pre-v3 versions
	// of this provider.
	CloudViewUUID string

	ActivationModules []string
	DefaultTags       []TagRef
}

// connectorWireBase is the request-side shared shape.
type connectorWireBase struct {
	Name                 string             `json:"name,omitempty"`
	Description          string             `json:"description,omitempty"`
	RunFrequency         int                `json:"runFrequency,omitempty"`
	Disabled             *bool              `json:"disabled,omitempty"`
	IsRemediationEnabled *bool              `json:"isRemediationEnabled,omitempty"`
	Activation           *activationSetWire `json:"activation,omitempty"`
	DefaultTags          *tagSetWire        `json:"defaultTags,omitempty"`
}

type activationSetWire struct {
	Set struct {
		ActivationModule []string `json:"ActivationModule"`
	} `json:"set"`
}

type tagSetWire struct {
	Set struct {
		TagSimple []tagSimple `json:"TagSimple"`
	} `json:"set"`
}

func baseInputToWire(in ConnectorBaseInput) connectorWireBase {
	w := connectorWireBase{
		Name:                 in.Name,
		Description:          in.Description,
		RunFrequency:         in.RunFrequency,
		Disabled:             in.Disabled,
		IsRemediationEnabled: in.IsRemediationEnabled,
	}
	if len(in.ActivationModules) > 0 {
		w.Activation = &activationSetWire{}
		w.Activation.Set.ActivationModule = in.ActivationModules
	}
	if len(in.DefaultTagIDs) > 0 {
		w.DefaultTags = &tagSetWire{}
		for _, id := range in.DefaultTagIDs {
			w.DefaultTags.Set.TagSimple = append(w.DefaultTags.Set.TagSimple,
				tagSimple{ID: json.Number(id)})
		}
	}
	return w
}

// connectorRespBase is the response-side shared shape.
type connectorRespBase struct {
	ID                   json.Number     `json:"id"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	ConnectorState       string          `json:"connectorState"`
	Disabled             flexBool        `json:"disabled"`
	RunFrequency         json.Number     `json:"runFrequency"`
	IsRemediationEnabled flexBool        `json:"isRemediationEnabled"`
	LastSync             string          `json:"lastSync"`
	NextSync             string          `json:"nextSync"`
	LastError            string          `json:"lastError"`
	CloudViewUUID        string          `json:"cloudviewUuid"`
	Activation           json.RawMessage `json:"activation"`
	DefaultTags          json.RawMessage `json:"defaultTags"`
}

func (w *connectorRespBase) toBase() ConnectorBase {
	freq, _ := strconv.Atoi(w.RunFrequency.String())
	return ConnectorBase{
		ID:                   w.ID.String(),
		Name:                 w.Name,
		Description:          w.Description,
		State:                w.ConnectorState,
		Disabled:             bool(w.Disabled),
		RunFrequency:         freq,
		IsRemediationEnabled: bool(w.IsRemediationEnabled),
		LastSync:             w.LastSync,
		NextSync:             w.NextSync,
		LastError:            w.LastError,
		CloudViewUUID:        w.CloudViewUUID,
		ActivationModules:    decodeActivationModules(w.Activation),
		DefaultTags:          decodeTagRefs(w.DefaultTags),
	}
}

// flexBool decodes the API's booleans whether they arrive as true/false or as
// the strings "true"/"false" (the documented v3 response samples show the
// string form while requests take real booleans).
type flexBool bool

func (b *flexBool) UnmarshalJSON(data []byte) error {
	s := strings.Trim(strings.TrimSpace(string(data)), `"`)
	switch strings.ToLower(s) {
	case "true":
		*b = true
	case "false", "", "null":
		*b = false
	default:
		return fmt.Errorf("qualys qps: cannot decode %q as a boolean", s)
	}
	return nil
}

// decodeActivationModules reads the activation collection from a response.
//
// The docs confirm the XML nesting (<activation><list><ActivationModule>…)
// but show no JSON response sample containing it, so this accepts every
// plausible JSON rendering of that XML rather than asserting one:
// {"list": {"ActivationModule": [...]}} / {"list": [...]} / a bare array.
// An unrecognised shape yields nil rather than an error: the field is
// informational state, not something worth failing a refresh over.
func decodeActivationModules(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var direct []string
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct
	}
	var wrapped struct {
		List json.RawMessage `json:"list"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil || len(wrapped.List) == 0 {
		return nil
	}
	if err := json.Unmarshal(wrapped.List, &direct); err == nil {
		return direct
	}
	var named struct {
		ActivationModule []string `json:"ActivationModule"`
	}
	if err := json.Unmarshal(wrapped.List, &named); err == nil {
		return named.ActivationModule
	}
	return nil
}

// decodeTagRefs reads the defaultTags collection from a response, with the
// same shape tolerance (and the same silent-nil on unrecognised shapes) as
// decodeActivationModules.
func decodeTagRefs(raw json.RawMessage) []TagRef {
	if len(raw) == 0 {
		return nil
	}
	toRefs := func(simples []tagSimple) []TagRef {
		out := make([]TagRef, 0, len(simples))
		for _, s := range simples {
			out = append(out, TagRef{ID: s.ID.String(), Name: s.Name})
		}
		return out
	}
	var direct []tagSimple
	if err := json.Unmarshal(raw, &direct); err == nil {
		return toRefs(direct)
	}
	var wrapped struct {
		List json.RawMessage `json:"list"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil || len(wrapped.List) == 0 {
		return nil
	}
	if err := json.Unmarshal(wrapped.List, &direct); err == nil {
		return toRefs(direct)
	}
	var named struct {
		TagSimple []tagSimple `json:"TagSimple"`
	}
	if err := json.Unmarshal(wrapped.List, &named); err == nil {
		return toRefs(named.TagSimple)
	}
	return nil
}

// requireConnectorID validates an id destined for a /<verb>/am/.../<id> path.
// v3 ids are numeric; a CloudView v1 UUID here means the caller skipped the
// state-adoption step, and the error says so instead of letting the API
// return a confusing parse failure.
func requireConnectorID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: connector id is required")
	}
	if _, err := strconv.ParseInt(id, 10, 64); err != nil {
		return fmt.Errorf("qualys qps: connector id %q is not a numeric Connector v3 id "+
			"(CloudView v1 UUIDs are not valid here; re-import the resource or let "+
			"the provider adopt the id via cloudviewUuid)", id)
	}
	return nil
}
