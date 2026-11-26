package client

import (
	"encoding/json"
	"net/url"
)

type User struct {
	UserID       string   `json:"userId"`
	FirstName    string   `json:"firstName"`
	LastName     string   `json:"lastName"`
	EmailAddress string   `json:"emailAddress"`
	Roles        []string `json:"roles"`
	Status       string   `json:"status"`
}

func (c *Client) UserList() ([]User, error) {
	resp, err := c.r.R().Get("/service/rest/v1/security/users")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var users []User
	return users, json.Unmarshal(resp.Body(), &users)
}

// UserCreate creates a local user. The roles are given by name and resolved to
// ids here: the server stores role assignments by id, and a name handed to it
// directly fails the assignment.
func (c *Client) UserCreate(userID, firstName, lastName, email, password string, roles []string) error {
	roleIDs, err := c.resolveRoleIDs(roles)
	if err != nil {
		return err
	}
	body := map[string]any{
		"userId":       userID,
		"firstName":    firstName,
		"lastName":     lastName,
		"emailAddress": email,
		"password":     password,
		"status":       "active",
		"roles":        roleIDs,
	}
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post("/service/rest/v1/security/users")
	if err != nil {
		return err
	}
	return checkErr(resp)
}

// UserUpdate changes a user's profile fields and role assignment.
//
// Only the fields that carry a value are sent: the server keeps every field
// the payload leaves out, so a partial payload changes exactly what it names.
//
// The roles do not travel in that payload. The update handler binds them, but
// the service behind it never reads them, so the assignment goes through the
// endpoint that owns it — the same one "role assign" uses, which adds the
// named roles and keeps the ones the user already has.
func (c *Client) UserUpdate(userID, firstName, lastName, email string, roles []string) error {
	body := map[string]any{}
	if firstName != "" {
		body["firstName"] = firstName
	}
	if lastName != "" {
		body["lastName"] = lastName
	}
	if email != "" {
		body["emailAddress"] = email
	}
	if len(body) > 0 {
		resp, err := c.r.R().
			SetHeader("Content-Type", "application/json").
			SetBody(body).
			Put("/service/rest/v1/security/users/" + url.PathEscape(userID))
		if err != nil {
			return err
		}
		if err := checkErr(resp); err != nil {
			return err
		}
	}
	if len(roles) == 0 {
		return nil
	}
	return c.RoleAssign(userID, roles)
}

// UserDelete removes a user.
func (c *Client) UserDelete(userID string) error {
	resp, err := c.r.R().Delete("/service/rest/v1/security/users/" + url.PathEscape(userID))
	if err != nil {
		return err
	}
	return checkErr(resp)
}

// UserChangePassword sets a new password for a user.
//
// The password is sent in the request body and nowhere else: it is not part
// of the URL, and neither this method nor the commands above it put it into
// an error message or into printed output.
func (c *Client) UserChangePassword(userID, password string) error {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{"newPassword": password}).
		Put("/service/rest/v1/security/users/" + url.PathEscape(userID) + "/change-password")
	if err != nil {
		return err
	}
	return checkErr(resp)
}
