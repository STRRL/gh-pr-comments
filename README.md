# gh-pr-comments

A GitHub CLI extension for structured access to Pull Request reviews and review comments.

## Problem

The `gh` CLI doesn't provide structured access to PR review comments:

```bash
# These don't exist or are limited in gh CLI:
gh pr review list              # Can't list all reviews with states
gh pr comments --by-review     # Can't group comments by review
gh pr comments --unresolved    # Can't filter unresolved comments
gh pr comments --outdated      # Can't filter outdated comments
```

## Solution

`gh pr-comments` fills this gap by providing structured access to:

- **Pull Request Reviews** - List all reviews with their states (APPROVED, CHANGES_REQUESTED, COMMENTED, etc.)
- **Review Comments** - List inline code comments, optionally filtered by review
- **Filtering** - Filter by outdated status, resolved status (resolved hidden by default)
- **Hierarchical View** - See the tree structure of reviews and their comments

## Installation

```bash
gh extension install STRRL/gh-pr-comments
```

## Usage

All commands support automatic PR detection - if no PR reference is given, the extension finds the PR for the current branch.

### List Reviews

List all reviews on a pull request:

```bash
gh pr-comments reviews                    # auto-detect PR for current branch
gh pr-comments reviews https://github.com/owner/repo/pull/123
gh pr-comments reviews owner/repo/123
gh pr-comments reviews 123                # in a repo context
```

Output:
```
ID            STATE              AUTHOR                      SUBMITTED
3581523351    COMMENTED          copilot[bot]                2025-12-16
3581000000    APPROVED           reviewer                    2025-12-15
3580000000    CHANGES_REQUESTED  another-reviewer            2025-12-14
```

### List Review Comments

List all review comments on a pull request (resolved comments hidden by default):

```bash
gh pr-comments list                       # auto-detect PR for current branch
gh pr-comments list https://github.com/owner/repo/pull/123
```

Output:
```
ID            FILE                        LINE  OUTDATED  RESOLVED  REVIEW ID     AUTHOR      BODY
2621968472    pkg/deviceflow/store.go     109   true      false     3581523351    copilot     Setting the status...
2621968513    cmd/wonder/worker.go        258   false     false     3581523351    copilot     Network or decoding...
```

Show all comments including resolved:

```bash
gh pr-comments list --all                 # show all comments
gh pr-comments list --resolved=true       # only resolved comments
gh pr-comments list --resolved=false      # only unresolved (default behavior)
```

Filter by review:

```bash
gh pr-comments list owner/repo/123 --review-id=3581523351
```

Filter outdated comments:

```bash
gh pr-comments list owner/repo/123 --outdated
gh pr-comments list owner/repo/123 --outdated=false  # only current comments
```

### View Full Content

