package provider

import (
	"context"
	"sort"

	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceUsers() *schema.Resource {
	return &schema.Resource{
		Description: "Look up subscription users (`/msp/user_list.php`, Confirmed by a " +
			"user-supplied primary-source excerpt: DTD name, full parameter table and a " +
			"complete sample response).\n\n" +
			"What a caller sees is governed by their own role: Managers/Administrators see " +
			"every user; Unit Managers see full detail for their own business unit and " +
			"full, partial or no detail for others depending on a subscription-level " +
			"visibility setting. This data source returns whatever the API returns for the " +
			"configured account — it does not attempt to model that filtering itself.\n\n" +
			"`assigned_asset_groups` looks like the real access-scoping mechanism (a sample " +
			"Manager-role user carries no `ASSIGNED_ASSET_GROUPS` element at all, while a " +
			"sample Scanner-role user does), not `business_unit` — but this is an inference " +
			"from one document, not independently confirmed by a Business Unit API " +
			"reference. This is read-only: user creation/edit and Business Unit management " +
			"use a different, not-yet-researched part of the API and are not modelled here.",

		ReadContext: dataSourceUsersRead,

		Schema: map[string]*schema.Schema{
			"external_id_contains": {
				Description: "Return only users whose external ID contains this string. " +
					"Mutually exclusive with `external_id_assigned`.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"external_id_assigned": {
				Description: "Return only users with (`true`) or without (`false`) an " +
					"external ID assigned. Mutually exclusive with `external_id_contains`.",
				Type:     schema.TypeBool,
				Optional: true,
			},
			"show_access_permissions": {
				Description: "Include GUI/API/SAML access permission info in the result.",
				Type:        schema.TypeBool,
				Optional:    true,
			},

			"users": {
				Description: "Matching users, ordered by user ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":               {Type: schema.TypeString, Computed: true},
						"login":            {Type: schema.TypeString, Computed: true},
						"first_name":       {Type: schema.TypeString, Computed: true},
						"last_name":        {Type: schema.TypeString, Computed: true},
						"title":            {Type: schema.TypeString, Computed: true},
						"phone":            {Type: schema.TypeString, Computed: true},
						"fax":              {Type: schema.TypeString, Computed: true},
						"email":            {Type: schema.TypeString, Computed: true},
						"company":          {Type: schema.TypeString, Computed: true},
						"address1":         {Type: schema.TypeString, Computed: true},
						"address2":         {Type: schema.TypeString, Computed: true},
						"city":             {Type: schema.TypeString, Computed: true},
						"country":          {Type: schema.TypeString, Computed: true},
						"state":            {Type: schema.TypeString, Computed: true},
						"zip_code":         {Type: schema.TypeString, Computed: true},
						"time_zone_code":   {Type: schema.TypeString, Computed: true},
						"status":           {Type: schema.TypeString, Computed: true},
						"creation_date":    {Type: schema.TypeString, Computed: true},
						"last_login_date":  {Type: schema.TypeString, Computed: true},
						"role":             {Type: schema.TypeString, Computed: true},
						"business_unit":    {Type: schema.TypeString, Computed: true},
						"unit_manager_poc": {Type: schema.TypeBool, Computed: true},
						"manager_poc":      {Type: schema.TypeBool, Computed: true},
						"assigned_asset_groups": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"permissions": {
							Description: "Whatever permission flags the API returned for " +
								"this user, keyed by their XML element name. Not a fixed " +
								"schema: the source documents this as an open-ended " +
								"\"extended permissions\" set.",
							Type:     schema.TypeMap,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeBool},
						},
						"notify_latest_vuln":   {Type: schema.TypeString, Computed: true},
						"notify_map":           {Type: schema.TypeString, Computed: true},
						"notify_scan":          {Type: schema.TypeString, Computed: true},
						"notify_daily_tickets": {Type: schema.TypeBool, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceUsersRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	filter := vmdr.UserFilter{
		ExternalIDContains:    d.Get("external_id_contains").(string),
		ShowAccessPermissions: d.Get("show_access_permissions").(bool),
	}
	if v, ok := d.GetOkExists("external_id_assigned"); ok {
		b := v.(bool)
		filter.ExternalIDAssigned = &b
	}

	users, err := c.ListUsers(ctx, filter)
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })

	out := make([]interface{}, 0, len(users))
	ids := make([]string, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
		perms := make(map[string]interface{}, len(u.Permissions))
		for k, v := range u.Permissions {
			perms[k] = v
		}
		out = append(out, map[string]interface{}{
			"id":                    u.ID,
			"login":                 u.Login,
			"first_name":            u.FirstName,
			"last_name":             u.LastName,
			"title":                 u.Title,
			"phone":                 u.Phone,
			"fax":                   u.Fax,
			"email":                 u.Email,
			"company":               u.Company,
			"address1":              u.Address1,
			"address2":              u.Address2,
			"city":                  u.City,
			"country":               u.Country,
			"state":                 u.State,
			"zip_code":              u.ZipCode,
			"time_zone_code":        u.TimeZoneCode,
			"status":                u.Status,
			"creation_date":         u.CreationDate,
			"last_login_date":       u.LastLoginDate,
			"role":                  u.Role,
			"business_unit":         u.BusinessUnit,
			"unit_manager_poc":      u.UnitManagerPOC,
			"manager_poc":           u.ManagerPOC,
			"assigned_asset_groups": u.AssignedAssetGroups,
			"permissions":           perms,
			"notify_latest_vuln":    u.NotifyLatestVuln,
			"notify_map":            u.NotifyMap,
			"notify_scan":           u.NotifyScan,
			"notify_daily_tickets":  u.NotifyDailyTickets,
		})
	}

	if err := d.Set("users", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
