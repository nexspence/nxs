# nxs CLI — CI/CD Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add batch push/pull, promotion, and API-token commands to the `nxs` CLI for CI/CD pipelines.

**Architecture:** New client methods (`tokens.go`, `promotion.go`, `SearchAssets` in `components.go`) wrap existing Nexspence REST endpoints. A new `internal/batch` package provides directory/glob walking + a worker pool. New cobra command groups `token` and `promote`; `push`/`pull` gain batch flags. Output uses the existing `Printer` interface.

**Tech Stack:** Go, cobra, resty/v2, doublestar/v4 (new), httptest for tests.

Spec: `docs/superpowers/specs/2026-05-26-nxs-cicd-commands-design.md`

---

## Conventions (apply to every task)

- Client methods live on `*client.Client`, use `c.r.R()` (resty) and `checkErr(resp)`; unmarshal with `json.Unmarshal(resp.Body(), &v)`.
- Commands: cobra, call `requireClient()` first, use package globals `nxsClient`, `printer`, `flagJSON`, `flagPlain`.
- Run `go build ./...` after each implementation step.
- Test package suffix `_test` (black-box), httptest servers — match `internal/client/client_test.go`.
- Commit after each task with the message shown.

---

## Task 1: Add `doublestar` dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

Run: `cd /Users/skensel/WORKING/AI/nxs && go get github.com/bmatcuk/doublestar/v4@latest`
Expected: `go.mod` gains `github.com/bmatcuk/doublestar/v4 vX.Y.Z`.

- [ ] **Step 2: Verify build still works**

Run: `go build ./...`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "build: add doublestar/v4 for glob matching"
```

---

## Task 2: Token client methods

**Files:**
- Create: `internal/client/tokens.go`
- Test: `internal/client/tokens_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/client/tokens_test.go`:

```go
package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

func TestClient_TokenList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tokens" || r.Method != http.MethodGet {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"t1","name":"ci","scopes":["read"]}]`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	toks, err := c.TokenList()
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 1 || toks[0].Name != "ci" {
		t.Errorf("unexpected: %+v", toks)
	}
}

func TestClient_TokenCreate(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tokens" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"t2","name":"deploy","token":"nxs_secret"}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	days := 30
	tok, err := c.TokenCreate("deploy", []string{"write"}, &days)
	if err != nil {
		t.Fatal(err)
	}
	if tok.Token != "nxs_secret" {
		t.Errorf("expected plaintext token, got %q", tok.Token)
	}
	if gotBody["name"] != "deploy" {
		t.Errorf("name not sent: %+v", gotBody)
	}
	if gotBody["expiresInDays"].(float64) != 30 {
		t.Errorf("expiresInDays not sent: %+v", gotBody)
	}
}

func TestClient_TokenDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.TokenDelete("t2"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/v1/tokens/t2" {
		t.Errorf("unexpected %s %s", gotMethod, gotPath)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/client/ -run TestClient_Token -v`
Expected: compile failure — `c.TokenList undefined`.

- [ ] **Step 3: Write the implementation**

Create `internal/client/tokens.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/client/ -run TestClient_Token -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/client/tokens.go internal/client/tokens_test.go
git commit -m "feat(client): add token list/create/delete methods"
```

---

## Task 3: Promotion client methods

**Files:**
- Create: `internal/client/promotion.go`
- Test: `internal/client/promotion_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/client/promotion_test.go`:

```go
package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

func TestClient_PromotionRules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/promotion/rules" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"r1","name":"to-prod","from_repo":"staging","to_repo":"prod","require_manual_approval":true}]`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	rules, err := c.PromotionRules()
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].FromRepo != "staging" || !rules[0].RequireManualApproval {
		t.Errorf("unexpected: %+v", rules)
	}
}

func TestClient_Promote(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/promotion/promote" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"requests":[{"id":"pr1","rule_id":"r1","component_id":"c1","status":"pending"}]}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	reqs, err := c.Promote("r1", []string{"c1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 || reqs[0].ID != "pr1" {
		t.Errorf("unexpected: %+v", reqs)
	}
	if gotBody["rule_id"] != "r1" {
		t.Errorf("rule_id not sent: %+v", gotBody)
	}
	ids := gotBody["component_ids"].([]any)
	if len(ids) != 1 || ids[0] != "c1" {
		t.Errorf("component_ids not sent: %+v", gotBody)
	}
}

