package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/STRRL/gh-pr-comments/internal/github"
	"github.com/spf13/cobra"
)

type CleanupInfo struct {
	ReviewID   int64  `json:"review_id"`
	ReviewerID string `json:"reviewer"`
	Minimized  bool   `json:"minimized"`
	Error      string `json:"error,omitempty"`
}

var (
	resolvePR         string
	resolveJsonOutput bool
)

var resolveCmd = &cobra.Command{
	Use:               "resolve <comment-id> [comment-id...]",
	Short:             "Resolve review threads",
	ValidArgsFunction: completeReviewCommentIDs,
	Long:              `Mark review comment threads as resolved.

The comment-id(s) can be found from the 'list', 'view', or 'tree' command output.
Each comment belongs to a review thread, and this command resolves the
entire thread containing the specified comment.

After resolving, this command automatically minimizes (hides) any reviews where
all inline comments are now resolved. This helps reduce noise in the PR timeline.

Examples:
  # Resolve a single thread
  gh pr-comments resolve 2621968472

  # Resolve multiple threads
  gh pr-comments resolve 2621968472 2621968473 2621968474

  # Specify PR explicitly
  gh pr-comments resolve 2621968472 --pr owner/repo/99

  # Get JSON output
  gh pr-comments resolve 2621968472 --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runResolve,
}

func init() {
	resolveCmd.Flags().StringVar(&resolvePR, "pr", "", "PR reference (e.g., owner/repo/123 or just 123)")
	resolveCmd.Flags().BoolVar(&resolveJsonOutput, "json", false, "Output in JSON format")
	rootCmd.AddCommand(resolveCmd)
}

type ResolveResult struct {
	CommentID int64  `json:"comment_id"`
	ThreadID  string `json:"thread_id"`
	Action    string `json:"action"`
	Success   bool   `json:"success"`
	Skipped   bool   `json:"skipped,omitempty"`
	Error     string `json:"error,omitempty"`
}

func runResolve(cmd *cobra.Command, args []string) error {
	client, err := github.NewClient()
	if err != nil {
		return err
	}

	var commentIDs []int64
	for _, arg := range args {
		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid comment ID: %s", arg)
		}
		commentIDs = append(commentIDs, id)
	}

	var prArgs []string
	if resolvePR != "" {
		prArgs = []string{resolvePR}
	}

	prRef, err := client.ResolvePRReference(prArgs)
	if err != nil {
		return fmt.Errorf("could not determine PR: %w\nPlease specify a PR with --pr or run from a branch with an associated PR", err)
	}

	threads, err := client.GetReviewThreads(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return fmt.Errorf("get review threads: %w", err)
	}

	commentToThread := make(map[int64]string)
	for _, t := range threads {
		for _, cid := range t.CommentIDs {
			commentToThread[cid] = t.ID
		}
	}

	action := "resolved"

	var results []ResolveResult
	processedThreads := make(map[string]bool)

	for _, commentID := range commentIDs {
		threadID, ok := commentToThread[commentID]
		if !ok {
			results = append(results, ResolveResult{
				CommentID: commentID,
				Action:    action,
				Success:   false,
				Error:     "comment not found in any review thread",
			})
			continue
		}

		if processedThreads[threadID] {
			results = append(results, ResolveResult{
				CommentID: commentID,
				ThreadID:  threadID,
				Action:    action,
				Success:   true,
				Skipped:   true,
			})
			continue
		}
		processedThreads[threadID] = true

		err := client.ResolveThread(threadID)

		result := ResolveResult{
			CommentID: commentID,
			ThreadID:  threadID,
			Action:    action,
			Success:   err == nil,
		}
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}

	var cleanupResults []CleanupInfo
	if !resolveUndo {
		cleanupResults = performAutoCleanup(client, prRef)
	}

	if resolveJsonOutput {
		output := struct {
			Results []ResolveResult `json:"results"`
			Cleanup []CleanupInfo   `json:"cleanup,omitempty"`
		}{
			Results: results,
			Cleanup: cleanupResults,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	printResolveResults(results, action, cleanupResults)
	return nil
}

func performAutoCleanup(client *github.Client, prRef *github.PRReference) []CleanupInfo {
	reviews, err := client.GetReviews(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return nil
	}

	reviewComments, err := client.GetReviewComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return nil
	}

	commentsByReview := make(map[int64][]github.ReviewComment)
	for _, c := range reviewComments {
		commentsByReview[c.PullRequestReviewID] = append(commentsByReview[c.PullRequestReviewID], c)
	}

	var cleanupResults []CleanupInfo

	for _, r := range reviews {
		comments := commentsByReview[r.ID]
		if len(comments) == 0 {
			continue
		}

		allResolved := true
		for _, c := range comments {
			if !c.IsResolved {
				allResolved = false
				break
			}
		}

		if !allResolved {
			continue
		}

		reviewerID := "unknown"
		if r.User.Login != "" {
			reviewerID = r.User.Login
		}

		info := CleanupInfo{
			ReviewID:   r.ID,
			ReviewerID: reviewerID,
		}

		err := client.MinimizeComment(r.NodeID, github.ClassifierResolved)
		if err != nil {
			info.Minimized = false
			info.Error = err.Error()
		} else {
			info.Minimized = true
		}

		cleanupResults = append(cleanupResults, info)
	}

	return cleanupResults
}

func printResolveResults(results []ResolveResult, action string, cleanupResults []CleanupInfo) {
	successCount := 0
	skippedCount := 0
	failCount := 0

	for _, r := range results {
		if r.Skipped {
			skippedCount++
			fmt.Printf("Skipped comment %d (thread already processed)\n", r.CommentID)
		} else if r.Success {
			successCount++
			fmt.Printf("Thread %s for comment %d\n", action, r.CommentID)
		} else {
			failCount++
			fmt.Fprintf(os.Stderr, "Failed to resolve thread for comment %d: %s\n",
				r.CommentID, r.Error)
		}
	}

	fmt.Println(strings.Repeat("─", 40))
	if successCount > 0 {
		fmt.Printf("Done: %d thread(s) %s\n", successCount, action)
	}
	if skippedCount > 0 {
		fmt.Printf("Skipped: %d comment(s) (same thread)\n", skippedCount)
	}
	if failCount > 0 {
		fmt.Printf("Failed: %d thread(s)\n", failCount)
	}

	if len(cleanupResults) > 0 {
		fmt.Println()
		fmt.Println("Auto-cleanup:")
		minimizedCount := 0
		for _, c := range cleanupResults {
			if c.Minimized {
				minimizedCount++
				fmt.Printf("  Minimized review %d by @%s\n", c.ReviewID, c.ReviewerID)
			} else {
				fmt.Fprintf(os.Stderr, "  Failed to minimize review %d: %s\n", c.ReviewID, c.Error)
			}
		}
		if minimizedCount > 0 {
			fmt.Printf("Cleaned up: %d review(s) minimized\n", minimizedCount)
		}
	}
}
