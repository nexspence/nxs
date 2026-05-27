# nxs CLI — CI/CD Commands Expansion

**Date:** 2026-05-26
**Status:** Approved design
**Repo:** `github.com/skensell201/nxs` (module `github.com/nexspence/nxs`)

## Goal

Extend the `nxs` CLI (currently v0.1.0) for CI/CD pipeline workflows. Three capabilities:

1. **Batch push/pull** — recursive directory + glob upload/download with concurrency and continue-on-error.
2. **Promotion** — promote builds between repos, manage promotion requests (approve/reject), with coordinate-based component resolution.
3. **API tokens** — create/list/delete `nxs_*` tokens for CI service accounts.

Out of scope (YAGNI): vulnerability gate (not selected this iteration), token scopes management UI, format-aware deploy (maven/npm), batch resume/checkpoint.

## Server API (verified against nexspence-core)

| Capability | Endpoint | Notes |
|------------|----------|-------|
| Token list | `GET /api/v1/tokens` | metadata only, no plaintext |
| Token create | `POST /api/v1/tokens` `{name, scopes?, expiresInDays?}` | 201; plaintext in `token` field, once |
| Token delete | `DELETE /api/v1/tokens/:id` | 204 |
| Token policy | `GET /api/v1/auth/token-policy` | `{tokenMaxDays}` (public) |
| Promotion rules | `GET /api/v1/promotion/rules` | |
| Promote | `POST /api/v1/promotion/promote` `{rule_id, component_ids}` | 200 `{requests:[...]}`, 422 on gate fail |
| Promotion requests | `GET /api/v1/promotion/requests?status=` | |
| Approve | `POST /api/v1/promotion/requests/:id/approve` | admin |
| Reject | `POST /api/v1/promotion/requests/:id/reject` `{reason}` | admin |
| Asset search | `GET /service/rest/v1/search/assets?repository=` | `continuationToken` pagination; `Asset` has `path`, `downloadUrl`, `fileSize` |

**Domain JSON shapes:**

```
PromotionRule:    {id, name, from_repo, to_repo, path_filter, require_scan_pass, require_manual_approval, created_at, updated_at}
PromotionRequest: {id, rule_id, component_id, status, requested_by, reviewed_by?, reviewed_at?, completed_at?, error?, created_at}
UserToken:        {id, userId, username?, name, scopes?, lastUsed?, expiresAt?, createdAt, token?}  // token only on create
Asset:            {id, componentId, repository, path, fileSize, downloadUrl?, sha256?, ...}
```

## Command Surface (UX contract)

### Batch push (extends existing `push`)

```
nxs push <repo> <remote-prefix> <local>  [-r] [--concurrency N] [--continue-on-error]
```

- 3 args, no `-r` → single file (backward compatible, unchanged).
- `-r/--recursive` + `<local>` is a directory → walk tree; server path = `<remote-prefix>/<rel-path>`.
- `<local>` contains glob metachars (`*` `?` `[` `**`) → expand via `doublestar`; files uploaded under the prefix at their path relative to the non-glob base.
- `--concurrency N` (default 4).
- `--continue-on-error` → don't abort on first failure; print summary `N uploaded, M failed`; exit 1 if any failed.

### Batch pull (extends existing `pull`)

```
nxs pull <repo> <remote-prefix> [-r] [-o DIR] [--concurrency N] [--continue-on-error]
```

- No `-r` → single file (unchanged).
- `-r` → `search/assets` for the repo, filter assets by `path` prefix, download tree into `-o DIR` (default `.`).

### Tokens (new `token` group)

```
nxs token list
nxs token create <name> [--expires-days N] [--scope S]...
nxs token delete <id>
```

- `create` prints plaintext value once; in `--json` mode emits `.token` (for `nxs token create ci --json | jq -r .token`).

### Promotion (new `promote` group)

```
nxs promote rules
nxs promote run --rule <name|id> --component <ref>...  [--component-id <uuid>]...
nxs promote requests [--status pending|approved|rejected|done]
nxs promote approve <request-id>
nxs promote reject  <request-id> [--reason TEXT]
```

- `--component <ref>` = `group:name:version`. Resolution: load rule → take `from_repo` → `search` that repo → exact match `group==g && name==n && version==v` → component ID.
- `--component-id <uuid>` escape hatch, bypasses resolution.
- Resolution errors: rule not found / ambiguous; 0 matches; >1 matches — all produce clear messages.

## Architecture

### Client layer (`internal/client/`)

