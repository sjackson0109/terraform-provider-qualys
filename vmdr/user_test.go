package vmdr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

// sampleUserListXML is adapted directly from the official "List Users" API
// page's sample response (both users, trimmed of surrounding whitespace
// noise but otherwise verbatim field-for-field).
const sampleUserListXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE USER_LIST_OUTPUT SYSTEM "user_list_output.dtd">
<USER_LIST_OUTPUT>
	<USER_LIST>
		<USER>
			<USER_LOGIN>acme_ab1</USER_LOGIN>
			<USER_ID>63</USER_ID>
			<CONTACT_INFO>
				<FIRSTNAME><![CDATA[Alex]]></FIRSTNAME>
				<LASTNAME><![CDATA[Kim]]></LASTNAME>
				<TITLE><![CDATA[Manager, Security]]></TITLE>
				<PHONE><![CDATA[650 801 6100]]></PHONE>
				<FAX><![CDATA[650 801 6101]]></FAX>
				<EMAIL><![CDATA[test@abc.com]]></EMAIL>
				<COMPANY><![CDATA[Acme, Inc.]]></COMPANY>
				<ADDRESS1><![CDATA[100 Summer Street]]></ADDRESS1>
				<ADDRESS2><![CDATA[]]></ADDRESS2>
				<CITY><![CDATA[San Francisco]]></CITY>
				<COUNTRY>United States of America</COUNTRY>
				<STATE>California</STATE>
				<ZIP_CODE><![CDATA[94111]]></ZIP_CODE>
				<TIME_ZONE_CODE><![CDATA[Auto]]></TIME_ZONE_CODE>
			</CONTACT_INFO>
			<USER_STATUS>Active</USER_STATUS>
			<CREATION_DATE>2017-07-26T19:43:01Z</CREATION_DATE>
			<LAST_LOGIN_DATE>2018-04-26T22:41:56Z</LAST_LOGIN_DATE>
			<USER_ROLE>Manager</USER_ROLE>
			<BUSINESS_UNIT><![CDATA[Unassigned]]></BUSINESS_UNIT>
			<UNIT_MANAGER_POC>0</UNIT_MANAGER_POC>
			<MANAGER_POC>1</MANAGER_POC>
			<UI_INTERFACE_STYLE>standard_blue</UI_INTERFACE_STYLE>
			<PERMISSIONS>
				<CREATE_OPTION_PROFILES>1</CREATE_OPTION_PROFILES>
				<PURGE_INFO>1</PURGE_INFO>
				<ADD_ASSETS>1</ADD_ASSETS>
				<EDIT_REMEDIATION_POLICY>1</EDIT_REMEDIATION_POLICY>
				<EDIT_AUTH_RECORDS>1</EDIT_AUTH_RECORDS>
			</PERMISSIONS>
			<NOTIFICATIONS>
				<LATEST_VULN>weekly</LATEST_VULN>
				<MAP>ags</MAP>
				<SCAN>ags</SCAN>
				<DAILY_TICKETS>0</DAILY_TICKETS>
			</NOTIFICATIONS>
		</USER>
		<USER>
			<USER_LOGIN>test_user</USER_LOGIN>
			<USER_ID>123456</USER_ID>
			<CONTACT_INFO>
				<FIRSTNAME><![CDATA[Geoff]]></FIRSTNAME>
				<LASTNAME><![CDATA[Holden]]></LASTNAME>
				<TITLE><![CDATA[Security Scanner]]></TITLE>
				<PHONE><![CDATA[650 801 6100]]></PHONE>
				<FAX><![CDATA[650 801 6101]]></FAX>
				<EMAIL><![CDATA[gholden@acme.com]]></EMAIL>
				<COMPANY><![CDATA[Acme, Inc.]]></COMPANY>
				<ADDRESS1><![CDATA[100 Summer Street]]></ADDRESS1>
				<ADDRESS2><![CDATA[]]></ADDRESS2>
				<CITY><![CDATA[San Francisco]]></CITY>
				<COUNTRY>United States of America</COUNTRY>
				<STATE>California</STATE>
				<ZIP_CODE><![CDATA[94111]]></ZIP_CODE>
				<TIME_ZONE_CODE><![CDATA[US-CA]]></TIME_ZONE_CODE>
			</CONTACT_INFO>
			<ASSIGNED_ASSET_GROUPS>
				<ASSET_GROUP_TITLE><![CDATA[AG 24]]></ASSET_GROUP_TITLE>
			</ASSIGNED_ASSET_GROUPS>
			<USER_STATUS>Pending Activation</USER_STATUS>
			<CREATION_DATE>2018-04-06T21:02:26Z</CREATION_DATE>
			<LAST_LOGIN_DATE>N/A</LAST_LOGIN_DATE>
			<USER_ROLE>Scanner</USER_ROLE>
			<BUSINESS_UNIT><![CDATA[Unassigned]]></BUSINESS_UNIT>
			<UNIT_MANAGER_POC>0</UNIT_MANAGER_POC>
			<MANAGER_POC>0</MANAGER_POC>
			<UI_INTERFACE_STYLE>standard_blue</UI_INTERFACE_STYLE>
			<PERMISSIONS>
				<CREATE_OPTION_PROFILES>1</CREATE_OPTION_PROFILES>
				<PURGE_INFO>0</PURGE_INFO>
				<ADD_ASSETS>0</ADD_ASSETS>
				<EDIT_REMEDIATION_POLICY>0</EDIT_REMEDIATION_POLICY>
				<EDIT_AUTH_RECORDS>0</EDIT_AUTH_RECORDS>
			</PERMISSIONS>
			<NOTIFICATIONS>
				<LATEST_VULN>weekly</LATEST_VULN>
				<MAP>ags</MAP>
				<SCAN>ags</SCAN>
				<DAILY_TICKETS>0</DAILY_TICKETS>
			</NOTIFICATIONS>
		</USER>
	</USER_LIST>
