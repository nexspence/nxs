package client

import (
	"encoding/json"
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
