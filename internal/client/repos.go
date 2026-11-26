package client

import (
	"encoding/json"
	"net/url"
)

type Repository struct {
	Name         string         `json:"name"`
	Format       string         `json:"format"`
	Type         string         `json:"type"`
	URL          string         `json:"url"`
	Online       bool           `json:"online"`
	FormatConfig map[string]any `json:"formatConfig,omitempty"`
}

func (c *Client) RepoList(format, repoType string) ([]Repository, error) {
	req := c.r.R()
	if format != "" {
		req = req.SetQueryParam("format", format)
	}
	if repoType != "" {
		req = req.SetQueryParam("type", repoType)
	}
	resp, err := req.Get("/service/rest/v1/repositories")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var repos []Repository
	return repos, json.Unmarshal(resp.Body(), &repos)
}

func (c *Client) RepoCreate(name, format, repoType, blobStore string) error {
	if blobStore == "" {
		blobStore = "default"
	}
	body := map[string]any{
		"name":   name,
		"online": true,
		"storage": map[string]string{
			"blobStoreName": blobStore,
		},
	}
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post("/service/rest/v1/repositories/" + format + "/" + repoType)
	if err != nil {
		return err
	}
	return checkErr(resp)
}

func (c *Client) RepoDelete(name string) error {
	resp, err := c.r.R().Delete("/service/rest/v1/repositories/" + name)
	if err != nil {
		return err
	}
	return checkErr(resp)
}

func (c *Client) RepoInfo(name string) (*Repository, error) {
	resp, err := c.r.R().Get("/service/rest/v1/repositories/" + name)
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var repo Repository
	return &repo, json.Unmarshal(resp.Body(), &repo)
}

// RepoUpdateOptions carries the repository settings a caller wants to change.
// A nil field is a setting the caller did not mention, and such a setting is
// not sent at all: the server reads an absent field as "unchanged", so sending
// one would change a setting nobody asked to change.
type RepoUpdateOptions struct {
	Online              *bool
	Description         *string
	WritePolicy         *string
	AllowRedeployLatest *bool
	PullBlockSeverity   *string
	MemberNames         []string
}

// RepoUpdate changes the settings of a repository.
//
// The endpoint is addressed by format and type, so the repository is looked up
// first and both are taken from the answer.
//
// The settings that live in a repository's formatConfig (write policy, pull
// gate, group members) must be sent inside formatConfig: the server refuses
// one of them at the top level, because a setting dropped by the decoder is a
// setting the caller believes is in force and is not.
func (c *Client) RepoUpdate(name string, opts RepoUpdateOptions) error {
	repo, err := c.RepoInfo(name)
	if err != nil {
		return err
	}

	body := map[string]any{}
	if opts.Online != nil {
		body["online"] = *opts.Online
	}
	if opts.Description != nil {
		body["description"] = *opts.Description
	}

	// The server replaces formatConfig as a whole, so the settings already
	// stored are carried over and only the requested ones are replaced. A
	// write policy set on a group repository must not cost the repository its
	// member list.
	formatConfig := map[string]any{}
	for k, v := range repo.FormatConfig {
		formatConfig[k] = v
	}
	touched := false
	if opts.WritePolicy != nil {
		formatConfig["write_policy"] = *opts.WritePolicy
		touched = true
	}
	if opts.AllowRedeployLatest != nil {
		formatConfig["allow_redeploy_latest"] = *opts.AllowRedeployLatest
		touched = true
	}
	if opts.PullBlockSeverity != nil {
		formatConfig["pull_block_severity"] = *opts.PullBlockSeverity
		touched = true
	}
	if opts.MemberNames != nil {
		formatConfig["member_names"] = opts.MemberNames
		touched = true
	}
	if touched {
		body["formatConfig"] = formatConfig
	}

	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Put("/service/rest/v1/repositories/" + url.PathEscape(repo.Format) + "/" + url.PathEscape(repo.Type) + "/" + url.PathEscape(name))
	if err != nil {
		return err
	}
	return checkErr(resp)
}
