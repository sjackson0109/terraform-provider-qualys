package vmdr

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
)

// User is a subscription user account, as returned by the legacy V1
// user_list.php script.
//
// Evidence tier: Confirmed. A user-supplied primary-source excerpt from the
// official "List Users" API page quotes the DTD name
// (user_list_output.dtd), the full request parameter table, and a complete
// sample response covering every field this type exposes.
//
// The same excerpt is the first direct evidence this project has for how
// visibility is actually scoped: the sample Manager-role user carries no
// ASSIGNED_ASSET_GROUPS element at all, while the sample Scanner-role user
// does. That is consistent with visibility being driven by explicit asset
// group assignment rather than automatically by BUSINESS_UNIT membership —
// BUSINESS_UNIT reads more like an organisational/reporting label per user
// (its value is literally "Unassigned" on both sample users). This is a
// reasonable inference from the one excerpt available, not something the
// excerpt states outright, and it is not yet corroborated by a Business
// Unit API document — treat it as a working hypothesis, not Confirmed.
type User struct {
	ID    string
	Login string

	FirstName string
	LastName  string
	Title     string
	Phone     string
	Fax       string
	Email     string
	Company   string
	Address1  string
	Address2  string
	City      string
	Country   string
	State     string
	ZipCode   string
	// TimeZoneCode is the same code space GetKnowledgeBaseEntries/
	// qualys_vm_scan_schedule's time_zone_code accepts, per doc 03 §8 — Confirmed
	// here only insofar as the field exists and carries values like "Auto" or
	// "US-CA"; the full code enum remains Unverified (doc 08).
	TimeZoneCode string

	Status         string
	CreationDate   string
	LastLoginDate  string
	Role           string
	BusinessUnit   string
	UnitManagerPOC bool
	ManagerPOC     bool

	// AssignedAssetGroups are the titles of asset groups this user can see.
	// See the type doc comment above: this, not BusinessUnit, looks like the
	// actual visibility-scoping mechanism.
	AssignedAssetGroups []string

	// Permissions is decoded generically (whatever PERMISSIONS child
	// elements the response actually carries), not a fixed field list. The
	// source excerpt shows five (CREATE_OPTION_PROFILES, PURGE_INFO,
	// ADD_ASSETS, EDIT_REMEDIATION_POLICY, EDIT_AUTH_RECORDS) but explicitly
	// calls the full set "extended permissions" without enumerating it —
	// hardcoding just the five shown would silently drop any others a real
	// tenant returns.
	Permissions map[string]bool

	// Notification preferences. LatestVuln/Map/Scan are enum-like strings
	// (e.g. "weekly", "ags") per the source sample; DailyTickets is a 0/1
	// flag.
	NotifyLatestVuln   string
	NotifyMap          string
	NotifyScan         string
	NotifyDailyTickets bool
}

type userPermissionFlag struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type userListOutput struct {
	Users []struct {
		UserLogin   string `xml:"USER_LOGIN"`
		UserID      string `xml:"USER_ID"`
		ContactInfo struct {
			FirstName    string `xml:"FIRSTNAME"`
			LastName     string `xml:"LASTNAME"`
			Title        string `xml:"TITLE"`
			Phone        string `xml:"PHONE"`
			Fax          string `xml:"FAX"`
			Email        string `xml:"EMAIL"`
			Company      string `xml:"COMPANY"`
			Address1     string `xml:"ADDRESS1"`
			Address2     string `xml:"ADDRESS2"`
			City         string `xml:"CITY"`
			Country      string `xml:"COUNTRY"`
			State        string `xml:"STATE"`
			ZipCode      string `xml:"ZIP_CODE"`
			TimeZoneCode string `xml:"TIME_ZONE_CODE"`
		} `xml:"CONTACT_INFO"`
		AssignedAssetGroups struct {
			Titles []string `xml:"ASSET_GROUP_TITLE"`
		} `xml:"ASSIGNED_ASSET_GROUPS"`
		UserStatus       string `xml:"USER_STATUS"`
		CreationDate     string `xml:"CREATION_DATE"`
		LastLoginDate    string `xml:"LAST_LOGIN_DATE"`
		UserRole         string `xml:"USER_ROLE"`
		BusinessUnit     string `xml:"BUSINESS_UNIT"`
		UnitManagerPOC   string `xml:"UNIT_MANAGER_POC"`
		ManagerPOC       string `xml:"MANAGER_POC"`
		UIInterfaceStyle string `xml:"UI_INTERFACE_STYLE"`
		Permissions      struct {
			Flags []userPermissionFlag `xml:",any"`
		} `xml:"PERMISSIONS"`
		Notifications struct {
			LatestVuln   string `xml:"LATEST_VULN"`
			Map          string `xml:"MAP"`
			Scan         string `xml:"SCAN"`
			DailyTickets string `xml:"DAILY_TICKETS"`
		} `xml:"NOTIFICATIONS"`
	} `xml:"USER_LIST>USER"`
}

