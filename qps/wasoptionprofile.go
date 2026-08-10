package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Performance levels accepted by a WAS option profile.
const (
	WASPerformanceLow    = "LOW"
	WASPerformanceMedium = "MEDIUM"
	WASPerformanceHigh   = "HIGH"
)

// WASOptionProfile is a WAS scan option profile.
type WASOptionProfile struct {
	ID                       string
	Name                     string
	Comments                 string
	MaxCrawlRequests         int
	Performance              string
	BruteforceOption         string
	TimeoutErrorThreshold    int
	UnexpectedErrorThreshold int
	DetectCreditCardNumbers  bool
	DetectSocialSecurityNums bool
}

// WASOptionProfileInput is the desired state of a WAS option profile.
type WASOptionProfileInput struct {
	Name                     string
	Comments                 string
	MaxCrawlRequests         int
	Performance              string
	BruteforceOption         string
	TimeoutErrorThreshold    int
	UnexpectedErrorThreshold int
	DetectCreditCardNumbers  bool
	DetectSocialSecurityNums bool
}

type sensitiveContentWire struct {
	CreditCardNumber     bool `json:"creditCardNumber"`
	SocialSecurityNumber bool `json:"socialSecurityNumber"`
}

type wasOptionProfileWire struct {
	ID                     json.Number           `json:"id,omitempty"`
	Name                   string                `json:"name,omitempty"`
	Comments               string                `json:"comments,omitempty"`
	MaxCrawlRequests       int                   `json:"maxCrawlRequests,omitempty"`
	Performance            string                `json:"performance,omitempty"`
	BruteforceOption       string                `json:"bruteforceOption,omitempty"`
	TimeoutErrorThreshold  int                   `json:"timeoutErrorThreshold,omitempty"`
	UnexpectedErrThreshold int                   `json:"unexpectedErrorThreshold,omitempty"`
	SensitiveContent       *sensitiveContentWire `json:"sensitiveContent,omitempty"`
}

// wasOptionProfileData is the "data" envelope. The wrapper key is
// OptionProfile — corrected from an earlier WasOptionProfile guess (this
// API's naming convention held for WasScanSchedule but not here) after a
// user-supplied create example showed <OptionProfile> verbatim.
type wasOptionProfileData struct {
	OptionProfile *wasOptionProfileWire `json:"OptionProfile,omitempty"`
}

func (w *wasOptionProfileWire) toProfile() *WASOptionProfile {
	out := &WASOptionProfile{
		ID:                       w.ID.String(),
		Name:                     w.Name,
		Comments:                 w.Comments,
		MaxCrawlRequests:         w.MaxCrawlRequests,
		Performance:              w.Performance,
		BruteforceOption:         w.BruteforceOption,
		TimeoutErrorThreshold:    w.TimeoutErrorThreshold,
		UnexpectedErrorThreshold: w.UnexpectedErrThreshold,
	}
	if w.SensitiveContent != nil {
		out.DetectCreditCardNumbers = w.SensitiveContent.CreditCardNumber
		out.DetectSocialSecurityNums = w.SensitiveContent.SocialSecurityNumber
	}
	return out
}

func wasOptionProfileInputToWire(in WASOptionProfileInput) *wasOptionProfileWire {
	return &wasOptionProfileWire{
		Name:                   in.Name,
		Comments:               in.Comments,
		MaxCrawlRequests:       in.MaxCrawlRequests,
		Performance:            in.Performance,
		BruteforceOption:       in.BruteforceOption,
		TimeoutErrorThreshold:  in.TimeoutErrorThreshold,
		UnexpectedErrThreshold: in.UnexpectedErrorThreshold,
		SensitiveContent: &sensitiveContentWire{
			CreditCardNumber:     in.DetectCreditCardNumbers,
			SocialSecurityNumber: in.DetectSocialSecurityNums,
		},
	}
}

// CreateWASOptionProfile creates a WAS option profile and returns it.
func (c *Client) CreateWASOptionProfile(ctx context.Context, in WASOptionProfileInput) (*WASOptionProfile, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("qualys qps: WAS option profile name is required")
	}

	var resp ServiceResponse
	err := c.call(ctx, http.MethodPost, "/qps/rest/3.0/create/was/optionprofile",
		&ServiceRequest{Data: wasOptionProfileData{OptionProfile: wasOptionProfileInputToWire(in)}}, &resp, true)
	if err != nil {
		return nil, err
	}
	profiles, err := decodeWASOptionProfiles(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS option profile was created but the API returned no data; " +
			"import it by ID to bring it under management")
	}
	return profiles[0], nil
}

// UpdateWASOptionProfile applies desired state to an existing WAS option profile.
//
// Every managed field is sent unconditionally, bypassing the omitempty wire
// struct used for encoding: several of these fields (the thresholds, the
// sensitive-content booleans) are meaningful at their zero value, and
// omitting them on update would leave Qualys's previous value in place
// instead of clearing it — the same class of bug fixed for asset-group CVSS
// fields.
func (c *Client) UpdateWASOptionProfile(ctx context.Context, id string, in WASOptionProfileInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys qps: WAS option profile id is required for update")
	}
	profile := map[string]interface{}{
		"name":                     in.Name,
		"comments":                 in.Comments,
		"maxCrawlRequests":         in.MaxCrawlRequests,
		"performance":              in.Performance,
		"bruteforceOption":         in.BruteforceOption,
		"timeoutErrorThreshold":    in.TimeoutErrorThreshold,
		"unexpectedErrorThreshold": in.UnexpectedErrorThreshold,
		"sensitiveContent": map[string]interface{}{
			"creditCardNumber":     in.DetectCreditCardNumbers,
			"socialSecurityNumber": in.DetectSocialSecurityNums,
		},
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/update/was/optionprofile/"+id,
		&ServiceRequest{Data: map[string]interface{}{"OptionProfile": profile}}, nil, false)
}

// DeleteWASOptionProfile removes a WAS option profile.
func (c *Client) DeleteWASOptionProfile(ctx context.Context, id string) error {
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/delete/was/optionprofile/"+id, nil, nil, true)
}

// GetWASOptionProfile returns one WAS option profile, or ErrNotFound.
func (c *Client) GetWASOptionProfile(ctx context.Context, id string) (*WASOptionProfile, error) {
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, "/qps/rest/3.0/get/was/optionprofile/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	profiles, err := decodeWASOptionProfiles(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, fmt.Errorf("qualys qps: WAS option profile %s: %w", id, ErrNotFound)
	}
	return profiles[0], nil
}

// SearchWASOptionProfiles returns WAS option profiles matching filters, following pagination.
func (c *Client) SearchWASOptionProfiles(ctx context.Context, filters *Filters) ([]*WASOptionProfile, error) {
	var all []*WASOptionProfile
	err := c.SearchAll(ctx, "/qps/rest/3.0/search/was/optionprofile", filters, 100, 0,
		func(raw json.RawMessage) error {
			profiles, err := decodeWASOptionProfiles(raw)
			if err != nil {
				return err
			}
			all = append(all, profiles...)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return all, nil
}

func decodeWASOptionProfiles(raw json.RawMessage) ([]*WASOptionProfile, error) {
	items := decodeListOrSingle(raw)
	out := make([]*WASOptionProfile, 0, len(items))
	for _, item := range items {
		var d wasOptionProfileData
		if err := json.Unmarshal(item, &d); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding WAS option profile data: %w", err)
		}
		if d.OptionProfile != nil {
			out = append(out, d.OptionProfile.toProfile())
		}
	}
	return out, nil
}
