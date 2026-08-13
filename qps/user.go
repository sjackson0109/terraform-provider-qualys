package qps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// User is a subscription user, as returned by the Administration RBAC API
// (/qps/rest/2.0/.../am/user). This is a DIFFERENT, modern object from
// vmdr.User (the legacy /msp/user_list.php script's XML response) — the two
// come from different API generations with different fields. This one
// carries no BUSINESS_UNIT field at all; it is scoped entirely by
// ScopeTags and Roles. vmdr.User's doc comment records a working
// hypothesis, from one sample response, that asset/tag assignment rather
// than Business Unit membership is what actually gates visibility — this
// object's existence (a modern user API with no Business Unit concept, only
// tags and roles) is consistent with that hypothesis without independently
// proving it: it may simply be that this API doesn't expose Business Unit
// as a field, not that Business Unit is irrelevant to access.
//
// Evidence tier: Corroborated (non-official). Reverse-engineered from the
// open-source qualysdk project (github.com/0x41424142/qualysdk,
// qualysdk/admin/rbac.py and qualysdk/admin/data_classes/User.py) — the
// same evidence source and tier this provider already used successfully
// for qualys_was_auth_record. That project's own docs (docs/admin.md)
// describe ScopeTags/Roles as "a user's roles/scopes", corroborating but
// not independently confirming their effect on visibility.
//
// This source exposes no user *create* endpoint — only get/search/update.
// That is an absence of evidence in one third-party project, not a
// Confirmed absence the way report-schedule CRUD is: no official source
// states user creation via this API is impossible. Because Terraform
// resources need Create, nothing here is wrapped as a full CRUD
// qualys_user resource; see resource_user_scope_assignment.go, which
// manages only the add/remove-able ScopeTags/Roles on an existing user,
// parallel to qualys_host_tag_assignment.
type User struct {
	ID           string
	Username     string
	FirstName    string
	LastName     string
	EmailAddress string
	Title        string
	ScopeTags    []AdminDataPoint
	Roles        []AdminDataPoint
}

// AdminDataPoint identifies a scope tag or role by ID and name, per the
// Administration RBAC API.
type AdminDataPoint struct {
	ID   string
	Name string
}

type adminDataPointWire struct {
	ID   json.Number `json:"id,omitempty"`
	Name string      `json:"name,omitempty"`
}

type userWire struct {
	ID           json.Number `json:"id,omitempty"`
	Username     string      `json:"username,omitempty"`
	FirstName    string      `json:"firstName,omitempty"`
	LastName     string      `json:"lastName,omitempty"`
	EmailAddress string      `json:"emailAddress,omitempty"`
	Title        string      `json:"title,omitempty"`
	// ScopeTags and RoleList are left as raw JSON: the source SDK's own
	// decode logic branches on whether a "list" wrapper is present (present
	// for multiple items; apparently sometimes absent, the element itself
	// bare, for exactly one) — decodeAdminDataPoints reproduces that
	// same defensive handling rather than assuming one shape.
	ScopeTags json.RawMessage `json:"scopeTags,omitempty"`
	RoleList  json.RawMessage `json:"roleList,omitempty"`
}

type userData struct {
	User *userWire `json:"User,omitempty"`
}

// decodeAdminDataPoints handles both shapes the source SDK's own decoder
// distinguishes: {"list": [{"<Key>": {id,name}}, ...]} for multiple items,
// or a bare {id,name} object for exactly one. The wrapper key ("TagData",
// "RoleData") is read from whatever key each list entry actually uses
// rather than hardcoded, since a scope-tags entry and a role entry use
// different keys for the same shape.
func decodeAdminDataPoints(raw json.RawMessage) ([]AdminDataPoint, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var wrapper struct {
		List []map[string]adminDataPointWire `json:"list"`
	}
	if err := json.Unmarshal(raw, &wrapper); err == nil && wrapper.List != nil {
		out := make([]AdminDataPoint, 0, len(wrapper.List))
		for _, item := range wrapper.List {
			for _, v := range item {
				out = append(out, AdminDataPoint{ID: v.ID.String(), Name: v.Name})
			}
		}
		return out, nil
	}

	var single adminDataPointWire
	if err := json.Unmarshal(raw, &single); err == nil && (single.ID != "" || single.Name != "") {
		return []AdminDataPoint{{ID: single.ID.String(), Name: single.Name}}, nil
	}
	return nil, nil
}

