package client

import (
	"encoding/json"
	"fmt"
)

// GCResult reports what a blob store compaction found and removed.
type GCResult struct {
	Store        string   `json:"store"`
	ScannedBlobs int      `json:"scannedBlobs"`
	Orphans      int      `json:"orphans"`
	FreedBytes   int64    `json:"freedBytes"`
	DryRun       bool     `json:"dryRun"`
	Errors       []string `json:"errors,omitempty"`
}

// BlobStoreCompact runs garbage collection on a blob store, removing blobs no
// longer referenced by any asset. When dryRun is true, orphans are reported but
// not deleted. minAge (e.g. "24h") overrides the server's grace period; an empty
// string uses the server default.
func (c *Client) BlobStoreCompact(name string, dryRun bool, minAge string) (*GCResult, error) {
	req := c.r.R().SetHeader("Content-Type", "application/json")
	if dryRun {
		req.SetQueryParam("dry_run", "true")
	}
	if minAge != "" {
		req.SetQueryParam("min_age", minAge)
	}
	resp, err := req.Post(fmt.Sprintf("/api/v1/blobstores/%s/compact", name))
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var res GCResult
	return &res, json.Unmarshal(resp.Body(), &res)
}
