# GitHub TUI - Development Plan

## Overview
A terminal UI (Bubble Tea) for interacting with GitHub PRs using the go-github SDK, with future extensibility for agent-based workflows.

## Tech Stack
- **Language**: Go
- **TUI**: Bubble Tea (charmbracelet/bubbletea) + Lip Gloss (styling) + Bubbles (components)
- **GitHub API**: google/go-github v6 (REST)
- **Auth**: Personal Access Token via `GITHUB_TOKEN` env var
- **Repo detection**: Auto-detect from git remote, with `--repo owner/repo` flag override

## Architecture

```
cmd/
  main.go              # Entry point, flag parsing, repo resolution
internal/
  github/
    client.go          # GitHub client wrapper, auth setup
    pulls.go           # PR listing, filtering, merging, reviewing
  tui/
    app.go             # Root model, top-level key bindings, screen routing
    prlist.go          # PR list view (table with state, author, title, checks)
    prdetail.go        # PR detail view (description, comments, checks, diff stats)
    styles.go          # Lip Gloss styles
  repo/
    detect.go          # Git remote parsing, flag override logic
```

## Screens & Features

### Screen 1: PR List
- Table columns: `#`, `Title`, `Author`, `State`, `Checks`, `Updated`
- Filter toggles:
  - **Mine** (PRs authored by authenticated user) — keybind `m`
  - **Review requested** — keybind `r`
  - **All** (default) — keybind `a`
- Search/filter by text — keybind `/`
- Sort by updated desc (default)
- Actions from list:
  - `Enter` → open PR detail
  - `M` → merge (only if checks pass & approved, confirm prompt)
  - `q` → quit
  - `R` → refresh

### Screen 2: PR Detail
- Tabs (navigate with `Tab`/`Shift+Tab`):
  1. **Overview**: title, body (rendered markdown), state, mergeable status, labels
  2. **Checks**: list of CI status checks with pass/fail/pending
  3. **Comments**: conversation thread (review comments + issue comments)
  4. **Files Changed**: file list with additions/deletions stats
- Actions:
  - `L` → LGTM (submit approving review with "LGTM" body)
  - `M` → merge (with confirmation)
  - `Esc` → back to list

## Implementation Tasks (ordered)

### Task 1: Project scaffold & GitHub client
- `go mod init`, dependencies
- `cmd/main.go`: flag parsing (`--repo`), env var `GITHUB_TOKEN`
- `internal/repo/detect.go`: parse `git remote get-url origin` for owner/repo
- `internal/github/client.go`: authenticated client constructor
- `internal/github/pulls.go`: `ListPRs(owner, repo) -> []PR`, `MergePR(...)`, `ApprovePR(...)`

### Task 2: TUI - PR List screen
- `internal/tui/styles.go`: color palette, table styles
- `internal/tui/prlist.go`: Bubble Tea model with table (bubbles/table), filter state, key bindings
- `internal/tui/app.go`: root model that delegates to prlist or prdetail
- `cmd/main.go`: wire up and launch

### Task 3: TUI - PR Detail screen
- `internal/tui/prdetail.go`: tabbed detail view with overview, checks, comments, files
- Merge & LGTM actions with confirmation prompts
- Wire into app.go navigation

### Task 4: Polish & agent hook prep
- Error handling, loading spinners, empty states
- Viewport scrolling for long content
- Expose a `Context` struct (`Owner`, `Repo`, `CurrentPR`, `FilteredPRs`) that future agent code can consume

## Future (out of scope for now)
- Agent integration: pass context (project, PR list, single PR) to an LLM agent
- PR creation, issue browsing, notifications