func TestClient_PromotionRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("status"); got != "pending" {
			t.Errorf("status filter not sent, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"pr1","status":"pending","component_id":"c1"}]`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	reqs, err := c.PromotionRequests("pending")
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 || reqs[0].ID != "pr1" {
		t.Errorf("unexpected: %+v", reqs)
	}
}

func TestClient_PromotionApproveReject(t *testing.T) {
	var paths []string
	var rejectBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/api/v1/promotion/requests/pr1/reject" {
			b, _ := io.ReadAll(r.Body)
			json.Unmarshal(b, &rejectBody)
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.PromotionApprove("pr1"); err != nil {
		t.Fatal(err)
	}
	if err := c.PromotionReject("pr1", "bad scan"); err != nil {
		t.Fatal(err)
	}
	if paths[0] != "/api/v1/promotion/requests/pr1/approve" {
		t.Errorf("approve path wrong: %v", paths)
	}
	if rejectBody["reason"] != "bad scan" {
		t.Errorf("reason not sent: %+v", rejectBody)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/client/ -run TestClient_Promot -v`
Expected: compile failure — `c.PromotionRules undefined`.

- [ ] **Step 3: Write the implementation**

Create `internal/client/promotion.go`:

```go
package client

import (
	"encoding/json"
	"time"
)

type PromotionRule struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	FromRepo              string `json:"from_repo"`
	ToRepo                string `json:"to_repo"`
	PathFilter            string `json:"path_filter,omitempty"`
	RequireScanPass       bool   `json:"require_scan_pass"`
	RequireManualApproval bool   `json:"require_manual_approval"`
}

type PromotionRequest struct {
	ID          string     `json:"id"`
	RuleID      string     `json:"rule_id"`
	ComponentID string     `json:"component_id"`
	Status      string     `json:"status"`
	RequestedBy string     `json:"requested_by"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (c *Client) PromotionRules() ([]PromotionRule, error) {
	resp, err := c.r.R().Get("/api/v1/promotion/rules")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var rules []PromotionRule
	return rules, json.Unmarshal(resp.Body(), &rules)
}

func (c *Client) Promote(ruleID string, componentIDs []string) ([]PromotionRequest, error) {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]any{"rule_id": ruleID, "component_ids": componentIDs}).
		Post("/api/v1/promotion/promote")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var out struct {
		Requests []PromotionRequest `json:"requests"`
	}
	return out.Requests, json.Unmarshal(resp.Body(), &out)
}

func (c *Client) PromotionRequests(status string) ([]PromotionRequest, error) {
	req := c.r.R()
	if status != "" {
		req = req.SetQueryParam("status", status)
	}
	resp, err := req.Get("/api/v1/promotion/requests")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var reqs []PromotionRequest
	return reqs, json.Unmarshal(resp.Body(), &reqs)
}

func (c *Client) PromotionApprove(id string) error {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		Post("/api/v1/promotion/requests/" + id + "/approve")
	if err != nil {
		return err
	}
	return checkErr(resp)
}

func (c *Client) PromotionReject(id, reason string) error {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]any{"reason": reason}).
		Post("/api/v1/promotion/requests/" + id + "/reject")
	if err != nil {
		return err
	}
	return checkErr(resp)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/client/ -run TestClient_Promot -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/client/promotion.go internal/client/promotion_test.go
git commit -m "feat(client): add promotion rules/promote/requests/approve/reject"
```

---

## Task 4: SearchAssets client method

**Files:**
- Modify: `internal/client/components.go` (append `Asset` struct + `SearchAssets`)
- Test: `internal/client/components_test.go` (new file; existing search test stays in `client_test.go`)

- [ ] **Step 1: Write the failing test**

Create `internal/client/components_test.go`:

```go
package client_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

func TestClient_SearchAssets_PrefixFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("repository") != "raw-hosted" {
			t.Errorf("repository not sent: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[
			{"id":"a1","repository":"raw-hosted","path":"dist/app.tar.gz","fileSize":10},
			{"id":"a2","repository":"raw-hosted","path":"other/x.txt","fileSize":5}
		],"continuationToken":null}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	assets, err := c.SearchAssets("raw-hosted", "dist/")
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 1 || assets[0].Path != "dist/app.tar.gz" {
		t.Errorf("prefix filter failed: %+v", assets)
	}
}

func TestClient_SearchAssets_Pagination(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("continuationToken") == "" {
			page++
			w.Write([]byte(`{"items":[{"id":"a1","path":"p1"}],"continuationToken":"50"}`))
			return
		}
		w.Write([]byte(`{"items":[{"id":"a2","path":"p2"}],"continuationToken":null}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	assets, err := c.SearchAssets("raw-hosted", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 {
		t.Errorf("expected 2 assets across pages, got %d", len(assets))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/client/ -run TestClient_SearchAssets -v`
Expected: compile failure — `c.SearchAssets undefined`.

- [ ] **Step 3: Write the implementation**

Append to `internal/client/components.go` (add `"strings"` to the existing import block):

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/client/ -run TestClient_SearchAssets -v`
Expected: PASS (2 tests). Then `go test ./internal/client/...` — all green.

- [ ] **Step 5: Commit**

```bash
git add internal/client/components.go internal/client/components_test.go
git commit -m "feat(client): add SearchAssets with prefix filter and pagination"
```

---

## Task 5: Batch engine (Walk + RunPool)

**Files:**
- Create: `internal/batch/batch.go`
- Test: `internal/batch/batch_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/batch/batch_test.go`:

```go
package batch_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/nexspence/nxs/internal/batch"
)

func TestWalk_SingleFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	os.WriteFile(f, []byte("x"), 0o644)

	jobs, err := batch.Walk(f, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].LocalPath != f || jobs[0].RelPath != "a.txt" {
		t.Errorf("unexpected: %+v", jobs)
	}
}

func TestWalk_Directory(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("y"), 0o644)

	jobs, err := batch.Walk(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	rels := []string{}
	for _, j := range jobs {
		rels = append(rels, j.RelPath)
	}
	sort.Strings(rels)
	if len(rels) != 2 || rels[0] != "a.txt" || rels[1] != "sub/b.txt" {
		t.Errorf("unexpected rels: %v", rels)
	}
}

func TestWalk_Glob(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "build"), 0o755)
	os.WriteFile(filepath.Join(dir, "build", "app.jar"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "build", "app.txt"), []byte("y"), 0o644)

	jobs, err := batch.Walk(filepath.Join(dir, "build", "**", "*.jar"), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || filepath.Base(jobs[0].LocalPath) != "app.jar" {
		t.Errorf("glob mismatch: %+v", jobs)
	}
}

func TestRunPool_AllSucceed(t *testing.T) {
	jobs := []batch.Job{{RelPath: "1"}, {RelPath: "2"}, {RelPath: "3"}}
	var count int32
	res := batch.RunPool(jobs, 2, false, func(j batch.Job) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	if res.OK != 3 || len(res.Failed) != 0 {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestRunPool_ContinueOnError(t *testing.T) {
	jobs := []batch.Job{{RelPath: "1"}, {RelPath: "2"}, {RelPath: "3"}}
	res := batch.RunPool(jobs, 2, true, func(j batch.Job) error {
		if j.RelPath == "2" {
			return fmt.Errorf("boom")
		}
		return nil
	})
	if res.OK != 2 || len(res.Failed) != 1 {
		t.Errorf("expected 2 ok / 1 failed, got %+v", res)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/batch/ -v`
Expected: compile failure — package `batch` does not exist.

- [ ] **Step 3: Write the implementation**

Create `internal/batch/batch.go`:

```go
// Package batch expands local file selections into upload jobs and runs work
// concurrently with a bounded worker pool.
package batch

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
)

// Job is a single file to transfer. RelPath is the path used to build the
// remote location (always forward-slash separated).
type Job struct {
	LocalPath string
	RelPath   string
}

// Result aggregates a pool run.
type Result struct {
	OK     int
	Failed []error
}

func hasGlob(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

// Walk turns local (a file, a directory, or a glob pattern) into jobs.
//   - single file: one job, RelPath = base name.
//   - directory (recursive=true): every file underneath, RelPath relative to dir.
//   - glob: every match, RelPath relative to the longest non-glob base segment.
func Walk(local string, recursive bool) ([]Job, error) {
	if hasGlob(local) {
		return walkGlob(local)
	}
	info, err := os.Stat(local)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		if !recursive {
			return nil, fmt.Errorf("%s is a directory; pass -r to upload recursively", local)
		}
		return walkDir(local)
	}
	return []Job{{LocalPath: local, RelPath: filepath.Base(local)}}, nil
}

func walkDir(dir string) ([]Job, error) {
	var jobs []Job
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		jobs = append(jobs, Job{LocalPath: path, RelPath: filepath.ToSlash(rel)})
		return nil
	})
	return jobs, err
}

func walkGlob(pattern string) ([]Job, error) {
	base, _ := doublestar.SplitPattern(filepath.ToSlash(pattern))
	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, err
	}
	var jobs []Job
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil || info.IsDir() {
			continue
		}
		rel := filepath.Base(m)
		if base != "" && base != "." {
			if r, err := filepath.Rel(base, m); err == nil {
				rel = r
			}
		}
		jobs = append(jobs, Job{LocalPath: m, RelPath: filepath.ToSlash(rel)})
	}
	return jobs, nil
}

// RunPool executes fn over jobs with up to concurrency workers. When
// continueOnError is false the first error stops new work from starting.
func RunPool(jobs []Job, concurrency int, continueOnError bool, fn func(Job) error) Result {
	if concurrency < 1 {
		concurrency = 1
	}
	var (
		mu      sync.Mutex
		res     Result
		stopped bool
		wg      sync.WaitGroup
	)
	sem := make(chan struct{}, concurrency)
	for _, j := range jobs {
		mu.Lock()
		if stopped {
			mu.Unlock()
			break
		}
		mu.Unlock()

		wg.Add(1)
		sem <- struct{}{}
		go func(j Job) {
			defer wg.Done()
			defer func() { <-sem }()
			err := fn(j)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				res.Failed = append(res.Failed, fmt.Errorf("%s: %w", j.RelPath, err))
				if !continueOnError {
					stopped = true
				}
			} else {
				res.OK++
			}
		}(j)
	}
	wg.Wait()
	return res
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/batch/ -v`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/batch/
git commit -m "feat(batch): add Walk (dir/glob) and RunPool worker pool"
```

---

## Task 6: `token` command group

**Files:**
- Create: `cmd/token.go`

- [ ] **Step 1: Write the implementation**

Create `cmd/token.go`:

```go
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage API tokens",
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your API tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		toks, err := nxsClient.TokenList()
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(toks)
			return nil
		}
		rows := make([][]string, 0, len(toks))
		for _, t := range toks {
			expires := "never"
			if t.ExpiresAt != nil {
				expires = t.ExpiresAt.Format(time.RFC3339)
			}
			last := "never"
			if t.LastUsed != nil {
				last = t.LastUsed.Format(time.RFC3339)
			}
			rows = append(rows, []string{t.ID, t.Name, strings.Join(t.Scopes, ","), expires, last})
		}
		printer.Table([]string{"ID", "NAME", "SCOPES", "EXPIRES", "LAST USED"}, rows)
		return nil
	},
}

var tokenCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new API token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		scopes, _ := cmd.Flags().GetStringSlice("scope")
		days, _ := cmd.Flags().GetInt("expires-days")
		var expPtr *int
		if days > 0 {
			expPtr = &days
		}
		tok, err := nxsClient.TokenCreate(args[0], scopes, expPtr)
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(tok)
			return nil
		}
		printer.Success(fmt.Sprintf("Token %q created", tok.Name))
		fmt.Println(tok.Token)
		fmt.Fprintln(cmd.ErrOrStderr(), "Save this token now — it will not be shown again.")
		return nil
	},
}

var tokenDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an API token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		if err := nxsClient.TokenDelete(args[0]); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Token %s deleted", args[0]))
		return nil
	},
}

func init() {
	tokenCreateCmd.Flags().StringSlice("scope", nil, "Scope to grant (repeatable)")
	tokenCreateCmd.Flags().Int("expires-days", 0, "Days until expiry (0 = server default / never)")
	tokenCmd.AddCommand(tokenListCmd, tokenCreateCmd, tokenDeleteCmd)
	rootCmd.AddCommand(tokenCmd)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: exit 0.

- [ ] **Step 3: Smoke test help output**

Run: `go run ./cmd/nxs token --help`
Expected: lists `list`, `create`, `delete` subcommands.

- [ ] **Step 4: Commit**

```bash
git add cmd/token.go
git commit -m "feat(cmd): add token list/create/delete commands"
```

---

## Task 7: `promote` command group + coordinate resolution

**Files:**
- Create: `cmd/promote.go`
- Test: `cmd/promote_test.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/promote_test.go`:

```go
package cmd

import (
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

// fakeResolver satisfies the lookups resolveComponents needs.
type fakeResolver struct {
	rules  []client.PromotionRule
	search []client.Component
}

func (f fakeResolver) PromotionRules() ([]client.PromotionRule, error) { return f.rules, nil }
func (f fakeResolver) Search(p client.SearchParams) ([]client.Component, error) {
	return f.search, nil
}

func TestResolveComponents_ByCoordinates(t *testing.T) {
	r := fakeResolver{
		rules:  []client.PromotionRule{{ID: "r1", Name: "to-prod", FromRepo: "staging"}},
		search: []client.Component{{ID: "c1", Group: "com.acme", Name: "app", Version: "1.0"}},
	}
	ruleID, ids, err := resolveComponents(r, "to-prod", []string{"com.acme:app:1.0"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ruleID != "r1" || len(ids) != 1 || ids[0] != "c1" {
		t.Errorf("unexpected: rule=%s ids=%v", ruleID, ids)
	}
}

func TestResolveComponents_NoMatch(t *testing.T) {
	r := fakeResolver{
		rules:  []client.PromotionRule{{ID: "r1", Name: "to-prod", FromRepo: "staging"}},
		search: []client.Component{{ID: "c1", Group: "com.acme", Name: "app", Version: "2.0"}},
	}
	_, _, err := resolveComponents(r, "to-prod", []string{"com.acme:app:1.0"}, nil)
	if err == nil {
		t.Fatal("expected no-match error")
	}
}

func TestResolveComponents_RuleNotFound(t *testing.T) {
	r := fakeResolver{rules: []client.PromotionRule{{ID: "r1", Name: "to-prod"}}}
	_, _, err := resolveComponents(r, "nope", nil, []string{"c9"})
	if err == nil {
		t.Fatal("expected rule-not-found error")
	}
}

func TestResolveComponents_RawIDBypass(t *testing.T) {
	r := fakeResolver{rules: []client.PromotionRule{{ID: "r1", Name: "to-prod", FromRepo: "staging"}}}
	ruleID, ids, err := resolveComponents(r, "r1", nil, []string{"raw-uuid"})
	if err != nil {
		t.Fatal(err)
	}
	if ruleID != "r1" || len(ids) != 1 || ids[0] != "raw-uuid" {
		t.Errorf("unexpected: rule=%s ids=%v", ruleID, ids)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestResolveComponents -v`
Expected: compile failure — `resolveComponents undefined`.

- [ ] **Step 3: Write the implementation**

Create `cmd/promote.go`:

```go
package cmd

import (
	"fmt"
	"strings"

	"github.com/nexspence/nxs/internal/client"
	"github.com/spf13/cobra"
)

// promoteResolver is the subset of *client.Client that resolveComponents needs,
// so the resolver can be unit-tested with a fake.
type promoteResolver interface {
	PromotionRules() ([]client.PromotionRule, error)
	Search(client.SearchParams) ([]client.Component, error)
}

// resolveComponents maps a rule name-or-id plus component references to a
// rule ID and concrete component IDs. coordRefs are "group:name:version"
// strings resolved against the rule's from_repo; rawIDs are passed through.
func resolveComponents(r promoteResolver, ruleNameOrID string, coordRefs, rawIDs []string) (string, []string, error) {
	rules, err := r.PromotionRules()
	if err != nil {
		return "", nil, err
	}
	var rule *client.PromotionRule
	for i := range rules {
		if rules[i].ID == ruleNameOrID || rules[i].Name == ruleNameOrID {
			rule = &rules[i]
			break
		}
	}
	if rule == nil {
		return "", nil, fmt.Errorf("promotion rule %q not found", ruleNameOrID)
	}

	ids := append([]string{}, rawIDs...)
	for _, ref := range coordRefs {
		parts := strings.SplitN(ref, ":", 3)
		if len(parts) != 3 {
			return "", nil, fmt.Errorf("invalid component %q: expected group:name:version", ref)
		}
		group, name, version := parts[0], parts[1], parts[2]
		comps, err := r.Search(client.SearchParams{Repo: rule.FromRepo, Query: name})
		if err != nil {
			return "", nil, err
		}
		var matched []string
		for _, c := range comps {
			if c.Group == group && c.Name == name && c.Version == version {
				matched = append(matched, c.ID)
			}
		}
		switch len(matched) {
		case 0:
			return "", nil, fmt.Errorf("no component %s in repo %q", ref, rule.FromRepo)
		case 1:
			ids = append(ids, matched[0])
		default:
			return "", nil, fmt.Errorf("ambiguous component %s: %d matches", ref, len(matched))
		}
	}
	return rule.ID, ids, nil
}

var promoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote artifacts between repositories",
}

var promoteRulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "List promotion rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		rules, err := nxsClient.PromotionRules()
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(rules)
			return nil
		}
		rows := make([][]string, 0, len(rules))
		for _, r := range rules {
			gates := []string{}
			if r.RequireScanPass {
				gates = append(gates, "scan")
			}
			if r.RequireManualApproval {
				gates = append(gates, "approval")
			}
			rows = append(rows, []string{r.ID, r.Name, r.FromRepo + "→" + r.ToRepo, strings.Join(gates, ",")})
		}
		printer.Table([]string{"ID", "NAME", "FLOW", "GATES"}, rows)
		return nil
	},
}

var promoteRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Promote components via a rule",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		rule, _ := cmd.Flags().GetString("rule")
		coords, _ := cmd.Flags().GetStringSlice("component")
		rawIDs, _ := cmd.Flags().GetStringSlice("component-id")
		if rule == "" {
			return fmt.Errorf("--rule is required")
		}
		if len(coords) == 0 && len(rawIDs) == 0 {
			return fmt.Errorf("at least one --component or --component-id is required")
		}
		ruleID, ids, err := resolveComponents(nxsClient, rule, coords, rawIDs)
		if err != nil {
			return err
		}
		reqs, err := nxsClient.Promote(ruleID, ids)
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(reqs)
			return nil
		}
		rows := make([][]string, 0, len(reqs))
		for _, rq := range reqs {
			rows = append(rows, []string{rq.ID, rq.ComponentID, rq.Status, rq.Error})
		}
		printer.Table([]string{"REQUEST", "COMPONENT", "STATUS", "ERROR"}, rows)
		return nil
	},
}

var promoteRequestsCmd = &cobra.Command{
	Use:   "requests",
	Short: "List promotion requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		status, _ := cmd.Flags().GetString("status")
		reqs, err := nxsClient.PromotionRequests(status)
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(reqs)
			return nil
		}
		rows := make([][]string, 0, len(reqs))
		for _, rq := range reqs {
			rows = append(rows, []string{rq.ID, rq.ComponentID, rq.Status, rq.Error})
		}
		printer.Table([]string{"REQUEST", "COMPONENT", "STATUS", "ERROR"}, rows)
		return nil
	},
}

var promoteApproveCmd = &cobra.Command{
	Use:   "approve <request-id>",
	Short: "Approve a promotion request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		if err := nxsClient.PromotionApprove(args[0]); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Request %s approved", args[0]))
		return nil
	},
}

var promoteRejectCmd = &cobra.Command{
	Use:   "reject <request-id>",
	Short: "Reject a promotion request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		reason, _ := cmd.Flags().GetString("reason")
		if err := nxsClient.PromotionReject(args[0], reason); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Request %s rejected", args[0]))
		return nil
	},
}

func init() {
	promoteRunCmd.Flags().String("rule", "", "Promotion rule name or ID (required)")
	promoteRunCmd.Flags().StringSlice("component", nil, "Component as group:name:version (repeatable)")
	promoteRunCmd.Flags().StringSlice("component-id", nil, "Raw component UUID (repeatable)")
	promoteRequestsCmd.Flags().String("status", "", "Filter by status (pending/approved/rejected/done)")
	promoteRejectCmd.Flags().String("reason", "", "Rejection reason")
	promoteCmd.AddCommand(promoteRulesCmd, promoteRunCmd, promoteRequestsCmd, promoteApproveCmd, promoteRejectCmd)
	rootCmd.AddCommand(promoteCmd)
}
```

Note: `*client.Client` satisfies `promoteResolver` because it has `PromotionRules()` (Task 3) and `Search(SearchParams)` (existing). The test's `fakeResolver` implements the same two methods.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run TestResolveComponents -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Verify build + help**

Run: `go build ./... && go run ./cmd/nxs promote --help`
Expected: lists `rules`, `run`, `requests`, `approve`, `reject`.

- [ ] **Step 6: Commit**

```bash
git add cmd/promote.go cmd/promote_test.go
git commit -m "feat(cmd): add promote command group with coordinate resolution"
```

---

## Task 8: Batch push

**Files:**
- Modify: `cmd/push.go`

- [ ] **Step 1: Rewrite push.go with batch support**

Replace the full contents of `cmd/push.go`:

```go
package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/nexspence/nxs/internal/batch"
	"github.com/nexspence/nxs/internal/output"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <repo> <remote-prefix> <local>",
	Short: "Upload an artifact (or a directory/glob with -r) to a repository",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		repo, remotePrefix, local := args[0], args[1], args[2]
		recursive, _ := cmd.Flags().GetBool("recursive")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

		jobs, err := batch.Walk(local, recursive)
		if err != nil {
			return err
		}

		// Single file → preserve the original per-file progress bar UX.
		if len(jobs) == 1 && !recursive && jobs[0].RelPath == filepath.Base(local) {
			remotePath := joinRemote(remotePrefix, jobs[0].RelPath)
			progressFn := func(size int64) io.Writer {
				bar := output.NewProgress(size, "Uploading "+filepath.Base(local), flagJSON, flagPlain)
				if bar == nil {
					return nil
				}
				return bar
			}
			if err := nxsClient.Push(repo, remotePath, jobs[0].LocalPath, progressFn); err != nil {
				return err
			}
			printer.Success(fmt.Sprintf("Uploaded %s → %s/%s", local, repo, remotePath))
			return nil
		}

		total := len(jobs)
		done := 0
		res := batch.RunPool(jobs, concurrency, continueOnError, func(j batch.Job) error {
			remotePath := joinRemote(remotePrefix, j.RelPath)
			err := nxsClient.Push(repo, remotePath, j.LocalPath, nil)
			done++
			if !flagJSON {
				fmt.Fprintf(cmd.ErrOrStderr(), "[%d/%d] %s\n", done, total, j.RelPath)
			}
			return err
		})

		printer.Success(fmt.Sprintf("%d uploaded, %d failed", res.OK, len(res.Failed)))
		for _, e := range res.Failed {
			printer.Error(e.Error())
		}
		if len(res.Failed) > 0 {
			return fmt.Errorf("%d uploads failed", len(res.Failed))
		}
		return nil
	},
}

// joinRemote joins a remote prefix and a relative path with a single slash,
// tolerating empty/"." prefixes.
func joinRemote(prefix, rel string) string {
	if prefix == "" || prefix == "." || prefix == "/" {
		return rel
	}
	return prefix + "/" + rel
}

func init() {
	pushCmd.Flags().BoolP("recursive", "r", false, "Upload a directory tree")
	pushCmd.Flags().Int("concurrency", 4, "Parallel uploads")
	pushCmd.Flags().Bool("continue-on-error", false, "Keep going after a failed upload")
	rootCmd.AddCommand(pushCmd)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: exit 0.

- [ ] **Step 3: Smoke test help**

Run: `go run ./cmd/nxs push --help`
Expected: shows `-r`, `--concurrency`, `--continue-on-error`.

- [ ] **Step 4: Commit**

```bash
git add cmd/push.go
git commit -m "feat(cmd): batch push with recursive/glob, concurrency, continue-on-error"
```

---

## Task 9: Batch pull

**Files:**
- Modify: `cmd/pull.go`

- [ ] **Step 1: Rewrite pull.go with batch support**

Replace the full contents of `cmd/pull.go`:

```go
package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nexspence/nxs/internal/batch"
	"github.com/nexspence/nxs/internal/output"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <repo> <remote-path-or-prefix>",
	Short: "Download an artifact (or a path prefix with -r) from a repository",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		repo, remote := args[0], args[1]
		outDir, _ := cmd.Flags().GetString("output")
		recursive, _ := cmd.Flags().GetBool("recursive")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

		if !recursive {
			filename := filepath.Base(remote)
			localPath := filepath.Join(outDir, filename)
			progressFn := func(size int64) io.Writer {
				bar := output.NewProgress(size, "Downloading "+filename, flagJSON, flagPlain)
				if bar == nil {
					return nil
				}
				return bar
			}
			if err := nxsClient.Pull(repo, remote, localPath, progressFn); err != nil {
				return err
			}
			printer.Success(fmt.Sprintf("Downloaded %s/%s → %s", repo, remote, localPath))
			return nil
		}

		assets, err := nxsClient.SearchAssets(repo, remote)
		if err != nil {
			return err
		}
		if len(assets) == 0 {
			return fmt.Errorf("no assets under %q in repo %q", remote, repo)
		}

		jobs := make([]batch.Job, 0, len(assets))
		for _, a := range assets {
			jobs = append(jobs, batch.Job{LocalPath: a.Path, RelPath: a.Path})
		}

		total := len(jobs)
		done := 0
		res := batch.RunPool(jobs, concurrency, continueOnError, func(j batch.Job) error {
			localPath := filepath.Join(outDir, filepath.FromSlash(j.RelPath))
			if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
				return err
			}
			err := nxsClient.Pull(repo, j.RelPath, localPath, nil)
			done++
			if !flagJSON {
				fmt.Fprintf(cmd.ErrOrStderr(), "[%d/%d] %s\n", done, total, j.RelPath)
			}
			return err
		})

		printer.Success(fmt.Sprintf("%d downloaded, %d failed", res.OK, len(res.Failed)))
		for _, e := range res.Failed {
			printer.Error(e.Error())
		}
		if len(res.Failed) > 0 {
			return fmt.Errorf("%d downloads failed", len(res.Failed))
		}
		return nil
	},
}

