# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Install

```bash
# Build and reinstall extension locally
go build -o gh-pr-comments . && gh extension remove pr-comments && gh extension install .

# Build only
go build

# Install from source (first time)
gh extension install .
```

## Architecture

This is a GitHub CLI extension (`gh pr-comments`) built with Go and Cobra. It provides structured access to PR reviews and comments that the standard `gh` CLI lacks.

### Package Structure

- `cmd/` - Cobra commands (list, reviews, tree, view, reply)
- `internal/github/` - GitHub API client and types

### Key Components

**client.go** - GitHub API abstraction using both REST and GraphQL:
- REST API for reviews, review comments, issue comments
- GraphQL API specifically for fetching `isResolved` status (not available in REST)
- PR reference parsing (URLs, owner/repo/number, just number, or auto-detect from branch)

**types.go** - Data structures:
- `Review` - PR reviews with state (APPROVED, CHANGES_REQUESTED, etc.)
- `ReviewComment` - Inline code comments on specific file/line
- `IssueComment` - General PR comments (not attached to code)

### Comment Types

| Type | Description | Has file/line |
|------|-------------|---------------|
| Review | Review with state | No |
| ReviewComment | Inline code comment | Yes |
| IssueComment | General PR comment | No |

### Filtering Behavior

- Resolved comments are hidden by default in `list` and `tree` commands
- Use `--all` to show resolved, or `--resolved=true` to show only resolved
- `--outdated` filters by whether the code has changed since the comment

## AskUserQuestion Tool Usage

**CRITICAL**: When working with PR comments or making changes to the codebase, you MUST use the `AskUserQuestion` tool to get explicit user confirmation before:

- Making code changes based on review feedback
- Resolving or hiding comments
- Replying to reviewers
- Creating issues to track deferred work
- Any action that modifies files or PR state

**Never assume what the user wants.** Even if a fix seems obvious, always present your analysis and proposed solution, then use `AskUserQuestion` to let the user choose how to proceed. This ensures the user maintains full control over their codebase and PR workflow.