**`tokens.go`** (new):
```go
type Token struct { ID, Name, Token string; Scopes []string; ExpiresAt, LastUsed *time.Time; CreatedAt time.Time }
func (c *Client) TokenList() ([]Token, error)                                  // GET    /api/v1/tokens
func (c *Client) TokenCreate(name string, scopes []string, expDays *int) (*Token, error) // POST /api/v1/tokens
func (c *Client) TokenDelete(id string) error                                  // DELETE /api/v1/tokens/:id
```

**`promotion.go`** (new):
```go
type PromotionRule struct { ID, Name, FromRepo, ToRepo, PathFilter string; RequireScanPass, RequireManualApproval bool }
type PromotionRequest struct { ID, RuleID, ComponentID, Status, RequestedBy, Error string; ... }
func (c *Client) PromotionRules() ([]PromotionRule, error)
func (c *Client) Promote(ruleID string, componentIDs []string) ([]PromotionRequest, error)
func (c *Client) PromotionRequests(status string) ([]PromotionRequest, error)
func (c *Client) PromotionApprove(id string) error
func (c *Client) PromotionReject(id, reason string) error
```

**`components.go`** (extend): add `Asset` struct (`Path`, `DownloadURL`, `FileSize`) and
```go
func (c *Client) SearchAssets(repo, prefix string) ([]Asset, error)  // GET /service/rest/v1/search/assets
```
with `continuationToken` pagination (same loop as `Search`) and client-side prefix filter on `Path`.

### Batch engine (`internal/batch/`, new package)

```go
type Job    struct { LocalPath, RelPath string }
type Result struct { OK int; Failed []error }
func Walk(local string, recursive bool) ([]Job, error)   // dir walk / glob expand / single file
func RunPool(jobs []Job, concurrency int, continueOnError bool, fn func(Job) error) Result
```

- `Walk`: directory (recursive) or glob (`doublestar`) → list of jobs with `RelPath` relative to the non-glob base. Single file → one job.
- `RunPool`: worker pool (`sync.WaitGroup` + buffered semaphore channel), aggregates `Result`. `continueOnError=false` cancels remaining work on first error.
- `push`/`pull` commands call `Walk` → `RunPool`. With concurrency > 1, per-file progress bars are replaced by an aggregate counter `[12/40] uploading…`; per-file bars only when N=1 (preserves current single-file UX).

### Coordinate resolution (`cmd/promote.go`, not in client)

```go
func resolveComponents(ruleNameOrID string, refs, rawIDs []string) (ruleID string, compIDs []string, err error)
```
1. `PromotionRules()` → match by ID or name (error on missing/ambiguous).
2. For each `g:n:v` ref: `Search({Repo: rule.FromRepo, Query: n})` → exact match on Group/Name/Version → ID. 0 or >1 → error.
3. Append `rawIDs` directly.

### Output (`internal/output/`)

New tables following the existing `Printer` interface (rich / plain / json):
- `token list` — Name, Scopes, Expires, LastUsed.
- `promote rules` — Name, From→To, Gates (scan/approval).
- `promote requests` — ID, Component, Status, Error.

## Files

| File | Change |
|------|--------|
| `internal/client/tokens.go` | new |
| `internal/client/promotion.go` | new |
| `internal/client/components.go` | +`Asset`, +`SearchAssets` |
| `internal/batch/batch.go` | new package |
| `cmd/token.go` | new — `token` group |
| `cmd/promote.go` | new — `promote` group + `resolveComponents` |
| `cmd/push.go` | +batch flags, single/batch branch |
| `cmd/pull.go` | +batch flags, batch via `SearchAssets` |
| `internal/output/*.go` | new tables |
| `go.mod` | +`github.com/bmatcuk/doublestar/v4` |
| `README.md` | Tokens, Promotion, batch flags |

## Testing

`go test ./...`, httptest pattern (assert path/method/body, parse response):

- `client/tokens_test.go` — create returns plaintext `token`; list/delete path+method.
- `client/promotion_test.go` — promote body `{rule_id, component_ids}`; requests `?status=`; reject `{reason}`.
- `client/components_test.go` — `SearchAssets` pagination + prefix filter.
- `batch/batch_test.go` — `Walk` (dir, glob `**`, single file); `RunPool` (concurrency, continue-on-error collects errors without aborting).
- `cmd/promote_test.go` — `resolveComponents`: success, 0-match error, >1-match error, ambiguous rule, `--component-id` bypass.

**Verification gate:** `go build ./...` + `go test ./...` green before commit. Live-server manual check noted as a separate follow-up (requires a running Nexspence).
