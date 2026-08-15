package aws

type ConnectorConfig struct {
	Name              string `json:"name,omitempty"`
	Description       string `json:"description,omitempty"`
	ARN               string `json:"arn,omitempty"`
	ExternalID        string `json:"externalId,omitempty"`
	IsPortalConnector bool   `json:"isPortalConnector"`
	IsGovCloud        bool   `json:"isGovCloud"`
	IsChinaRegion     bool   `json:"isChinaRegion"`
}

func NewConnectorConfig() *ConnectorConfig {
	return &ConnectorConfig{}
}

func (c *ConnectorConfig) WithName(name string) *ConnectorConfig {
	c.Name = name
	return c
}

func (c *ConnectorConfig) WithDescription(desc string) *ConnectorConfig {
	c.Description = desc
	return c
}

func (c *ConnectorConfig) WithARN(arn string) *ConnectorConfig {
	c.ARN = arn
	return c
}

func (c *ConnectorConfig) WithExternalID(externalID string) *ConnectorConfig {
	c.ExternalID = externalID
	return c
}

func (c *ConnectorConfig) WithIsPortalConnector(isPortalConnector bool) *ConnectorConfig {
	c.IsPortalConnector = isPortalConnector
	return c
}

func (c *ConnectorConfig) WithIsGovCloud(isGovCloud bool) *ConnectorConfig {
	c.IsGovCloud = isGovCloud
	return c
}

func (c *ConnectorConfig) WithIsChinaRegion(isChinaRegion bool) *ConnectorConfig {
	c.IsChinaRegion = isChinaRegion
	return c
}
