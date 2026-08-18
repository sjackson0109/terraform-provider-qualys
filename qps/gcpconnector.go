package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const gcpConnectorPath = "/qps/rest/3.0/%s/am/gcpassetdataconnector"

// GCPConnector is a GCP asset data connector (Connector v3).
type GCPConnector struct {
	ConnectorBase

	// ProjectID is the only credential field the API ever returns; the rest
	// of the service-account key is write-only.
	ProjectID string
}

// GCPConnectorInput is the desired state of a GCP connector.
type GCPConnectorInput struct {
	ConnectorBaseInput

	// CredentialsJSON is the GCP service-account key file, verbatim. The v3
	// JSON authRecord uses the key file's own field names (project_id,
	// private_key, client_email, ...; doc 12), so the file is embedded
	// field-for-field rather than re-mapped.
	CredentialsJSON string
}

type gcpConnectorResp struct {
	connectorRespBase

	AuthRecord *struct {
		// The XML samples call this projectId, the JSON samples project_id
		// (the raw key-file name). Reads here are JSON; accept both.
		ProjectID      string `json:"projectId"`
		ProjectIDSnake string `json:"project_id"`
	} `json:"authRecord"`
}

func (w *gcpConnectorResp) toConnector() *GCPConnector {
	out := &GCPConnector{ConnectorBase: w.toBase()}
	if w.AuthRecord != nil {
		out.ProjectID = w.AuthRecord.ProjectID
		if out.ProjectID == "" {
			out.ProjectID = w.AuthRecord.ProjectIDSnake
		}
	}
	return out
}

// gcpInputToWire builds the request body. The credentials file is decoded
// and embedded as the authRecord object; a file that is not a JSON object
// fails here, at plan/apply time, rather than as an opaque API rejection.
func gcpInputToWire(in GCPConnectorInput) (map[string]interface{}, error) {
	body := map[string]interface{}{}

	base := baseInputToWire(in.ConnectorBaseInput)
	raw, err := json.Marshal(base)
	if err != nil {
		return nil, fmt.Errorf("qualys qps: encoding connector fields: %w", err)
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("qualys qps: encoding connector fields: %w", err)
	}

	// The GCP create API documents type as required, unlike AWS and Azure.
	body["type"] = "GCP"

	if strings.TrimSpace(in.CredentialsJSON) != "" {
		var authRecord map[string]interface{}
		if err := json.Unmarshal([]byte(in.CredentialsJSON), &authRecord); err != nil {
			return nil, fmt.Errorf("qualys qps: gcp credentials are not a JSON object "+
				"(expected a service-account key file): %w", err)
		}
		body["authRecord"] = authRecord
	}
	return body, nil
}

func decodeGCPConnectors(raw json.RawMessage) ([]*GCPConnector, error) {
	var out []*GCPConnector
	for _, item := range decodeListOrSingle(raw) {
		var wrap struct {
			Connector *gcpConnectorResp `json:"GcpAssetDataConnector"`
		}
		if err := json.Unmarshal(item, &wrap); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding GcpAssetDataConnector: %w", err)
		}
		if wrap.Connector != nil {
			out = append(out, wrap.Connector.toConnector())
		}
	}
	return out, nil
}

// CreateGCPConnector creates a GCP connector and returns it.
//
// The GCP create API documents name, authRecord and runFrequency as required
// (stricter than AWS/Azure); the input validation here mirrors that.
func (c *Client) CreateGCPConnector(ctx context.Context, in GCPConnectorInput) (*GCPConnector, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("qualys qps: connector name is required")
	}
	if strings.TrimSpace(in.CredentialsJSON) == "" {
		return nil, fmt.Errorf("qualys qps: gcp credentials are required")
	}
	if in.RunFrequency <= 0 {
		return nil, fmt.Errorf("qualys qps: runFrequency is required for GCP connectors")
	}
	body, err := gcpInputToWire(in)
	if err != nil {
		return nil, err
	}
	var resp ServiceResponse
	err = c.call(ctx, http.MethodPost, fmt.Sprintf(gcpConnectorPath, "create"),
		&ServiceRequest{Data: map[string]interface{}{"GcpAssetDataConnector": body}},
		&resp, true)
	if err != nil {
		return nil, err
	}
	conns, err := decodeGCPConnectors(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(conns) == 0 {
		return nil, fmt.Errorf("qualys qps: GCP connector was created but the API returned " +
			"no connector data; import it by id to bring it under management")
	}
	return conns[0], nil
}

// GetGCPConnector returns one connector by numeric id, or ErrNotFound.
func (c *Client) GetGCPConnector(ctx context.Context, id string) (*GCPConnector, error) {
	if err := requireConnectorID(id); err != nil {
		return nil, err
	}
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, fmt.Sprintf(gcpConnectorPath, "get")+"/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	conns, err := decodeGCPConnectors(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(conns) == 0 {
		return nil, fmt.Errorf("qualys qps: GCP connector %s: %w", id, ErrNotFound)
	}
	return conns[0], nil
}

// UpdateGCPConnector applies desired state to an existing connector. The API
// accepts an authRecord carrying only projectId when the key is unchanged, so
// CredentialsJSON may be empty here; the API responds with the id only, so
// callers re-Get to observe the result.
func (c *Client) UpdateGCPConnector(ctx context.Context, id string, in GCPConnectorInput) error {
	if err := requireConnectorID(id); err != nil {
		return err
	}
	body, err := gcpInputToWire(in)
	if err != nil {
		return err
	}
	return c.call(ctx, http.MethodPost, fmt.Sprintf(gcpConnectorPath, "update")+"/"+id,
		&ServiceRequest{Data: map[string]interface{}{"GcpAssetDataConnector": body}},
		nil, false)
}

// DeleteGCPConnector removes a connector by numeric id.
func (c *Client) DeleteGCPConnector(ctx context.Context, id string) error {
	if err := requireConnectorID(id); err != nil {
		return err
	}
	return c.call(ctx, http.MethodPost, fmt.Sprintf(gcpConnectorPath, "delete")+"/"+id,
		nil, nil, true)
}

// SearchGCPConnectors returns connectors matching the criteria (nil for all).
func (c *Client) SearchGCPConnectors(ctx context.Context, filters *Filters) ([]*GCPConnector, error) {
	var out []*GCPConnector
	err := c.SearchAll(ctx, fmt.Sprintf(gcpConnectorPath, "search"), filters, 100, 0,
		func(page json.RawMessage) error {
			conns, err := decodeGCPConnectors(page)
			if err != nil {
				return err
			}
			out = append(out, conns...)
			return nil
		})
	return out, err
}

// FindGCPConnectorByCloudViewUUID locates the v3 connector that corresponds
// to a deprecated CloudView v1 connector UUID, or ErrNotFound. The match is
// client-side: cloudviewUuid is not a documented search filter field.
func (c *Client) FindGCPConnectorByCloudViewUUID(ctx context.Context, uuid string) (*GCPConnector, error) {
	conns, err := c.SearchGCPConnectors(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, conn := range conns {
		if strings.EqualFold(conn.CloudViewUUID, uuid) {
			return conn, nil
		}
	}
	return nil, fmt.Errorf("qualys qps: no GCP connector with cloudviewUuid %s: %w", uuid, ErrNotFound)
}