View the full content of any item (auto-detects whether it's a review comment, review, or issue comment):

```bash
gh pr-comments view 2621968472            # view any item by ID
gh pr-comments show 3581523351            # 'show' is an alias for 'view'
gh pr-comments view 2621968472 --json     # output as JSON
```

Output:
```
Review Comment 2621968472
────────────────────────────────────────────────────────────
File:      pkg/deviceflow/store.go:109
Author:    Copilot
Created:   2025-12-16 06:31:06
Review ID: 3581523351
Outdated:  true
Resolved:  false
URL:       https://github.com/STRRL/wonder-mesh-net/pull/11#discussion_r2621968472
────────────────────────────────────────────────────────────

Setting the status on a shared object without holding a write lock can lead to race conditions...

────────────────────────────────────────────────────────────
Diff context:
────────────────────────────────────────────────────────────
@@ -0,0 +1,195 @@
+package deviceflow
...
```

### Tree View

Show hierarchical view of reviews and their comments (resolved comments hidden by default):

```bash
gh pr-comments tree                       # auto-detect PR for current branch
gh pr-comments tree --all                 # include resolved comments
gh pr-comments tree https://github.com/owner/repo/pull/123
```

Output (sorted by time):
```
PR #123: Add OAuth device flow
│
├── Review 3580000000 by another-reviewer (CHANGES_REQUESTED) - 2025-12-14
│   └── (no inline comments)
│
├── Review 3581000000 by reviewer (APPROVED) - 2025-12-15
│   └── (no inline comments)
│
├── Review 3581523351 by copilot[bot] (COMMENTED) - 2025-12-16
│   ├── [2621968472] pkg/deviceflow/store.go:109 (outdated)
│   │   └── Setting the status on a shared object without holding...
│   ├── [2621968513] cmd/wonder/worker.go:258
│   │   └── Network or decoding errors during polling are silently...
│   └── [2621968599] pkg/deviceflow/store.go:193
│       └── The modulo operation with cryptographically random bytes...
│
└── Issue Comments (2)
    ├── 3659032896 by claude[bot] - 2025-12-16
    └── 3659064743 by claude[bot] - 2025-12-16
```

With `--all`, resolved comments are shown with a `(resolved)` tag.

### Output Formats

All commands support multiple output formats:

```bash
gh pr-comments reviews owner/repo/123 --json
gh pr-comments list owner/repo/123 --json
gh pr-comments tree owner/repo/123 --json
```

## GitHub API Types Reference

This extension works with these GitHub API types:

| Type | API Endpoint | URL Fragment |
|------|--------------|--------------|
| Issue Comments | `/repos/{owner}/{repo}/issues/{pr}/comments` | `#issuecomment-{id}` |
| Pull Request Reviews | `/repos/{owner}/{repo}/pulls/{pr}/reviews` | `#pullrequestreview-{id}` |
| Review Comments | `/repos/{owner}/{repo}/pulls/{pr}/comments` | `#discussion_r{id}` |

### Review States

| State | Description |
|-------|-------------|
| `PENDING` | Created but not yet submitted |
| `APPROVED` | Changes approved |
| `CHANGES_REQUESTED` | Changes requested |
| `COMMENTED` | Feedback without approval/rejection |
| `DISMISSED` | Review dismissed |

### Outdated Detection

A review comment is considered **outdated** when `position` or `line` is `null`, indicating the code has changed since the comment was made.

### Resolved Detection

Resolved status is fetched via the GraphQL API (not available in REST). Comments are grouped by review threads, and a thread's `isResolved` status applies to all comments in that thread. By default, resolved comments are hidden in `list` and `tree` commands.

## Shell Completion

Shell completion is available for bash, zsh, fish, and powershell.

### Zsh

```bash
# Generate and install completion
gh pr-comments completion zsh > "${fpath[1]}/_gh-pr-comments"

# Make sure completion is enabled in ~/.zshrc:
# autoload -U compinit; compinit

# Restart your shell or run:
source ~/.zshrc
```

### Bash

```bash
# For current session
source <(gh pr-comments completion bash)

# For all sessions (Linux)
gh pr-comments completion bash > /etc/bash_completion.d/gh-pr-comments

# For all sessions (macOS with Homebrew)
gh pr-comments completion bash > $(brew --prefix)/etc/bash_completion.d/gh-pr-comments
```

### Fish

```bash
gh pr-comments completion fish > ~/.config/fish/completions/gh-pr-comments.fish
```

### PowerShell

```powershell
gh pr-comments completion powershell | Out-String | Invoke-Expression
```

### Features

Completions include:
- Subcommands and flags
- Dynamic comment ID suggestions for `view` and `reply` commands (with content previews)
- Dynamic review ID suggestions for `--review-id` flag
- Flag value suggestions (e.g., `--type`, `--resolved`, `--outdated`)

## Development

```bash
# Build
go build

# Install locally
gh extension install .

# Run
gh pr-comments reviews owner/repo/123
```

## Claude Code Plugin

This project includes a Claude Code plugin for AI-assisted PR review comment handling.

### Installation

```bash
# Step 1: Install or upgrade the gh extension
gh extension install STRRL/gh-pr-comments --force
```

Then, install the Claude Code plugin:

```bash
# Step 2: Add the marketplace
claude plugin marketplace add STRRL/gh-pr-comments

# Step 3: Install the plugin
claude plugin install gh-pr-comments
```

### Usage

Use the `/gh-pr-comments:address-comments` slash command to:

1. List all unresolved PR comments
2. Get AI analysis and suggested fixes for each comment
3. Apply fixes with your confirmation
4. Mark comments as resolved

The plugin also includes a skill that automatically activates when you discuss PR reviews or code review feedback.

## License

MIT
