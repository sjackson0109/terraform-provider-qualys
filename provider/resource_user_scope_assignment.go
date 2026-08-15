package provider

import (
	"context"
	"errors"

	"github.com/sjackson0109/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceUserScopeAssignment() *schema.Resource {
	return &schema.Resource{
		Description: "Manages which scope tags and roles are assigned to a subscription " +
			"user, via the Administration RBAC API (`POST /qps/rest/2.0/update/am/user/<id>`, " +
			"`scopeTags`/`roleList` add/remove).\n\n" +
			"This is the closest thing this provider has to Terraform-managed \"which " +
			"assets can this user see\": the modern Administration API scopes a user " +
			"entirely through tags and roles, with no business unit field at all. If your " +
			"asset tags already express per-customer scope (`qualys_asset_tag` + " +
			"`qualys_host_tag_assignment`), assigning the matching tag here as a user's " +
			"scope tag is the mechanism this API exposes for tying the two together — " +
			"though the exact effect a scope tag has on visibility is not independently " +
			"confirmed by an official source; see the resource's evidence-tier note below.\n\n" +
			"There is no confirmed API to *create* a user at all — only get/search/update. " +
			"This resource therefore requires `user_id` to already exist; it manages scope " +
			"on an existing user rather than the user's lifecycle, the same relationship " +
			"`qualys_host_tag_assignment` has to a host asset.\n\n" +
			"Evidence tier: Corroborated (non-official) — reverse-engineered from the " +
			"open-source qualysdk project, the same source and tier already used for " +
			"`qualys_was_auth_record`. Verify against your tenant before relying on this " +
			"in production.",

		CreateContext: resourceUserScopeAssignmentCreate,
		ReadContext:   resourceUserScopeAssignmentRead,
		UpdateContext: resourceUserScopeAssignmentUpdate,
		DeleteContext: resourceUserScopeAssignmentDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"user_id": {
				Description: "Admin user ID (from `data.qualys_am_users`) to assign scope to.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"tag_ids": {
				Description: "Scope tag IDs assigned to this user. Managed authoritatively: " +
					"removing an ID here unassigns that scope tag.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"role_ids": {
				Description: "Role IDs assigned to this user. Managed authoritatively: " +
					"removing an ID here unassigns that role.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceUserScopeAssignmentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	userID := d.Get("user_id").(string)
	tagIDs := stringSet(d, "tag_ids")
	roleIDs := stringSet(d, "role_ids")

	if len(tagIDs) > 0 || len(roleIDs) > 0 {
		if err := c.UpdateUserScope(ctx, userID, qps.UserScopeUpdate{
			AddTagIDs:  tagIDs,
			AddRoleIDs: roleIDs,
		}); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(userID)
	return resourceUserScopeAssignmentRead(ctx, d, meta)
}

func resourceUserScopeAssignmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	u, err := c.GetUser(ctx, d.Id())
	if err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	tagIDs := make([]string, 0, len(u.ScopeTags))
	for _, t := range u.ScopeTags {
		tagIDs = append(tagIDs, t.ID)
	}
	roleIDs := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		roleIDs = append(roleIDs, r.ID)
	}

	return diag.FromErr(setAll(d, map[string]interface{}{
		"user_id":  u.ID,
		"tag_ids":  tagIDs,
		"role_ids": roleIDs,
	}))
}

func resourceUserScopeAssignmentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	update := qps.UserScopeUpdate{}
	if d.HasChange("tag_ids") {
		oldRaw, newRaw := d.GetChange("tag_ids")
		update.AddTagIDs, update.RemoveTagIDs = diffStringSets(
			stringSetFromInterface(oldRaw), stringSetFromInterface(newRaw))
	}
	if d.HasChange("role_ids") {
		oldRaw, newRaw := d.GetChange("role_ids")
		update.AddRoleIDs, update.RemoveRoleIDs = diffStringSets(
			stringSetFromInterface(oldRaw), stringSetFromInterface(newRaw))
	}

	if len(update.AddTagIDs)+len(update.RemoveTagIDs)+len(update.AddRoleIDs)+len(update.RemoveRoleIDs) > 0 {
		if err := c.UpdateUserScope(ctx, d.Id(), update); err != nil {
			return diag.FromErr(err)
		}
	}
	return resourceUserScopeAssignmentRead(ctx, d, meta)
}

func resourceUserScopeAssignmentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	tagIDs := stringSet(d, "tag_ids")
	roleIDs := stringSet(d, "role_ids")
	if len(tagIDs) == 0 && len(roleIDs) == 0 {
		return nil
	}
	if err := c.UpdateUserScope(ctx, d.Id(), qps.UserScopeUpdate{
		RemoveTagIDs:  tagIDs,
		RemoveRoleIDs: roleIDs,
	}); err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
