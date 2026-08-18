package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const awsConnectorPath = "/qps/rest/3.0/%s/am/awsassetdataconnector"

// AWSConnector is an AWS asset data connector (Connector v3).
type AWSConnector struct {
	ConnectorBase

	ARN        string
	ExternalID string
	AllRegions bool
	IsGovCloud bool

	// AWSAccountID is the scanned account; QualysAWSAccountID is the Qualys
	// account that assumes the role. Both are read-only.
	AWSAccountID       string
	QualysAWSAccountID string
	IsChinaConfigured  bool
}

// AWSConnectorInput is the desired state of an AWS connector.
type AWSConnectorInput struct {
	ConnectorBaseInput

	// ARN of the IAM role Qualys assumes; ExternalID must match the role's
	// trust policy condition.
	ARN        string
	ExternalID string

	// AllRegions nil omits the field (server default true per doc samples);
	// setting it false requires region endpoints, which this client does not
	// carry yet (doc 12 defers the endpoints block).
	AllRegions *bool

	IsGovCloud *bool
}

type awsConnectorWire struct {
	connectorWireBase

	ARN                  string `json:"arn,omitempty"`
	ExternalID           string `json:"externalId,omitempty"`
	AllRegions           *bool  `json:"allRegions,omitempty"`
	IsGovCloudConfigured *bool  `json:"isGovCloudConfigured,omitempty"`
}

type awsConnectorResp struct {
	connectorRespBase

	ARN                  string   `json:"arn"`
	ExternalID           string   `json:"externalId"`
	AllRegions           flexBool `json:"allRegions"`
	IsGovCloudConfigured flexBool `json:"isGovCloudConfigured"`
	IsChinaConfigured    flexBool `json:"isChinaConfigured"`
	AWSAccountID         string   `json:"awsAccountId"`
	QualysAWSAccountID   string   `json:"qualysAwsAccountId"`
}

func (w *awsConnectorResp) toConnector() *AWSConnector {
	return &AWSConnector{
		ConnectorBase:      w.toBase(),
		ARN:                w.ARN,
		ExternalID:         w.ExternalID,
		AllRegions:         bool(w.AllRegions),
		IsGovCloud:         bool(w.IsGovCloudConfigured),
		IsChinaConfigured:  bool(w.IsChinaConfigured),
		AWSAccountID:       w.AWSAccountID,
		QualysAWSAccountID: w.QualysAWSAccountID,
	}
}

func awsInputToWire(in AWSConnectorInput) *awsConnectorWire {
	return &awsConnectorWire{
		connectorWireBase:    baseInputToWire(in.ConnectorBaseInput),
		ARN:                  in.ARN,
		ExternalID:           in.ExternalID,
		AllRegions:           in.AllRegions,
		IsGovCloudConfigured: in.IsGovCloud,
	}
}

func decodeAWSConnectors(raw json.RawMessage) ([]*AWSConnector, error) {
	var out []*AWSConnector
	for _, item := range decodeListOrSingle(raw) {
		var wrap struct {
			Connector *awsConnectorResp `json:"AwsAssetDataConnector"`
		}
		if err := json.Unmarshal(item, &wrap); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding AwsAssetDataConnector: %w", err)
		}
		if wrap.Connector != nil {
			out = append(out, wrap.Connector.toConnector())
		}
	}
	return out, nil
}

// CreateAWSConnector creates an AWS connector and returns it.
func (c *Client) CreateAWSConnector(ctx context.Context, in AWSConnectorInput) (*AWSConnector, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("qualys qps: connector name is required")
	}
	var resp ServiceResponse
	err := c.call(ctx, http.MethodPost, fmt.Sprintf(awsConnectorPath, "create"),
		&ServiceRequest{Data: map[string]interface{}{"AwsAssetDataConnector": awsInputToWire(in)}},
		&resp, true)
	if err != nil {
		return nil, err
	}
	conns, err := decodeAWSConnectors(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(conns) == 0 {
		return nil, fmt.Errorf("qualys qps: AWS connector was created but the API returned " +
			"no connector data; import it by id to bring it under management")
	}
	return conns[0], nil
}

// GetAWSConnector returns one connector by numeric id, or ErrNotFound.
func (c *Client) GetAWSConnector(ctx context.Context, id string) (*AWSConnector, error) {
	if err := requireConnectorID(id); err != nil {
		return nil, err
	}
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, fmt.Sprintf(awsConnectorPath, "get")+"/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	conns, err := decodeAWSConnectors(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(conns) == 0 {
		return nil, fmt.Errorf("qualys qps: AWS connector %s: %w", id, ErrNotFound)
	}
	return conns[0], nil
}

// UpdateAWSConnector applies desired state to an existing connector. The API
// responds with the id only, so callers re-Get to observe the result.
func (c *Client) UpdateAWSConnector(ctx context.Context, id string, in AWSConnectorInput) error {
	if err := requireConnectorID(id); err != nil {
		return err
	}
	return c.call(ctx, http.MethodPost, fmt.Sprintf(awsConnectorPath, "update")+"/"+id,
		&ServiceRequest{Data: map[string]interface{}{"AwsAssetDataConnector": awsInputToWire(in)}},
		nil, false)
}

// DeleteAWSConnector removes a connector by numeric id.
func (c *Client) DeleteAWSConnector(ctx context.Context, id string) error {
	if err := requireConnectorID(id); err != nil {
		return err
	}
	return c.call(ctx, http.MethodPost, fmt.Sprintf(awsConnectorPath, "delete")+"/"+id,
		nil, nil, true)
}

// SearchAWSConnectors returns connectors matching the criteria (nil for all).
func (c *Client) SearchAWSConnectors(ctx context.Context, filters *Filters) ([]*AWSConnector, error) {
	var out []*AWSConnector
	err := c.SearchAll(ctx, fmt.Sprintf(awsConnectorPath, "search"), filters, 100, 0,
		func(page json.RawMessage) error {
			conns, err := decodeAWSConnectors(page)
			if err != nil {
				return err
			}
			out = append(out, conns...)
			return nil
		})
	return out, err
}

// FindAWSConnectorByCloudViewUUID locates the v3 connector that corresponds
// to a deprecated CloudView v1 connector UUID, or ErrNotFound.
//
// cloudviewUuid is not a documented search filter field, so the match is
// client-side over the full listing.
func (c *Client) FindAWSConnectorByCloudViewUUID(ctx context.Context, uuid string) (*AWSConnector, error) {
	conns, err := c.SearchAWSConnectors(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, conn := range conns {
		if strings.EqualFold(conn.CloudViewUUID, uuid) {
			return conn, nil
		}
	}
	return nil, fmt.Errorf("qualys qps: no AWS connector with cloudviewUuid %s: %w", uuid, ErrNotFound)
}
