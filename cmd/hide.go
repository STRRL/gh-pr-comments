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

var (
	hideReason     string
	hideUndo       bool
	hideAuthor     string
	hidePR         string
	hideJsonOutput bool
	hideDryRun     bool
)

var hideCmd = &cobra.Command{
	Use:               "hide [comment-id]",
	Short:             "Hide (minimize) PR comments",
	ValidArgsFunction: completeCommentIDs,
	Long:              `Hide PR comments by marking them with a reason.

When a comment ID is provided, hides that specific comment.
When no ID is provided, uses filters to select comments for batch hiding.

Reasons (--reason):
  abuse     - Abusive or harmful content
  duplicate - Duplicate comment
  off-topic - Not relevant to the discussion
  outdated  - Information no longer applies
  resolved  - Issue has been addressed (default)
  spam      - Spam content

Examples:
  # Hide a single comment (default reason: resolved)
  gh pr-comments hide 2621968472

  # Hide with specific reason
  gh pr-comments hide 2621968472 --reason outdated

  # Hide all comments by a specific author
  gh pr-comments hide --author "claude[bot]" --reason outdated

  # Unhide a comment
  gh pr-comments hide 2621968472 --undo

  # Dry run to see what would be hidden
  gh pr-comments hide --author "bot" --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHide,
}

func init() {
	hideCmd.Flags().StringVar(&hideReason, "reason", "resolved",
		"Reason for hiding (abuse, duplicate, off-topic, outdated, resolved, spam)")
	hideCmd.Flags().BoolVar(&hideUndo, "undo", false,
		"Unhide the comment instead")
	hideCmd.Flags().StringVar(&hideAuthor, "author", "",
		"Filter by comment author for batch operations")
	hideCmd.Flags().StringVar(&hidePR, "pr", "",
		"PR reference (e.g., owner/repo/123)")
	hideCmd.Flags().BoolVar(&hideJsonOutput, "json", false,
		"Output in JSON format")
	hideCmd.Flags().BoolVar(&hideDryRun, "dry-run", false,
		"Show what would be hidden without actually doing it")

	rootCmd.AddCommand(hideCmd)
}

type hideResult struct {
	ID      int64  `json:"id"`
	NodeID  string `json:"node_id"`
	Type    string `json:"type"`
	Author  string `json:"author"`
	Success bool   `json:"success"`
	Action  string `json:"action"`
	Error   string `json:"error,omitempty"`
}

func runHide(cmd *cobra.Command, args []string) error {
	client, err := github.NewClient()
	if err != nil {
		return err
	}

	var prArgs []string
	if hidePR != "" {
		prArgs = []string{hidePR}
	}

	prRef, err := client.ResolvePRReference(prArgs)
	if err != nil {
		return fmt.Errorf("could not determine PR: %w\nPlease specify a PR with --pr or run from a branch with an associated PR", err)
	}

	var classifier github.CommentClassifier
	if !hideUndo {
		classifier, err = github.ParseClassifier(hideReason)
		if err != nil {
			return err
		}
	}

	if len(args) > 0 {
		return hideSingleComment(client, prRef, args[0], classifier)
	}

	if hideAuthor == "" {
		return fmt.Errorf("batch hide requires --author filter\nProvide a comment ID for single comment, or use --author for batch operations")
	}

	return hideBatch(client, prRef, classifier)
}

func hideSingleComment(client *github.Client, prRef *github.PRReference, commentIDStr string, classifier github.CommentClassifier) error {
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid comment ID: %s", commentIDStr)
	}

	nodeID, commentType, author, err := findCommentNodeID(client, prRef, commentID)
	if err != nil {
		return err
	}

	result := hideResult{
		ID:     commentID,
		NodeID: nodeID,
		Type:   commentType,
		Author: author,
	}

	if hideDryRun {
		result.Action = "would_hide"
		if hideUndo {
			result.Action = "would_unhide"
		}
		result.Success = true
		return outputResult(result)
	}

	if hideUndo {
		err = client.UnminimizeComment(nodeID)
		result.Action = "unhide"
	} else {
		err = client.MinimizeComment(nodeID, classifier)
		result.Action = "hide"
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
	}

	return outputResult(result)
}

func hideBatch(client *github.Client, prRef *github.PRReference, classifier github.CommentClassifier) error {
	reviewComments, err := client.GetReviewComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return err
	}

	issueComments, err := client.GetIssueComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return err
	}

	var targets []hideResult

	authorFilter := strings.ToLower(hideAuthor)
	for _, c := range reviewComments {
		if strings.ToLower(c.User.Login) == authorFilter {
			targets = append(targets, hideResult{
				ID:     c.ID,
				NodeID: c.NodeID,
				Type:   "review",
				Author: c.User.Login,
			})
		}
	}

	for _, c := range issueComments {
		if strings.ToLower(c.User.Login) == authorFilter {
			targets = append(targets, hideResult{
				ID:     c.ID,
				NodeID: c.NodeID,
				Type:   "issue",
				Author: c.User.Login,
			})
		}
	}

	if len(targets) == 0 {
		if hideJsonOutput {
			return json.NewEncoder(os.Stdout).Encode([]hideResult{})
		}
		fmt.Printf("No comments found by author '%s'\n", hideAuthor)
		return nil
	}

	var results []hideResult
	for _, t := range targets {
		result := t

		if hideDryRun {
			result.Action = "would_hide"
			if hideUndo {
				result.Action = "would_unhide"
			}
			result.Success = true
			results = append(results, result)
			continue
		}

		var opErr error
		if hideUndo {
			opErr = client.UnminimizeComment(t.NodeID)
			result.Action = "unhide"
		} else {
			opErr = client.MinimizeComment(t.NodeID, classifier)
			result.Action = "hide"
		}

		if opErr != nil {
			result.Success = false
			result.Error = opErr.Error()
		} else {
			result.Success = true
		}
		results = append(results, result)
	}

	return outputResults(results)
}

func findCommentNodeID(client *github.Client, prRef *github.PRReference, commentID int64) (nodeID, commentType, author string, err error) {
	reviewComments, err := client.GetReviewComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return "", "", "", err
	}

	for _, c := range reviewComments {
		if c.ID == commentID {
			return c.NodeID, "review", c.User.Login, nil
		}
	}

	issueComments, err := client.GetIssueComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return "", "", "", err
	}

	for _, c := range issueComments {
		if c.ID == commentID {
			return c.NodeID, "issue", c.User.Login, nil
		}
	}

	return "", "", "", fmt.Errorf("comment with ID %d not found in PR %d", commentID, prRef.Number)
}

func getActionDisplayString(action string) string {
	switch action {
	case "unhide":
		return "Unhidden"
	case "would_hide":
		return "Would hide"
	case "would_unhide":
		return "Would unhide"
	default:
		return "Hidden"
	}
}

func outputResult(result hideResult) error {
	if hideJsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	if result.Success {
		fmt.Printf("%s comment %d (%s by %s)\n", getActionDisplayString(result.Action), result.ID, result.Type, result.Author)
	} else {
		fmt.Printf("Failed to process comment %d: %s\n", result.ID, result.Error)
	}
	return nil
}

func outputResults(results []hideResult) error {
	if hideJsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	successCount := 0
	failCount := 0

	for _, r := range results {
		if r.Success {
			successCount++
			fmt.Printf("%s comment %d (%s by %s)\n", getActionDisplayString(r.Action), r.ID, r.Type, r.Author)
		} else {
			failCount++
			fmt.Printf("Failed: comment %d - %s\n", r.ID, r.Error)
		}
	}

	fmt.Println(strings.Repeat("─", 40))
	if hideDryRun {
		fmt.Printf("Dry run: %d comment(s) would be processed\n", len(results))
	} else {
		fmt.Printf("Processed: %d succeeded, %d failed\n", successCount, failCount)
	}

	return nil
}
