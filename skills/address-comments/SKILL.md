---
name: address-comments
description: Help address GitHub PR review comments. Use when the user mentions PR reviews, code review feedback, reviewer comments, or needs to resolve review threads.
allowed-tools: Bash(gh pr-comments:*), Bash(gh issue create:*), Read, Edit, Grep, Glob, mcp__conductor__AskUserQuestion
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
3. **Ask User**: Use `AskUserQuestion` to let the user choose how to handle each comment:
   - **Fix it now** - Make code changes, reply, and resolve
   - **Fix it later** - Create a GitHub issue to track, reply with link, optionally resolve
   - **No changes needed** - Reply explaining why, then resolve
4. **Execute**: Based on user choice, take the appropriate action
5. **Reply**: **IMPORTANT** - Always reply before resolving. Explain what was done or why no changes were made
6. **Resolve**: Use `gh pr-comments resolve <id>` to mark as resolved

### Handling "Fix it now"

When the user wants to address the comment immediately:
```bash
# 1. Make the code changes using Edit tool

# 2. Reply explaining what was fixed
gh pr-comments reply <id> --body "Fixed: <brief description of what was changed>"

# 3. Mark as resolved
gh pr-comments resolve <id>
```

### Handling "Fix it later"

When the user chooses to defer a fix:
```bash
# Create a GitHub issue to track the work
gh issue create --title "<concise-title>" --body "From PR review: <link-to-comment>

<description of what needs to be done>"

# Reply to the comment with the issue link
gh pr-comments reply <id> --body "Tracked in #<issue-number>"

# Optionally resolve the comment
gh pr-comments resolve <id>
```

### Handling "No changes needed"

When the suggested change is not appropriate:
```bash
# Reply with explanation
gh pr-comments reply <id> --body "No changes needed: <clear explanation>"

# Resolve the comment
gh pr-comments resolve <id>
```

## Best Practices

- Always ask the user before deciding how to handle a comment
- Group related comments by file and address them together
- Read the surrounding code context, not just the commented line
- If a comment is a question, reply instead of making code changes
- Be respectful and professional in all replies
- Provide clear explanations for decisions
- Use `--json` flag for programmatic parsing
- After resolving all comments in a review, use `cleanup` to minimize the review
