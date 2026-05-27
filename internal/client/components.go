package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Component struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Format     string `json:"format"`
	Group      string `json:"group"`
	Name       string `json:"name"`
	Version    string `json:"version"`
}

type SearchParams struct {
	Repo   string
	Format string
	Query  string
	Tag    string
}

type searchResponse struct {
	Items             []Component `json:"items"`
	ContinuationToken *string     `json:"continuationToken"`
}

func (c *Client) Search(params SearchParams) ([]Component, error) {
	req := c.r.R()
	if params.Repo != "" {
		req = req.SetQueryParam("repository", params.Repo)
	}
	if params.Format != "" {
		req = req.SetQueryParam("format", params.Format)
	}
	if params.Query != "" {
		req = req.SetQueryParam("q", params.Query)
	}
	if params.Tag != "" {
		req = req.SetQueryParam("tag", params.Tag)
	}

	var all []Component
	for {
		resp, err := req.Get("/service/rest/v1/search")
		if err != nil {
			return nil, err
		}
		if err := checkErr(resp); err != nil {
			return nil, err
		}
		var page searchResponse
		if err := json.Unmarshal(resp.Body(), &page); err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
		if page.ContinuationToken == nil {
			break
		}
		req = req.SetQueryParam("continuationToken", *page.ContinuationToken)
	}
	return all, nil
}

// Push streams localFile to PUT /repository/<repo>/<remotePath>.
func (c *Client) Push(repo, remotePath, localFile string, progressFn func(size int64) io.Writer) error {
	f, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("open %s: %w", localFile, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	var body io.Reader = f
	if progressFn != nil {
		if pw := progressFn(info.Size()); pw != nil {
			body = io.TeeReader(f, pw)
		}
	}

	url := c.base + "/repository/" + repo + "/" + remotePath
	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.r.Header.Get("Authorization"))
	req.ContentLength = info.Size()

	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode >= 300 {
		return fmt.Errorf("upload failed: server returned %d", httpResp.StatusCode)
	}
	return nil
}

// Pull downloads /repository/<repo>/<remotePath> to localPath.
func (c *Client) Pull(repo, remotePath, localPath string, progressFn func(size int64) io.Writer) error {
	resp, err := c.r.R().SetDoNotParseResponse(true).
		Get("/repository/" + repo + "/" + remotePath)
	if err != nil {
		return err
	}
	defer resp.RawBody().Close()
	if err := checkErr(resp); err != nil {
		return err
	}

	size := resp.RawResponse.ContentLength
	var body io.Reader = resp.RawBody()
	if progressFn != nil {
		if pw := progressFn(size); pw != nil {
			body = io.TeeReader(resp.RawBody(), pw)
		}
	}

	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, body)
	return err
}

type Asset struct {
	ID          string `json:"id"`
	Repository  string `json:"repository"`
	Path        string `json:"path"`
	FileSize    int64  `json:"fileSize"`
	DownloadURL string `json:"downloadUrl,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
}

type assetSearchResponse struct {
	Items             []Asset `json:"items"`
	ContinuationToken *string `json:"continuationToken"`
}

// SearchAssets lists assets in repo, optionally filtered to those whose path
// starts with prefix. Pagination follows continuationToken.
func (c *Client) SearchAssets(repo, prefix string) ([]Asset, error) {
	req := c.r.R()
	if repo != "" {
		req = req.SetQueryParam("repository", repo)
	}
	var all []Asset
	for {
		resp, err := req.Get("/service/rest/v1/search/assets")
		if err != nil {
			return nil, err
		}
		if err := checkErr(resp); err != nil {
			return nil, err
		}
		var page assetSearchResponse
		if err := json.Unmarshal(resp.Body(), &page); err != nil {
			return nil, err
		}
		for _, a := range page.Items {
			if prefix == "" || strings.HasPrefix(a.Path, prefix) {
				all = append(all, a)
			}
		}
		if page.ContinuationToken == nil {
			break
		}
		req = req.SetQueryParam("continuationToken", *page.ContinuationToken)
	}
	return all, nil
}