</USER_LIST_OUTPUT>`

func TestListUsersParsesSampleResponse(t *testing.T) {
	var gotPath, gotMethod string
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		fmt.Fprint(w, sampleUserListXML)
	}))
	defer srv.Close()

	users, err := c.ListUsers(context.Background(), UserFilter{})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/msp/user_list.php" {
		t.Errorf("method/path = %s %s", gotMethod, gotPath)
	}
	if len(users) != 2 {
		t.Fatalf("got %d users, want 2", len(users))
	}

	manager := users[0]
	if manager.Login != "acme_ab1" || manager.ID != "63" || manager.Role != "Manager" {
		t.Errorf("manager = %+v", manager)
	}
	if manager.FirstName != "Alex" || manager.LastName != "Kim" || manager.Email != "test@abc.com" {
		t.Errorf("manager contact info = %+v", manager)
	}
	if !manager.ManagerPOC || manager.UnitManagerPOC {
		t.Errorf("manager POC flags = ManagerPOC=%v UnitManagerPOC=%v", manager.ManagerPOC, manager.UnitManagerPOC)
	}
	if manager.BusinessUnit != "Unassigned" {
		t.Errorf("manager business unit = %q", manager.BusinessUnit)
	}
	// Confirmed by the source sample: a Manager carries no
	// ASSIGNED_ASSET_GROUPS element at all.
	if len(manager.AssignedAssetGroups) != 0 {
		t.Errorf("manager assigned asset groups = %v, want none", manager.AssignedAssetGroups)
	}
	if !manager.Permissions["PURGE_INFO"] || !manager.Permissions["ADD_ASSETS"] {
		t.Errorf("manager permissions = %v", manager.Permissions)
	}
	if manager.NotifyLatestVuln != "weekly" || manager.NotifyMap != "ags" || manager.NotifyDailyTickets {
		t.Errorf("manager notifications = latest_vuln=%q map=%q daily_tickets=%v",
			manager.NotifyLatestVuln, manager.NotifyMap, manager.NotifyDailyTickets)
	}

	scanner := users[1]
	if scanner.Login != "test_user" || scanner.Role != "Scanner" {
		t.Errorf("scanner = %+v", scanner)
	}
	if scanner.Status != "Pending Activation" || scanner.LastLoginDate != "N/A" {
		t.Errorf("scanner status/last login = %q / %q", scanner.Status, scanner.LastLoginDate)
	}
	if len(scanner.AssignedAssetGroups) != 1 || scanner.AssignedAssetGroups[0] != "AG 24" {
		t.Errorf("scanner assigned asset groups = %v, want [AG 24]", scanner.AssignedAssetGroups)
	}
	if scanner.Permissions["PURGE_INFO"] || scanner.Permissions["ADD_ASSETS"] {
		t.Errorf("scanner permissions = %v, want all false", scanner.Permissions)
	}
}

func TestListUsersSendsFilterParams(t *testing.T) {
	var gotForm url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.Form
		fmt.Fprint(w, `<USER_LIST_OUTPUT><USER_LIST></USER_LIST></USER_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	assigned := true
	_, err := c.ListUsers(context.Background(), UserFilter{
		ExternalIDContains:    "",
		ShowAccessPermissions: true,
	})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if gotForm.Get("show_access_permissions") != "1" {
		t.Errorf("form = %v", gotForm)
	}

	_, err = c.ListUsers(context.Background(), UserFilter{ExternalIDAssigned: &assigned})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if gotForm.Get("external_id_assigned") != "1" {
		t.Errorf("form = %v", gotForm)
	}
}

func TestListUsersRejectsConflictingExternalIDFilters(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent with conflicting filters")
	}))
	defer srv.Close()

	assigned := true
	_, err := c.ListUsers(context.Background(), UserFilter{
		ExternalIDContains: "foo",
		ExternalIDAssigned: &assigned,
	})
	if err == nil {
		t.Fatal("expected an error for mutually exclusive external_id filters")
	}
}