func (o *userListOutput) users() []*User {
	out := make([]*User, 0, len(o.Users))
	for _, u := range o.Users {
		perms := make(map[string]bool, len(u.Permissions.Flags))
		for _, f := range u.Permissions.Flags {
			perms[f.XMLName.Local] = boolFromFlag(f.Value)
		}

		out = append(out, &User{
			ID:    strings.TrimSpace(u.UserID),
			Login: strings.TrimSpace(u.UserLogin),

			FirstName:    u.ContactInfo.FirstName,
			LastName:     u.ContactInfo.LastName,
			Title:        u.ContactInfo.Title,
			Phone:        u.ContactInfo.Phone,
			Fax:          u.ContactInfo.Fax,
			Email:        u.ContactInfo.Email,
			Company:      u.ContactInfo.Company,
			Address1:     u.ContactInfo.Address1,
			Address2:     u.ContactInfo.Address2,
			City:         u.ContactInfo.City,
			Country:      u.ContactInfo.Country,
			State:        u.ContactInfo.State,
			ZipCode:      u.ContactInfo.ZipCode,
			TimeZoneCode: u.ContactInfo.TimeZoneCode,

			Status:         strings.TrimSpace(u.UserStatus),
			CreationDate:   strings.TrimSpace(u.CreationDate),
			LastLoginDate:  strings.TrimSpace(u.LastLoginDate),
			Role:           strings.TrimSpace(u.UserRole),
			BusinessUnit:   strings.TrimSpace(u.BusinessUnit),
			UnitManagerPOC: boolFromFlag(u.UnitManagerPOC),
			ManagerPOC:     boolFromFlag(u.ManagerPOC),

			AssignedAssetGroups: u.AssignedAssetGroups.Titles,
			Permissions:         perms,

			NotifyLatestVuln:   strings.TrimSpace(u.Notifications.LatestVuln),
			NotifyMap:          strings.TrimSpace(u.Notifications.Map),
			NotifyScan:         strings.TrimSpace(u.Notifications.Scan),
			NotifyDailyTickets: boolFromFlag(u.Notifications.DailyTickets),
		})
	}
	return out
}

// UserFilter narrows a user list request. Fields map to the Confirmed
// request parameters on user_list.php.
type UserFilter struct {
	// ExternalIDContains and ExternalIDAssigned are mutually exclusive
	// (Confirmed).
	ExternalIDContains string
	ExternalIDAssigned *bool

	// ShowAccessPermissions includes GUI/API/SAML access permission info in
	// the response when true.
	ShowAccessPermissions bool
}

// ListUsers returns subscription users matching filter.
//
// Session-based authentication is documented as unsupported for this
// endpoint — not a concern for this client, which always authenticates
// per-request. No truncation or pagination behaviour is documented for this
// script (the same as ListReportTemplates, another /msp/*.php endpoint), so
// this is a single request rather than a listAll walk.
//
// What a caller sees is governed by their own role: Managers/Administrators
// see every user; Unit Managers see full detail for their own business
// unit and full-or-partial-or-none detail for others depending on a
// subscription-level visibility setting (see the User type's doc comment).
// This client does not attempt to model that filtering — it returns
// whatever the API returns for the authenticated account.
func (c *Client) ListUsers(ctx context.Context, filter UserFilter) ([]*User, error) {
	if filter.ExternalIDContains != "" && filter.ExternalIDAssigned != nil {
		return nil, fmt.Errorf("qualys vmdr: external_id_contains and external_id_assigned " +
			"are mutually exclusive")
	}

	params := url.Values{}
	setIfNotEmpty(params, "external_id_contains", filter.ExternalIDContains)
	if filter.ExternalIDAssigned != nil {
		if *filter.ExternalIDAssigned {
			params.Set("external_id_assigned", "1")
		} else {
			params.Set("external_id_assigned", "0")
		}
	}
	if filter.ShowAccessPermissions {
		params.Set("show_access_permissions", "1")
	}

	var out userListOutput
	if err := c.do(ctx, request{
		rawEndpoint: c.legacyEndpoint("user_list.php"),
		method:      "GET",
		params:      params,
	}, &out); err != nil {
		return nil, err
	}
	return out.users(), nil
}
