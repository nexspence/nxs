package client

import (
	"encoding/json"
	"net/url"
	"time"
)

// CleanupScope narrows a policy to one repository and/or a path prefix.
type CleanupScope struct {
	RepositoryName string `json:"repositoryName,omitempty"`
	PathPrefix     string `json:"pathPrefix,omitempty"`
}

// CleanupPolicy mirrors the cleanup policy the server stores and returns.
// The criteria map holds the server's own keys (artifactAgeDays,
// lastDownloadedDays, pathPrefix, nameGlob) — the server reads no others.
type CleanupPolicy struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	Format          string         `json:"format"`
	Criteria        map[string]any `json:"criteria"`
	ScheduleCron    string         `json:"scheduleCron,omitempty"`
	Enabled         bool           `json:"enabled"`
	DryRun          bool           `json:"dryRun"`
	RetainNVersions int            `json:"retainNVersions"`
	Scope           CleanupScope   `json:"scope"`
	LastRunAt       *time.Time     `json:"lastRunAt,omitempty"`
	LastRunFreed    int64          `json:"lastRunFreedBytes,omitempty"`
	LastRunCount    int            `json:"lastRunCount,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// cleanupPolicyBody carries the fields the create and update endpoints write.
// The id, the run statistics and the timestamps belong to the server, so they
// are left out of a request rather than sent back and ignored.
type cleanupPolicyBody struct {
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	Format          string         `json:"format"`
	Criteria        map[string]any `json:"criteria"`
	ScheduleCron    string         `json:"scheduleCron,omitempty"`
	Enabled         bool           `json:"enabled"`
	DryRun          bool           `json:"dryRun"`
	RetainNVersions int            `json:"retainNVersions"`
	Scope           CleanupScope   `json:"scope"`
}

// body renders the policy as the create and update endpoints expect it.
func (p CleanupPolicy) body() cleanupPolicyBody {
	criteria := p.Criteria
	if criteria == nil {
		criteria = map[string]any{}
	}
	return cleanupPolicyBody{
		Name:            p.Name,
		Description:     p.Description,
		Format:          p.Format,
		Criteria:        criteria,
		ScheduleCron:    p.ScheduleCron,
		Enabled:         p.Enabled,
		DryRun:          p.DryRun,
		RetainNVersions: p.RetainNVersions,
		Scope:           p.Scope,
	}
}

// CleanupPreviewAsset is one asset a policy would remove.
type CleanupPreviewAsset struct {
	Path           string     `json:"path"`
	Repository     string     `json:"repository"`
	SizeBytes      int64      `json:"sizeBytes"`
	LastDownloaded *time.Time `json:"lastDownloaded"`
	CreatedAt      time.Time  `json:"createdAt"`
	Reason         string     `json:"reason"`
}

// CleanupPreview is what a policy would remove, without removing it.
type CleanupPreview struct {
	Assets     []CleanupPreviewAsset `json:"assets"`
	TotalCount int                   `json:"totalCount"`
	TotalBytes int64                 `json:"totalBytes"`
}

// CleanupList returns every cleanup policy the server holds.
func (c *Client) CleanupList() ([]CleanupPolicy, error) {
	resp, err := c.r.R().Get("/service/rest/v1/cleanup-policies")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var policies []CleanupPolicy
	return policies, json.Unmarshal(resp.Body(), &policies)
}

// CleanupCreate stores a new cleanup policy.
func (c *Client) CleanupCreate(policy CleanupPolicy) error {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(policy.body()).
		Post("/service/rest/v1/cleanup-policies")
	if err != nil {
		return err
	}
	return checkErr(resp)
}

// CleanupUpdate replaces the configuration of the policy with the given id.
func (c *Client) CleanupUpdate(id string, policy CleanupPolicy) error {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(policy.body()).
		Put("/service/rest/v1/cleanup-policies/" + url.PathEscape(id))
	if err != nil {
		return err
	}
	return checkErr(resp)
}

// CleanupDelete removes the cleanup policy with the given id.
func (c *Client) CleanupDelete(id string) error {
	resp, err := c.r.R().Delete("/service/rest/v1/cleanup-policies/" + url.PathEscape(id))
	if err != nil {
		return err
	}
	return checkErr(resp)
}

// CleanupPreview reports what the policy with the given id would remove,
// without removing anything.
func (c *Client) CleanupPreview(id string) (CleanupPreview, error) {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		Post("/api/v1/cleanup-policies/" + url.PathEscape(id) + "/preview")
	if err != nil {
		return CleanupPreview{}, err
	}
	if err := checkErr(resp); err != nil {
		return CleanupPreview{}, err
	}
	var preview CleanupPreview
	return preview, json.Unmarshal(resp.Body(), &preview)
}
