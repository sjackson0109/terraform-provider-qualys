package aws

type Connector struct {
	ErrorResponse

	ConnectorID         string            `json:"connectorId"`
	Name                string            `json:"name"`
	Description         string            `json:"description"`
	Provider            string            `json:"provider"`
	State               string            `json:"state"`
	TotalAssets         int               `json:"totalAssets"`
	LastSyncedOn        string            `json:"lastSyncedOn"`
	NextSyncedOn        string            `json:"nextSyncedOn"`
	RemediationEnabled  bool              `json:"remediationEnabled"`
	IsGovCloud          bool              `json:"isGovCloud"`
	IsChinaRegion       bool              `json:"isChinaRegion"`
	AWSAccountID        string            `json:"awsAccountId"`
	AccountAlias        string            `json:"accountAlias"`
	IsDisabled          bool              `json:"isDisabled"`
	Tags                []Tag             `json:"tags"`
	PollingFrequency    *PollingFrequency `json:"pollingFrequency"`
	BaseAccountID       string            `json:"baseAccountId"`
	ExternalID          string            `json:"externalId"`
	ARN                 string            `json:"arn"`
	PortalConnectorUUID string            `json:"portalConnectorUuid"`
	IsPortalConnector   bool              `json:"isPortalConnector"`
}

type Tag struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

type PollingFrequency struct {
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
}

type ErrorResponse struct {
	Timestamp    string `json:"timestamp"`
	Status       int    `json:"status"`
	ServiceError string `json:"error"`
	ErrorCode    string `json:"errorCode"`
	Message      string `json:"message"`
	Path         string `json:"path"`
}

type Pageable struct {
	Offset     int  `json:"offset"`
	PageNumber int  `json:"pageNumber"`
	PageSize   int  `json:"pageSize"`
	Paged      bool `json:"paged"`
	Sort       Sort `json:"sort"`
	UnPaged    bool `json:"unpaged"`
}

type Sort struct {
	Sorted   bool `json:"sorted"`
	Unsorted bool `json:"unsorted"`
}

type ConnectorList struct {
	ErrorResponse

	List     []Connector `json:"content"`
	Pageable *Pageable   `json:"pageable"`
	IsFirst  bool        `json:"first"`
	IsLast   bool        `json:"last"`
	Number   int         `json:"number"`
	Total    int         `json:"numberOfElements"`
}

type ConnectorIds struct {
	ConnectorIds []string `json:"connectorIds,omitempty"`
}
