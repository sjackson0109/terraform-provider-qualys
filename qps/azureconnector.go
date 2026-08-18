package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const azureConnectorPath = "/qps/rest/3.0/%s/am/azureassetdataconnector"

// AzureConnector is an Azure asset data connector (Connector v3).
type AzureConnector struct {
	ConnectorBase

	// Service-principal identity. The API returns these on get but never
	// returns the client secret (create echoes an empty authRecord), so
	// AuthenticationKey is absent here by design.
	ApplicationID  string
	DirectoryID    string
	SubscriptionID string

	SubscriptionName string
	IsGovCloud       bool
}

// AzureConnectorInput is the desired state of an Azure connector.
type AzureConnectorInput struct {
	ConnectorBaseInput

	// The Azure AD service principal Qualys authenticates as. All four are
	// carried inside the request's authRecord element.
	ApplicationID     string
	DirectoryID       string
	SubscriptionID    string
	AuthenticationKey string
}

type azureAuthRecordWire struct {
	ApplicationID     string `json:"applicationId,omitempty"`
	DirectoryID       string `json:"directoryId,omitempty"`
	SubscriptionID    string `json:"subscriptionId,omitempty"`
	AuthenticationKey string `json:"authenticationKey,omitempty"`
}

type azureConnectorWire struct {
	connectorWireBase

	AuthRecord *azureAuthRecordWire `json:"authRecord,omitempty"`
}

type azureConnectorResp struct {
	connectorRespBase

	AuthRecord *struct {
		ApplicationID  string `json:"applicationId"`
		DirectoryID    string `json:"directoryId"`
		SubscriptionID string `json:"subscriptionId"`
	} `json:"authRecord"`
	SubscriptionName     string   `json:"subscriptionName"`
	IsGovCloudConfigured flexBool `json:"isGovCloudConfigured"`
}

func (w *azureConnectorResp) toConnector() *AzureConnector {
	out := &AzureConnector{
		ConnectorBase:    w.toBase(),
		SubscriptionName: w.SubscriptionName,
		IsGovCloud:       bool(w.IsGovCloudConfigured),
	}
	if w.AuthRecord != nil {
		out.ApplicationID = w.AuthRecord.ApplicationID
		out.DirectoryID = w.AuthRecord.DirectoryID
		out.SubscriptionID = w.AuthRecord.SubscriptionID
	}
	return out
}

func azureInputToWire(in AzureConnectorInput) *azureConnectorWire {
	w := &azureConnectorWire{
		connectorWireBase: baseInputToWire(in.ConnectorBaseInput),
	}
	if in.ApplicationID != "" || in.DirectoryID != "" ||
		in.SubscriptionID != "" || in.AuthenticationKey != "" {
		w.AuthRecord = &azureAuthRecordWire{
			ApplicationID:     in.ApplicationID,
			DirectoryID:       in.DirectoryID,
			SubscriptionID:    in.SubscriptionID,
			AuthenticationKey: in.AuthenticationKey,
		}
	}
	return w
}

func decodeAzureConnectors(raw json.RawMessage) ([]*AzureConnector, error) {
	var out []*AzureConnector
	for _, item := range decodeListOrSingle(raw) {
		var wrap struct {
			Connector *azureConnectorResp `json:"AzureAssetDataConnector"`
		}
		if err := json.Unmarshal(item, &wrap); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding AzureAssetDataConnector: %w", err)
		}
		if wrap.Connector != nil {
			out = append(out, wrap.Connector.toConnector())
		}
	}
	return out, nil
}

// CreateAzureConnector creates an Azure connector and returns it.
func (c *Client) CreateAzureConnector(ctx context.Context, in AzureConnectorInput) (*AzureConnector, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("qualys qps: connector name is required")
	}
	var resp ServiceResponse
	err := c.call(ctx, http.MethodPost, fmt.Sprintf(azureConnectorPath, "create"),
		&ServiceRequest{Data: map[string]interface{}{"AzureAssetDataConnector": azureInputToWire(in)}},
		&resp, true)
	if err != nil {
		return nil, err
	}
	conns, err := decodeAzureConnectors(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(conns) == 0 {
		return nil, fmt.Errorf("qualys qps: Azure connector was created but the API returned " +
			"no connector data; import it by id to bring it under management")
	}
	return conns[0], nil
}

// GetAzureConnector returns one connector by numeric id, or ErrNotFound.
func (c *Client) GetAzureConnector(ctx context.Context, id string) (*AzureConnector, error) {
	if err := requireConnectorID(id); err != nil {
		return nil, err
	}
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, fmt.Sprintf(azureConnectorPath, "get")+"/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	conns, err := decodeAzureConnectors(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(conns) == 0 {
		return nil, fmt.Errorf("qualys qps: Azure connector %s: %w", id, ErrNotFound)
	}
	return conns[0], nil
}

// UpdateAzureConnector applies desired state to an existing connector. The
// API responds with the id only, so callers re-Get to observe the result.
func (c *Client) UpdateAzureConnector(ctx context.Context, id string, in AzureConnectorInput) error {
	if err := requireConnectorID(id); err != nil {
		return err
	}
	return c.call(ctx, http.MethodPost, fmt.Sprintf(azureConnectorPath, "update")+"/"+id,
		&ServiceRequest{Data: map[string]interface{}{"AzureAssetDataConnector": azureInputToWire(in)}},
		nil, false)
}

// DeleteAzureConnector removes a connector by numeric id.
func (c *Client) DeleteAzureConnector(ctx context.Context, id string) error {
	if err := requireConnectorID(id); err != nil {
		return err
	}
	return c.call(ctx, http.MethodPost, fmt.Sprintf(azureConnectorPath, "delete")+"/"+id,
		nil, nil, true)
}

// SearchAzureConnectors returns connectors matching the criteria (nil for all).
func (c *Client) SearchAzureConnectors(ctx context.Context, filters *Filters) ([]*AzureConnector, error) {
	var out []*AzureConnector
	err := c.SearchAll(ctx, fmt.Sprintf(azureConnectorPath, "search"), filters, 100, 0,
		func(page json.RawMessage) error {
			conns, err := decodeAzureConnectors(page)
			if err != nil {
				return err
			}
			out = append(out, conns...)
			return nil
		})
	return out, err
}

// FindAzureConnectorByCloudViewUUID locates the v3 connector that corresponds
// to a deprecated CloudView v1 connector UUID, or ErrNotFound. The match is
// client-side: cloudviewUuid is not a documented search filter field.
func (c *Client) FindAzureConnectorByCloudViewUUID(ctx context.Context, uuid string) (*AzureConnector, error) {
	conns, err := c.SearchAzureConnectors(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, conn := range conns {
		if strings.EqualFold(conn.CloudViewUUID, uuid) {
			return conn, nil
		}
	}
	return nil, fmt.Errorf("qualys qps: no Azure connector with cloudviewUuid %s: %w", uuid, ErrNotFound)
}
