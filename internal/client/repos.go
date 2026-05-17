package client

import "encoding/json"

type Repository struct {
	Name   string `json:"name"`
	Format string `json:"format"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Online bool   `json:"online"`
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
