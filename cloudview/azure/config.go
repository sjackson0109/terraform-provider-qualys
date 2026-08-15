package azure

type ConnectorConfig struct {
	Name              string `json:"name,omitempty"`
	Description       string `json:"description,omitempty"`
	ApplicationID     string `json:"applicationId,omitempty"`
	DirectoryID       string `json:"directoryId,omitempty"`
	SubscriptionID    string `json:"subscriptionId,omitempty"`
	AuthenticationKey string `json:"authenticationKey,omitempty"`
	IsGovCloud        bool   `json:"isGovCloud"`
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

func (c *ConnectorConfig) WithApplicationID(applicationID string) *ConnectorConfig {
	c.ApplicationID = applicationID
	return c
}

func (c *ConnectorConfig) WithDirectoryID(directoryID string) *ConnectorConfig {
	c.DirectoryID = directoryID
	return c
}

func (c *ConnectorConfig) WithSubscriptionID(subscriptionID string) *ConnectorConfig {
	c.SubscriptionID = subscriptionID
	return c
}

func (c *ConnectorConfig) WithAuthenticationKey(authenticationKey string) *ConnectorConfig {
	c.AuthenticationKey = authenticationKey
	return c
}

func (c *ConnectorConfig) WithIsGovCloud(isGovCloud bool) *ConnectorConfig {
	c.IsGovCloud = isGovCloud
	return c
}
