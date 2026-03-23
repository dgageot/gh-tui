# TODO — GitHub TUI

## Bugs & Correctness

- [ ] **Review-requested filter is fake** — `FilterReviewRequested` currently just shows "not mine" PRs. Use the `requested_reviewers` field from the API or the `review-requested` search qualifier to actually filter PRs where the current user's review is requested.
- [ ] **Checks not loaded in list view** — `ListPRs` never populates `PR.Checks`, so the "Checks" column always shows `—`. Either batch-fetch checks per PR or fetch combined status.
- [ ] **Detail load errors silently swallowed** — `loadDetail` ignores errors from `GetChecks`, `GetComments`, `GetChangedFiles`. Surface partial errors to the user.
- [ ] **Merge from list bypasses confirmation** — Pressing `M` on the list opens detail with `confirm = "merge"` but the confirm dialog renders only after detail data loads, which is racy. Handle the flow cleanly.
- [ ] **Pagination missing** — `ListPRs` fetches only the first 50 PRs. `GetChangedFiles` fetches only the first 100. Implement pagination for repos with many PRs/files.
- [ ] **`go.mod` Go version** — `go 1.26.1` is not a valid Go version. Fix to actual version (e.g. `go 1.22`).

## Missing Features (PR Workflow)

- [ ] **View actual diff / patch** — Files tab only shows stats. Add the ability to view the actual diff content for a selected file (use `GetRaw` or the patch field from the API).
- [ ] **Review comments (inline)** — Only issue comments and top-level review bodies are shown. Fetch and display inline review comments (`PullRequests.ListComments`) threaded by file/line.
- [ ] **Submit review comment** — Allow writing a comment (not just LGTM) from the detail view.
- [ ] **Request changes** — Support submitting a "request changes" review, not only approve.
- [ ] **Merge method selection** — Currently hardcoded to `squash`. Let user choose `merge`, `squash`, or `rebase` (or read repo default).
- [ ] **Close / reopen PR** — Allow closing or reopening a PR from the detail view.
- [ ] **Check out PR branch locally** — Keybind to run `git checkout` / `git switch` to the PR's head branch.
- [ ] **Open in browser** — Keybind to open the PR URL in the default browser.
- [ ] **Closed/merged PR listing** — Allow toggling the list to show closed/merged PRs, not just open.
- [ ] **PR labels management** — Add/remove labels from detail view.
- [ ] **Assignees & reviewers** — Display and manage assignees and requested reviewers.

## UX / Polish

- [ ] **Loading spinner** — Replace static "Loading..." text with an animated spinner (use `bubbles/spinner`).
- [ ] **Markdown rendering** — Render PR body as styled markdown (use `charmbracelet/glamour`).
- [ ] **Relative timestamps** — Show "2 hours ago" instead of "Jan 02 15:04".
- [ ] **Empty state messaging** — Show helpful messages when no PRs match the current filter/search.
- [ ] **Notification on action success** — Flash a styled success/error banner that auto-dismisses after a few seconds.
- [ ] **Help screen** — Add a `?` keybind that shows a full help overlay with all available keybindings.
- [ ] **Mouse support** — Enable mouse clicks on table rows and tabs.
- [ ] **Color-coded state column** — Color the State column green/red/yellow based on open/closed/draft.
- [ ] **Checks summary in list** — Show a single ✓/✗/● icon per PR in the list based on combined check status.
- [ ] **Resize handling for detail** — Ensure viewport and tabs reflow properly on terminal resize while in detail view.
- [ ] **Confirmation dialog as overlay** — Render the merge/LGTM confirmation as a centered modal overlay instead of inline text.
- [ ] **Search highlight** — Highlight matching text in PR titles when searching.

## Architecture / Code Quality

- [ ] **Error types** — Define typed errors (rate limit, auth failure, not found) and show user-friendly messages.
- [ ] **Context cancellation** — Use proper context cancellation when switching screens or quitting mid-request.
- [ ] **Rate limit handling** — Detect GitHub rate limit responses and show a countdown or retry.
- [ ] **Caching** — Cache PR list and detail data to avoid redundant API calls when navigating back and forth.
- [ ] **Config file** — Support a `~/.config/gh-tui/config.yaml` for defaults (merge method, default filter, token, etc.).
- [ ] **Tests** — Unit tests for `repo/detect.go` parsing, mock-based tests for GitHub client, model update tests for TUI logic.
- [ ] **Logging** — Add debug logging to a file (`--debug` flag) for troubleshooting API issues.

## Agent Integration (Future)

- [ ] **Agent context interface** — Define a clean interface/struct that captures the current TUI state (repo, selected PR, filtered list, visible diff) for agent consumption.
- [ ] **Agent command palette** — A keybind (e.g. `:` or `Ctrl+A`) that opens a prompt to send natural language commands to an agent.
- [ ] **Agent scopes** — Let the user scope the agent to: current project, current PR list (filtered), or a single PR.
- [ ] **Agent actions** — Define the set of actions an agent can take: summarize PR, generate review, suggest merge strategy, batch operations on filtered PRs.
- [ ] **Agent output panel** — A split pane or overlay that shows agent responses streaming in real-time.
- [ ] **Tool definitions for agent** — Expose GitHub client operations as tool calls the agent can invoke (list PRs, read diff, post comment, merge, etc.).
