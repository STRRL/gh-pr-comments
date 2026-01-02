---
allowed-tools: Bash(gh pr-comments:*), Read, Edit, Grep, Glob
description: Address unresolved PR review comments by analyzing feedback, fixing code, and marking as resolved
---

# Address PR Review Comments

You are helping the user address unresolved review comments on their pull request.

## Workflow

### Step 1: List unresolved comments

Run:
```bash
gh pr-comments list --json
```

If there are no unresolved comments, inform the user "No pending comments to address" and stop.

### Step 2: Group comments by file

Parse the JSON output and group review comments by their `file` field. Issue comments (without file) should be handled separately at the end.

### Step 3: Process each file group

For each file with comments:

1. **Fetch details**: For each comment ID in the group, run:
   ```bash
   gh pr-comments view <comment-id>
   ```

2. **Read the source file**: Use the Read tool to read the current state of the file.

3. **Analyze**: Understand what the reviewer is asking for. Consider:
   - Is it a bug fix request?
   - Is it a style/formatting suggestion?
   - Is it asking for additional error handling?
   - Is it requesting documentation or comments?
   - Is it a question that needs a reply rather than code change?

4. **Present to user**: Show the user:
   - The file name
   - Each comment with its line context
   - Your analysis of what changes are needed
   - Your proposed fix (if applicable)

5. **Get confirmation**: Ask the user if they want to:
   - Apply the suggested fix
   - Skip this comment
   - Handle it differently

6. **Execute changes**: If confirmed, use the Edit tool to make the changes.

7. **Mark as resolved**: Ask the user if they want to mark these comments as resolved. If yes:
   ```bash
   gh pr-comments resolve <comment-id-1> <comment-id-2> ...
   ```

### Step 4: Handle issue comments

For any issue comments (general PR comments not attached to specific code):
- Show the comment content
- Ask if the user wants to reply or just acknowledge

### Step 5: Summary

After processing all comments, provide a summary:
- Number of comments addressed
- Number of files modified
- Number of comments marked as resolved
- Any comments that were skipped

## Important Notes

- Always read the file before suggesting changes
- Consider the context around the commented line, not just the line itself
- If a comment is asking a question, suggest using `gh pr-comments reply` instead of code changes
- Be conservative with changes - only modify what the reviewer specifically requested
- If multiple comments on the same file are related, consider addressing them together
