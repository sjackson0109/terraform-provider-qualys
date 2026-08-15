package provider

import (
	"context"
	"sort"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAMUsers() *schema.Resource {
	return &schema.Resource{
		Description: "Look up subscription users via the Administration RBAC API " +
			"(`/qps/rest/2.0/.../am/user`). This is a **different, modern API from " +
			"`data.qualys_users`**, which reads the legacy `/msp/user_list.php` script — " +
			"the two return different fields for what may or may not be exactly the same " +
			"underlying user, and this project has not confirmed how (or whether) the two " +
			"IDs correspond. This one carries no business unit field at all; scope is " +
			"expressed entirely through `scope_tags` and `roles`.\n\n" +
			"Evidence tier: Corroborated (non-official) — reverse-engineered from the " +
			"open-source qualysdk project, the same source and tier already used for " +
			"`qualys_was_auth_record`. Verify against your tenant before depending on this.",

		ReadContext: dataSourceAMUsersRead,

		Schema: map[string]*schema.Schema{
			"id_filter": {
				Description: "Return only the user with this admin user ID.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"username": {
				Description: "Return only the user with this username.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"role_name": {
				Description: "Return only users with this role name.",
				Type:        schema.TypeString,
				Optional:    true,
			},

			"users": {
				Description: "Matching users, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":            {Type: schema.TypeString, Computed: true},
						"username":      {Type: schema.TypeString, Computed: true},
						"first_name":    {Type: schema.TypeString, Computed: true},
						"last_name":     {Type: schema.TypeString, Computed: true},
						"email_address": {Type: schema.TypeString, Computed: true},
						"title":         {Type: schema.TypeString, Computed: true},
						"scope_tags": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id":   {Type: schema.TypeString, Computed: true},
									"name": {Type: schema.TypeString, Computed: true},
								},
							},
						},
						"roles": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id":   {Type: schema.TypeString, Computed: true},
									"name": {Type: schema.TypeString, Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceAMUsersRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	var criteria []qps.Criterion
	if v := d.Get("id_filter").(string); v != "" {
		criteria = append(criteria, qps.Criterion{Field: "id", Operator: "EQUALS", Value: v})
	}
	if v := d.Get("username").(string); v != "" {
		criteria = append(criteria, qps.Criterion{Field: "username", Operator: "EQUALS", Value: v})
	}
	if v := d.Get("role_name").(string); v != "" {
		criteria = append(criteria, qps.Criterion{Field: "roleName", Operator: "EQUALS", Value: v})
	}
	if len(criteria) == 0 {
		// The source SDK documents "id=1, operator=GREATER" as the way to
		// list every user, and enforces client-side that at least one
		// criterion is always sent — reproduced here as the default when
		// the caller supplies no filter.
		criteria = append(criteria, qps.Criterion{Field: "id", Operator: "GREATER", Value: "1"})
	}

	users, err := c.SearchUsers(ctx, &qps.Filters{Criteria: criteria})
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })

	out := make([]interface{}, 0, len(users))
	ids := make([]string, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
		out = append(out, map[string]interface{}{
			"id":            u.ID,
			"username":      u.Username,
			"first_name":    u.FirstName,
			"last_name":     u.LastName,
			"email_address": u.EmailAddress,
			"title":         u.Title,
			"scope_tags":    adminDataPointsToList(u.ScopeTags),
			"roles":         adminDataPointsToList(u.Roles),
		})
	}

	if err := d.Set("users", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}

func adminDataPointsToList(points []qps.AdminDataPoint) []interface{} {
	out := make([]interface{}, 0, len(points))
	for _, p := range points {
		out = append(out, map[string]interface{}{"id": p.ID, "name": p.Name})
	}
	return out
}