func init() {
	pullCmd.Flags().StringP("output", "o", ".", "Output directory")
	pullCmd.Flags().BoolP("recursive", "r", false, "Download every asset under the path prefix")
	pullCmd.Flags().Int("concurrency", 4, "Parallel downloads")
	pullCmd.Flags().Bool("continue-on-error", false, "Keep going after a failed download")
	rootCmd.AddCommand(pullCmd)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: exit 0.

- [ ] **Step 3: Smoke test help**

Run: `go run ./cmd/nxs pull --help`
Expected: shows `-r`, `--concurrency`, `--continue-on-error`, `-o`.

- [ ] **Step 4: Commit**

```bash
git add cmd/pull.go
git commit -m "feat(cmd): batch pull by path prefix with concurrency"
```

---

## Task 10: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update the Artifacts section**

In `README.md`, replace the `### Artifacts` fenced block with:

```
nxs push <repo> <remote-prefix> <local>                Upload a file
       [-r] [--concurrency N] [--continue-on-error]    …a directory or glob (-r)
nxs pull <repo> <remote-prefix> [-o DIR]               Download a file
       [-r] [--concurrency N] [--continue-on-error]    …everything under a prefix (-r)
nxs search [--repo NAME] [--format FMT] [-q QUERY]     Search components
         [--tag KEY=VALUE]
```

- [ ] **Step 2: Add Tokens and Promotion sections**

In `README.md`, after the `### Users & Roles` block, insert:

```
### API Tokens
```
nxs token list                                         List your tokens
nxs token create <name> [--expires-days N] [--scope S] Create a token (printed once)
nxs token delete <id>                                  Revoke a token
```

### Promotion
```
nxs promote rules                                      List promotion rules
nxs promote run --rule <name|id>                       Promote components
              --component group:name:version           …by coordinates (repeatable)
              [--component-id UUID]                     …or by raw ID
nxs promote requests [--status STATUS]                 List promotion requests
nxs promote approve <request-id>                       Approve a request
nxs promote reject <request-id> [--reason TEXT]        Reject a request
```
```

(Match the existing README's heading + fenced-block style; keep the surrounding sections intact.)

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: document batch push/pull, token, and promote commands"
```

---

## Task 11: Final verification

- [ ] **Step 1: Full build + test**

Run: `go build ./... && go test ./...`
Expected: all packages PASS, exit 0.

- [ ] **Step 2: go vet**

Run: `go vet ./...`
Expected: no findings.

- [ ] **Step 3: Confirm command tree**

Run: `go run ./cmd/nxs --help`
Expected: top-level lists `token` and `promote` alongside existing commands.

- [ ] **Step 4 (manual, deferred):** Against a running Nexspence, verify `nxs token create`, `nxs push -r ./dist myrepo`, `nxs pull -r myrepo dist/ -o ./out`, and a full `promote run → requests → approve` flow. Note this as a follow-up if no server is available.
```
