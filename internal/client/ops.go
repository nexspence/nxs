package client

import (
	"encoding/json"
	"fmt"
)

type SystemInfo struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	AppName string `json:"appName"`
}

func (c *Client) SystemInfo() (*SystemInfo, error) {
	resp, err := c.r.R().Get("/health")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var info SystemInfo
	return &info, json.Unmarshal(resp.Body(), &info)
}

// CleanupRun triggers immediate execution of a cleanup policy by name.
// Verify the exact endpoint path in nexspence-core internal/api/router.go before use.
func (c *Client) CleanupRun(policyName string) error {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		Post(fmt.Sprintf("/api/v1/cleanup-policies/%s/run", policyName))
	if err != nil {
		return err
	}
	return checkErr(resp)
}

type MigrateRequest struct {
	SourceURL string `json:"sourceUrl"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Repos     bool   `json:"repos"`
	Users     bool   `json:"users"`
	Blobs     bool   `json:"blobs"`
}

type MigrateJob struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (c *Client) MigrateStart(req MigrateRequest) (*MigrateJob, error) {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		Post("/api/v1/migration/jobs")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var job MigrateJob
	return &job, json.Unmarshal(resp.Body(), &job)
}
