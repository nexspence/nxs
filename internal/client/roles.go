package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// Role is a role as the server reports it.
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RoleList returns every role defined on the server.
func (c *Client) RoleList() ([]Role, error) {
	resp, err := c.r.R().Get("/service/rest/v1/security/roles")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var roles []Role
	return roles, json.Unmarshal(resp.Body(), &roles)
}

// resolveRoleIDs maps role names to the ids the assignment endpoint expects.
// A name the server does not know is an error rather than a silent skip: the
// caller asked for a role and would otherwise end up with fewer than it
// believes it has.
func (c *Client) resolveRoleIDs(names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	roles, err := c.RoleList()
	if err != nil {
		return nil, err
	}
	byName := make(map[string]string, len(roles))
	for _, r := range roles {
		byName[r.Name] = r.ID
	}
	ids := make([]string, 0, len(names))
	for _, n := range names {
		id, ok := byName[n]
		if !ok {
			return nil, fmt.Errorf("role %q not found", n)
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// UserGet returns one user. The list endpoint ignores a userId filter, so the
// user is fetched by its own path instead of being picked out of a list.
func (c *Client) UserGet(userID string) (*User, error) {
	resp, err := c.r.R().Get("/service/rest/v1/security/users/" + url.PathEscape(userID))
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var u User
	if err := json.Unmarshal(resp.Body(), &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// RoleAssign adds the named roles to a user, keeping the ones already there.
//
// The assignment endpoint takes role ids and replaces the whole set, so the
// current roles are read first and sent back along with the new ones.
func (c *Client) RoleAssign(userID string, roles []string) error {
	u, err := c.UserGet(userID)
	if err != nil {
		return err
	}
	existing := make([]string, 0, len(u.Roles)+len(roles))
	existing = append(existing, u.Roles...)
	for _, r := range roles {
		if !slices.Contains(existing, r) {
			existing = append(existing, r)
		}
	}
	ids, err := c.resolveRoleIDs(existing)
	if err != nil {
		return err
	}
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string][]string{"roleIds": ids}).
		Put("/service/rest/v1/security/users/" + url.PathEscape(userID) + "/roles")
	if err != nil {
		return err
	}
	return checkErr(resp)
}

// RoleRevoke removes the named roles from a user, keeping the others.
func (c *Client) RoleRevoke(userID string, roles []string) error {
	u, err := c.UserGet(userID)
	if err != nil {
		return err
	}
	kept := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		if !slices.Contains(roles, r) {
			kept = append(kept, r)
		}
	}
	if len(kept) == len(u.Roles) {
		return fmt.Errorf("user %q does not have %s", userID, strings.Join(roles, ", "))
	}
	ids, err := c.resolveRoleIDs(kept)
	if err != nil {
		return err
	}
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string][]string{"roleIds": ids}).
		Put("/service/rest/v1/security/users/" + url.PathEscape(userID) + "/roles")
	if err != nil {
		return err
	}
	return checkErr(resp)
}
