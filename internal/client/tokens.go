package client

import (
	"encoding/json"
	"time"
)

type Token struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes,omitempty"`
	LastUsed  *time.Time `json:"lastUsed,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	// Token holds the plaintext value, populated only by TokenCreate.
	Token string `json:"token,omitempty"`
}

func (c *Client) TokenList() ([]Token, error) {
	resp, err := c.r.R().Get("/api/v1/tokens")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var toks []Token
	return toks, json.Unmarshal(resp.Body(), &toks)
}

func (c *Client) TokenCreate(name string, scopes []string, expiresInDays *int) (*Token, error) {
	body := map[string]any{"name": name}
	if len(scopes) > 0 {
		body["scopes"] = scopes
	}
	if expiresInDays != nil {
		body["expiresInDays"] = *expiresInDays
	}
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post("/api/v1/tokens")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var tok Token
	return &tok, json.Unmarshal(resp.Body(), &tok)
}

func (c *Client) TokenDelete(id string) error {
	resp, err := c.r.R().Delete("/api/v1/tokens/" + id)
	if err != nil {
		return err
	}
	return checkErr(resp)
}
