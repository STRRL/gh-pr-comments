package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/STRRL/gh-pr-comments/internal/github"
	"github.com/spf13/cobra"
)

var (
	cleanupDryRun     bool
	cleanupReviewID   int64
	cleanupJsonOutput bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup [pr-reference]",
	Short: "Minimize reviews with all comments resolved",
	Long: `Automatically minimize PR reviews when all their inline comments are resolved.

This helps reduce noise in the PR timeline by hiding reviews that have been
fully addressed. Only reviews with at least one inline comment where ALL
comments are resolved will be minimized.

Reviews are NOT minimized if they:
- Have no inline comments (nothing to "clean up")
- Have any unresolved comments

If no PR reference is given, finds the PR for the current branch.

PR reference can be:
  - Full URL: https://github.com/owner/repo/pull/123
  - Short form: owner/repo/123
  - Just number: 123 (when in a repo context)
  - Omitted: uses current branch's PR

Examples:
  # Preview what would be cleaned up
  gh pr-comments cleanup --dry-run

  # Clean up all eligible reviews
  gh pr-comments cleanup

  # Clean up a specific review only
  gh pr-comments cleanup --review-id 12345678

  # Get JSON output
  gh pr-comments cleanup --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCleanup,
}

func init() {
	cleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "Preview which reviews would be minimized without making changes")
	cleanupCmd.Flags().Int64Var(&cleanupReviewID, "review-id", 0, "Only process a specific review ID")
	cleanupCmd.Flags().BoolVar(&cleanupJsonOutput, "json", false, "Output in JSON format")
	rootCmd.AddCommand(cleanupCmd)
}

type ReviewCleanupCandidate struct {
	Review        github.Review `json:"review"`
	TotalCount    int           `json:"total_comments"`
	ResolvedCount int           `json:"resolved_comments"`
	CanMinimize   bool          `json:"can_minimize"`
	Reason        string        `json:"reason,omitempty"`
}

type CleanupOutput struct {
	PRNumber  int                      `json:"pr_number"`
	DryRun    bool                     `json:"dry_run"`
	Minimized []ReviewCleanupCandidate `json:"minimized"`
	Failed    []ReviewCleanupCandidate `json:"failed,omitempty"`
	Skipped   []ReviewCleanupCandidate `json:"skipped"`
}

func runCleanup(cmd *cobra.Command, args []string) error {
	client, err := github.NewClient()
	if err != nil {
		return err
	}

	prRef, err := client.ResolvePRReference(args)
	if err != nil {
		return err
	}

	reviews, err := client.GetReviews(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return err
	}

	reviewComments, err := client.GetReviewComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return err
	}

	candidates := identifyCleanupCandidates(reviews, reviewComments)

	if cleanupReviewID != 0 {
		var filtered []ReviewCleanupCandidate
		for _, c := range candidates {
			if c.Review.ID == cleanupReviewID {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("review with ID %d not found", cleanupReviewID)
		}
		candidates = filtered
	}

	output := CleanupOutput{
		PRNumber: prRef.Number,
		DryRun:   cleanupDryRun,
	}

	for _, c := range candidates {
		if c.CanMinimize {
			output.Minimized = append(output.Minimized, c)
		} else {
			output.Skipped = append(output.Skipped, c)
		}
	}

	if !cleanupDryRun {
		var successful []ReviewCleanupCandidate
		for _, c := range output.Minimized {
			err := client.MinimizeComment(c.Review.NodeID, "RESOLVED")
			if err != nil {
				c.CanMinimize = false
				c.Reason = err.Error()
				output.Failed = append(output.Failed, c)
			} else {
				successful = append(successful, c)
			}
		}
		output.Minimized = successful
	}

	if cleanupJsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	printCleanupResults(output, cleanupDryRun)
	return nil
}

func identifyCleanupCandidates(reviews []github.Review, comments []github.ReviewComment) []ReviewCleanupCandidate {
	commentsByReview := make(map[int64][]github.ReviewComment)
	for _, c := range comments {
		commentsByReview[c.PullRequestReviewID] = append(commentsByReview[c.PullRequestReviewID], c)
	}

	var candidates []ReviewCleanupCandidate

	for _, r := range reviews {
		reviewComments := commentsByReview[r.ID]
		total := len(reviewComments)

		resolvedCount := 0
		for _, c := range reviewComments {
			if c.IsResolved {
				resolvedCount++
			}
		}

		candidate := ReviewCleanupCandidate{
			Review:        r,
			TotalCount:    total,
			ResolvedCount: resolvedCount,
		}

		if total == 0 {
			candidate.CanMinimize = false
			candidate.Reason = "no inline comments"
		} else if resolvedCount < total {
			candidate.CanMinimize = false
			candidate.Reason = "has unresolved comments"
		} else {
			candidate.CanMinimize = true
		}

		candidates = append(candidates, candidate)
	}

	return candidates
}

func printCleanupResults(output CleanupOutput, dryRun bool) {
	if dryRun {
		fmt.Printf("Analyzing PR #%d for cleanup...\n\n", output.PRNumber)
	} else {
		fmt.Printf("Cleaning up PR #%d...\n\n", output.PRNumber)
	}

	if len(output.Minimized) > 0 {
		if dryRun {
			fmt.Println("Reviews that would be minimized:")
		} else {
			fmt.Println("Minimized reviews:")
		}
		for _, c := range output.Minimized {
			submitted := ""
			if !c.Review.SubmittedAt.IsZero() {
				submitted = c.Review.SubmittedAt.Format("2006-01-02")
			}
			fmt.Printf("  Review %d by @%s (%s) - %s\n",
				c.Review.ID, c.Review.User.Login, c.Review.State, submitted)
			fmt.Printf("    %d/%d comments resolved\n", c.ResolvedCount, c.TotalCount)
		}
		fmt.Println()
	}

	if len(output.Skipped) > 0 {
		fmt.Println("Reviews not eligible for cleanup:")
		for _, c := range output.Skipped {
			submitted := ""
			if !c.Review.SubmittedAt.IsZero() {
				submitted = c.Review.SubmittedAt.Format("2006-01-02")
			}
			fmt.Printf("  Review %d by @%s (%s) - %s\n",
				c.Review.ID, c.Review.User.Login, c.Review.State, submitted)
			fmt.Printf("    %d/%d comments resolved (%s)\n", c.ResolvedCount, c.TotalCount, c.Reason)
		}
		fmt.Println()
	}

	if len(output.Failed) > 0 {
		fmt.Fprintln(os.Stderr, "Failed to minimize:")
		for _, c := range output.Failed {
			submitted := ""
			if !c.Review.SubmittedAt.IsZero() {
				submitted = c.Review.SubmittedAt.Format("2006-01-02")
			}
			fmt.Fprintf(os.Stderr, "  Review %d by @%s (%s) - %s\n",
				c.Review.ID, c.Review.User.Login, c.Review.State, submitted)
			fmt.Fprintf(os.Stderr, "    Error: %s\n", c.Reason)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("─", 40))
	if dryRun {
		fmt.Printf("Total: %d review(s) would be minimized\n", len(output.Minimized))
	} else {
		fmt.Printf("Done: %d review(s) minimized\n", len(output.Minimized))
		if len(output.Failed) > 0 {
			fmt.Printf("Failed: %d review(s)\n", len(output.Failed))
		}
	}
}