func (w *userWire) toUser() (*User, error) {
	tags, err := decodeAdminDataPoints(w.ScopeTags)
	if err != nil {
		return nil, fmt.Errorf("qualys qps: decoding user scopeTags: %w", err)
	}
	roles, err := decodeAdminDataPoints(w.RoleList)
	if err != nil {
		return nil, fmt.Errorf("qualys qps: decoding user roleList: %w", err)
	}
	return &User{
		ID:           w.ID.String(),
		Username:     w.Username,
		FirstName:    w.FirstName,
		LastName:     w.LastName,
		EmailAddress: w.EmailAddress,
		Title:        w.Title,
		ScopeTags:    tags,
		Roles:        roles,
	}, nil
}

func decodeUsers(raw json.RawMessage) ([]*User, error) {
	items := decodeListOrSingle(raw)
	out := make([]*User, 0, len(items))
	for _, item := range items {
		var d userData
		if err := json.Unmarshal(item, &d); err != nil {
			return nil, fmt.Errorf("qualys qps: decoding user data: %w", err)
		}
		if d.User != nil {
			u, err := d.User.toUser()
			if err != nil {
				return nil, err
			}
			out = append(out, u)
		}
	}
	return out, nil
}

// GetUser returns one user by admin user ID, or ErrNotFound.
func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	var resp ServiceResponse
	err := c.call(ctx, http.MethodGet, "/qps/rest/2.0/get/am/user/"+id, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	users, err := decodeUsers(resp.Data)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("qualys qps: user %s: %w", id, ErrNotFound)
	}
	return users[0], nil
}

// SearchUsers returns users matching filters, following pagination.
// Supported criteria fields (per the source SDK): "id" (EQUALS/GREATER/
// LESSER), "username" (EQUALS), "roleName" (EQUALS).
func (c *Client) SearchUsers(ctx context.Context, filters *Filters) ([]*User, error) {
	var all []*User
	err := c.SearchAll(ctx, "/qps/rest/2.0/search/am/user", filters, 100, 0,
		func(raw json.RawMessage) error {
			users, err := decodeUsers(raw)
			if err != nil {
				return err
			}
			all = append(all, users...)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return all, nil
}

// UserScopeUpdate adds and/or removes scope tags and roles on a user. Each
// entry may be identified by ID or by name (the source SDK accepts both).
type UserScopeUpdate struct {
	AddTagIDs, AddTagNames       []string
	RemoveTagIDs, RemoveTagNames []string

	AddRoleIDs, AddRoleNames       []string
	RemoveRoleIDs, RemoveRoleNames []string
}

func (u UserScopeUpdate) empty() bool {
	return len(u.AddTagIDs) == 0 && len(u.AddTagNames) == 0 &&
		len(u.RemoveTagIDs) == 0 && len(u.RemoveTagNames) == 0 &&
		len(u.AddRoleIDs) == 0 && len(u.AddRoleNames) == 0 &&
		len(u.RemoveRoleIDs) == 0 && len(u.RemoveRoleNames) == 0
}

// UpdateUserScope adds and/or removes scope tags and roles on an existing
// user. At least one add/remove list must be non-empty (Confirmed by the
// source SDK's own validation, reproduced here).
func (c *Client) UpdateUserScope(ctx context.Context, userID string, in UserScopeUpdate) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("qualys qps: user id is required")
	}
	if in.empty() {
		return fmt.Errorf("qualys qps: at least one tag or role to add or remove must be provided")
	}

	userPayload := map[string]interface{}{}
	if scopeTags := adminScopePayload(in.AddTagIDs, in.AddTagNames, in.RemoveTagIDs, in.RemoveTagNames, "TagData"); scopeTags != nil {
		userPayload["scopeTags"] = scopeTags
	}
	if roleList := adminScopePayload(in.AddRoleIDs, in.AddRoleNames, in.RemoveRoleIDs, in.RemoveRoleNames, "RoleData"); roleList != nil {
		userPayload["roleList"] = roleList
	}

	return c.call(ctx, http.MethodPost, "/qps/rest/2.0/update/am/user/"+userID,
		&ServiceRequest{Data: map[string]interface{}{"User": userPayload}}, nil, false)
}

func adminScopePayload(addIDs, addNames, removeIDs, removeNames []string, dataKey string) map[string]interface{} {
	add := adminDataPointRefs(addIDs, addNames)
	remove := adminDataPointRefs(removeIDs, removeNames)
	if len(add) == 0 && len(remove) == 0 {
		return nil
	}
	out := map[string]interface{}{}
	if len(add) > 0 {
		out["add"] = map[string]interface{}{dataKey: add}
	}
	if len(remove) > 0 {
		out["remove"] = map[string]interface{}{dataKey: remove}
	}
	return out
}

func adminDataPointRefs(ids, names []string) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(ids)+len(names))
	for _, id := range ids {
		out = append(out, map[string]interface{}{"id": json.Number(id)})
	}
	for _, name := range names {
		out = append(out, map[string]interface{}{"name": name})
	}
	return out
}
