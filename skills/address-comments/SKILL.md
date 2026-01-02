---
name: address-comments
description: Help address GitHub PR review comments. Use when the user mentions PR reviews, code review feedback, reviewer comments, or needs to resolve review threads.
allowed-tools: Bash(gh pr-comments:*), Read, Edit, Grep, Glob
---

# Addressing PR Review Comments

This skill helps you work with GitHub PR review comments using the `gh-pr-comments` CLI extension.

## Comment Types

There are two types of comments on a PR:

1. **Review Comments** (inline): Attached to specific lines of code
   - Have a file path and line number
   - Can be marked as resolved
   - May be "outdated" if the code has changed

2. **Issue Comments** (general): Not attached to code
   - Appear in the PR conversation
   - Cannot be resolved (only hidden)

## Available Commands

```bash
# List unresolved comments (resolved hidden by default)
gh pr-comments list

# List all comments including resolved
gh pr-comments list --all

# Show hierarchical tree view
gh pr-comments tree

# View full details of a comment
gh pr-comments view <comment-id>

# Reply to a comment
gh pr-comments reply <comment-id> --body "message"

# Mark comments as resolved
gh pr-comments resolve <comment-id> [comment-id...]

# Hide (minimize) comments
gh pr-comments hide <comment-id> --reason resolved

# Clean up reviews with all comments resolved
gh pr-comments cleanup
```

## Workflow for Addressing Comments

1. **List**: Start with `gh pr-comments list --json` to see all unresolved comments
2. **Understand**: Use `gh pr-comments view <id>` to see full context including diff
3. **Fix**: Make the necessary code changes
4. **Resolve**: Use `gh pr-comments resolve <id>` to mark as resolved

## Best Practices

- Group related comments by file and address them together
- Read the surrounding code context, not just the commented line
- If a comment is a question, reply instead of making code changes
- Use `--json` flag for programmatic parsing
- After resolving all comments in a review, use `cleanup` to minimize the review
