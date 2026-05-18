# nxs

<p align="center">
  <strong>Official CLI for <a href="https://nexspence.online">Nexspence</a> — free, open-source artifact repository manager</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img src="https://img.shields.io/badge/License-AGPLv3-22c55e?style=flat-square">
  <img src="https://img.shields.io/github/v/release/skensell201/nxs?style=flat-square&color=3b82f6">
</p>

---

`nxs` lets you manage Nexspence repositories, users, and artifacts from the terminal or CI/CD pipelines — with rich output by default and `--json` / `--plain` modes for scripting.

---

## Installation

**macOS / Linux (curl):**
```bash
curl -sSfL https://raw.githubusercontent.com/skensell201/nxs/main/install.sh | sh
```

**Build from source:**
```bash
go install github.com/nexspence/nxs/cmd/nxs@latest
```

---

## Quick Start

```bash
# Authenticate (saves token to ~/.config/nxs/config.yaml)
nxs login --url http://nexspence:8081 --user admin

# List repositories
nxs repo list

# Upload an artifact
nxs push my-raw-repo assets/app-v1.0.tar.gz app-v1.0.tar.gz

# Download an artifact
nxs pull my-raw-repo app-v1.0.tar.gz --output ./downloads

# Search components
nxs search --repo maven-releases --q mylib
```

---

## Commands

### Auth
```
nxs login [--url URL] [--user USER] [--context NAME]   Authenticate and save token
nxs logout                                              Clear token for active context
nxs context list                                        List configured contexts
nxs context use <name>                                  Switch active context
```

### Repositories
```
nxs repo list [--format FORMAT] [--type TYPE]          List repositories
nxs repo create <name> --format FORMAT --type TYPE     Create a repository
nxs repo delete <name> [--force]                       Delete a repository
nxs repo info <name>                                   Show repository details
```

### Artifacts
```
nxs push <repo> <remote-path> <local-file>             Upload an artifact
nxs pull <repo> <remote-path> [-o DIR]                 Download an artifact
nxs search [--repo NAME] [--format FMT] [-q QUERY]     Search components
         [--tag KEY=VALUE]
```

### Users & Roles
```
nxs user list                                          List all users
nxs user create <username> --email EMAIL --password P  Create a user
nxs role assign <username> <role>                      Assign a role to a user
```

### Operations
```
nxs cleanup run <policy-name>                          Run a cleanup policy now
nxs migrate from <nexus-url> [--user U] [--repos]     Migrate from Nexus
                [--users] [--blobs]
nxs health [--watch]                                   Show server status
```

### Output flags (global)
```
--json          Machine-readable JSON output (stdout)
--plain         Tab-separated plain text, no colors
--url URL       Override server URL (env: NXS_URL)
--token TOKEN   Override auth token (env: NXS_TOKEN)
--context NAME  Use named context (env: NXS_CONTEXT)
```

---

## Configuration

`nxs` stores credentials in `~/.config/nxs/config.yaml`:

```yaml
current_context: prod
contexts:
  prod:
    url: https://nexspence.company.com
    token: nxs_abc123
  local:
    url: http://localhost:8081
    token: nxs_xyz789
```

**Environment variables override the config file:**

| Variable | Description |
|----------|-------------|
| `NXS_URL` | Server URL |
| `NXS_TOKEN` | Auth token or JWT |
| `NXS_CONTEXT` | Active context name |
| `NXS_CONFIG` | Path to config file (default: `~/.config/nxs/config.yaml`) |

**Priority:** `--flag` > env var > config file

---

## CI/CD Usage

```bash
# Use env vars — no config file needed in CI
export NXS_URL=https://nexspence.company.com
export NXS_TOKEN=${{ secrets.NXS_TOKEN }}

# Upload build artifact
nxs push releases target/app-1.2.0.jar com/example/app/1.2.0/app-1.2.0.jar

# JSON output for scripting
nxs repo list --json | jq '.[].name'

# Search with plain output for awk
nxs search --repo maven-releases --q mylib --plain | awk '{print $5}'
```

---

## Nexspence Server

`nxs` connects to a running [Nexspence](https://github.com/nexspence-oss/nexspence) instance.

**Run Nexspence locally in 30 seconds:**
```bash
git clone https://github.com/nexspence-oss/nexspence
cd nexspence && docker compose up -d
# → http://localhost:8081  admin / admin123
```

---

## License

AGPLv3 — see [LICENSE](LICENSE)
