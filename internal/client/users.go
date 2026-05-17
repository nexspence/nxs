package client

import (
	"encoding/json"
	"fmt"
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

func (c *Client) UserCreate(userID, firstName, lastName, email, password string, roles []string) error {
	body := map[string]any{
		"userId":       userID,
		"firstName":    firstName,
		"lastName":     lastName,
		"emailAddress": email,
		"password":     password,
		"status":       "active",
		"roles":        roles,
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

func (c *Client) RoleAssign(userID string, roles []string) error {
	resp, err := c.r.R().
		SetQueryParam("userId", userID).
		Get("/service/rest/v1/security/users")
	if err != nil {
		return err
	}
	if err := checkErr(resp); err != nil {
		return err
	}
	var users []User
	if err := json.Unmarshal(resp.Body(), &users); err != nil || len(users) == 0 {
		return fmt.Errorf("user %q not found", userID)
	}
	u := users[0]
	existing := map[string]bool{}
	for _, r := range u.Roles {
		existing[r] = true
	}
	for _, r := range roles {
		if !existing[r] {
			u.Roles = append(u.Roles, r)
		}
	}
	resp2, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(u).
		Put("/service/rest/v1/security/users/" + userID)
	if err != nil {
		return err
	}
	return checkErr(resp2)
}
