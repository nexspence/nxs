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

// CleanupRun triggers immediate execution of a cleanup policy by id.
// The endpoint addresses a policy by its id, so a name has to be resolved to
// one before it is called.
func (c *Client) CleanupRun(policyID string) error {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		Post(fmt.Sprintf("/service/rest/v1/cleanup-policies/%s/run", policyID))
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
