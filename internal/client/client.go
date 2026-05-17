package client

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	r    *resty.Client
	base string
}

func New(url, token string) *Client {
	r := resty.New().
		SetBaseURL(url).
		SetHeader("Authorization", "Bearer "+token).
		SetHeader("Accept", "application/json")
	return &Client{r: r, base: url}
}

func checkErr(resp *resty.Response) error {
	if resp == nil {
		return fmt.Errorf("no response from server")
	}
	switch resp.StatusCode() {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent, http.StatusAccepted:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("invalid token or session expired")
	case http.StatusForbidden:
		return fmt.Errorf("insufficient permissions")
	case http.StatusNotFound:
		return fmt.Errorf("resource not found")
	case http.StatusConflict:
		return fmt.Errorf("conflict: resource already exists")
	default:
		if body := resp.String(); body != "" {
			return fmt.Errorf("server error %d: %s", resp.StatusCode(), body)
		}
		return fmt.Errorf("server returned %d", resp.StatusCode())
	}
}

func (c *Client) HealthCheck() error {
	resp, err := c.r.R().Get("/health")
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	return checkErr(resp)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

// Login authenticates against url and returns a JWT token.
func Login(url, username, password string) (string, error) {
	r := resty.New().SetBaseURL(url)
	var result loginResponse
	resp, err := r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(loginRequest{Username: username, Password: password}).
		SetResult(&result).
		Post("/api/v1/login")
	if err != nil {
		return "", fmt.Errorf("connection failed: %w", err)
	}
	if resp.StatusCode() == http.StatusUnauthorized {
		return "", fmt.Errorf("invalid username or password")
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("server returned %d", resp.StatusCode())
	}
	return result.Token, nil
}
