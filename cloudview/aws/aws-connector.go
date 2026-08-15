package aws

import (
	"errors"
	"fmt"

	"github.com/go-resty/resty/v2"
)

const ApiPath = "/cloudview-api/rest/v1/aws/connectors"

type ConnectorService struct {
	client *resty.Client
}

func NewService(baseURL, username, password string) *ConnectorService {
	client := resty.New().
		SetHostURL(baseURL).
		SetBasicAuth(username, password).
		SetRetryCount(3)

	return &ConnectorService{client: client}
}

func (s *ConnectorService) Get(id string) (*Connector, error) {
	path := fmt.Sprintf("%s/%s", ApiPath, id)

	connector := &Connector{}

	resp, err := s.client.R().
		SetResult(connector).
		Get(path)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= 400 {
		return nil, errors.New(resp.String())
	}
	if len(connector.ServiceError) > 0 {
		return nil, errors.New(resp.String())
	}
	if len(connector.ErrorCode) > 0 {
		return nil, errors.New(resp.String())
	}

	return connector, err
}

func (s *ConnectorService) Create(opt *ConnectorConfig) (*Connector, error) {
	connector := new(Connector)

	resp, err := s.client.R().
		SetResult(connector).
		SetBody(opt).
		Post(ApiPath)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() >= 400 {
		return nil, errors.New(string(resp.Body()))
	}
	if len(connector.ServiceError) > 0 {
		return nil, errors.New(string(resp.Body()))
	}
	if len(connector.ErrorCode) > 0 {
		return nil, errors.New(string(resp.Body()))
	}

	return connector, err
}

func (s *ConnectorService) Update(id string, opt *ConnectorConfig) error {
	path := fmt.Sprintf("%s/%s", ApiPath, id)

	connector := new(Connector)

	resp, err := s.client.R().
		SetResult(connector).
		SetBody(opt).
		Put(path)
	if err != nil {
		return err
	}

	if resp.StatusCode() >= 400 {
		return errors.New(string(resp.Body()))
	}
	if len(connector.ServiceError) > 0 {
		return errors.New(string(resp.Body()))
	}
	if len(connector.ErrorCode) > 0 {
		return errors.New(string(resp.Body()))
	}
	return nil
}

func (s *ConnectorService) Delete(connectorIds []string) error {
	resp, err := s.client.R().
		SetBody(ConnectorIds{connectorIds}).
		SetHeader("Content-Type", "application/json").
		Delete(ApiPath)

	if err != nil {
		return err
	}

	if resp.StatusCode() >= 400 {
		return fmt.Errorf("delete failed with status code %d", resp.StatusCode())
	}
	return nil
}
