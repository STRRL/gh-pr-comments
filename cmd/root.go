package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh-pr-comments",
	Short: "Structured access to PR reviews and review comments",
	Long: `A GitHub CLI extension for structured access to Pull Request reviews and review comments.

Unlike the standard gh CLI, this extension provides:
  - List all reviews with their states
  - List review comments grouped by review
  - Filter by outdated and resolved status (resolved hidden by default)
  - Hierarchical tree view of all comments`,
	Example: `  # List all reviews on current branch's PR
  gh pr-comments reviews

  # List unresolved review comments (default)
  gh pr-comments list

  # Include resolved comments
  gh pr-comments list --all

  # List only resolved comments
  gh pr-comments list --resolved=true

  # List only outdated comments
  gh pr-comments list --outdated=true

  # Filter comments by review ID
  gh pr-comments list --review-id=3581523351

  # Show tree view (resolved hidden by default)
  gh pr-comments tree
  gh pr-comments tree --all

  # View full content of any item (auto-detects type)
  gh pr-comments view 2621968472
  gh pr-comments show 3581523351

  # Resolve review threads by comment ID
  gh pr-comments resolve 2621968472
  gh pr-comments resolve 2621968472 2621968473 2621968474
  gh pr-comments resolve 2621968472 --pr owner/repo/99

  # Clean up reviews with all comments resolved
  gh pr-comments cleanup --dry-run
  gh pr-comments cleanup
  gh pr-comments cleanup --review-id 3581523351
  gh pr-comments cleanup https://github.com/owner/repo/pull/123

  # Use with explicit PR reference
  gh pr-comments reviews owner/repo/123
  gh pr-comments list https://github.com/owner/repo/pull/123

  # Output as JSON
  gh pr-comments list --json
  gh pr-comments tree --json`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(reviewsCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(treeCmd)
}
