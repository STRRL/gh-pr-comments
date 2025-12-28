package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/STRRL/gh-pr-comments/internal/github"
	"github.com/spf13/cobra"
)

var viewJsonOutput bool

var viewCmd = &cobra.Command{
	Use:               "view <id>",
	Aliases:           []string{"show"},
	Short:             "View full content of a review comment, review, or issue comment",
	Long: `View the full content of an item by its ID.

Automatically detects the type (review comment, review, or issue comment).

The ID can be found from the 'list', 'reviews', or 'tree' command output.

Examples:
  gh pr-comments view 2621968472
  gh pr-comments show 3581523351
  gh pr-comments view 2621968472 --json`,
	Args:              cobra.ExactArgs(1),
	RunE:              runView,
	ValidArgsFunction: completeCommentIDs,
}

func init() {
	viewCmd.Flags().BoolVar(&viewJsonOutput, "json", false, "Output in JSON format")
	rootCmd.AddCommand(viewCmd)
}

func runView(cmd *cobra.Command, args []string) error {
	client, err := github.NewClient()
	if err != nil {
		return err
	}

	id := args[0]

	prRef, err := client.ResolvePRReference(nil)
	if err != nil {
		return fmt.Errorf("could not determine PR: %w\nPlease run this command from a branch with an associated PR", err)
	}

	if found, err := tryViewReviewComment(client, prRef, id); err != nil {
		return err
	} else if found {
		return nil
	}

	if found, err := tryViewReview(client, prRef, id); err != nil {
		return err
	} else if found {
		return nil
	}

	if found, err := tryViewIssueComment(client, prRef, id); err != nil {
		return err
	} else if found {
		return nil
	}

	return fmt.Errorf("item with ID %s not found in PR %d (searched review comments, reviews, and issue comments)", id, prRef.Number)
}

func tryViewReviewComment(client *github.Client, prRef *github.PRReference, commentID string) (bool, error) {
	comments, err := client.GetReviewComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return false, err
	}

	for _, c := range comments {
		if fmt.Sprintf("%d", c.ID) == commentID {
			if viewJsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return true, enc.Encode(c)
			}

			printReviewCommentDetail(c)
			return true, nil
		}
	}

	return false, nil
}

func tryViewReview(client *github.Client, prRef *github.PRReference, reviewID string) (bool, error) {
	reviews, err := client.GetReviews(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return false, err
	}

	for _, r := range reviews {
		if fmt.Sprintf("%d", r.ID) == reviewID {
			if viewJsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return true, enc.Encode(r)
			}

			printReviewDetail(r)
			return true, nil
		}
	}

	return false, nil
}

func tryViewIssueComment(client *github.Client, prRef *github.PRReference, commentID string) (bool, error) {
	comments, err := client.GetIssueComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return false, err
	}

	for _, c := range comments {
		if fmt.Sprintf("%d", c.ID) == commentID {
			if viewJsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return true, enc.Encode(c)
			}

			printIssueCommentDetail(c)
			return true, nil
		}
	}

	return false, nil
}

func printReviewCommentDetail(c github.ReviewComment) {
	fmt.Printf("Review Comment %d\n", c.ID)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("File:      %s", c.Path)
	if c.OriginalLine != nil {
		fmt.Printf(":%d", *c.OriginalLine)
	}
	fmt.Println()
	fmt.Printf("Author:    %s\n", c.User.Login)
	fmt.Printf("Created:   %s\n", c.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Review ID: %d\n", c.PullRequestReviewID)
	fmt.Printf("Outdated:  %v\n", c.IsOutdated())
	fmt.Printf("Resolved:  %v\n", c.IsResolved)
	fmt.Printf("URL:       %s\n", c.HTMLURL)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()
	fmt.Println(c.Body)
	fmt.Println()

	if c.DiffHunk != "" {
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println("Diff context:")
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println(c.DiffHunk)
	}
}

func printReviewDetail(r github.Review) {
	fmt.Printf("Review %d\n", r.ID)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Author:    %s\n", r.User.Login)
	fmt.Printf("State:     %s\n", r.State)
	if !r.SubmittedAt.IsZero() {
		fmt.Printf("Submitted: %s\n", r.SubmittedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("URL:       %s\n", r.HTMLURL)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()
	if r.Body != "" {
		fmt.Println(r.Body)
	} else {
		fmt.Println("(no body)")
	}
	fmt.Println()
}

func printIssueCommentDetail(c github.IssueComment) {
	fmt.Printf("Issue Comment %d\n", c.ID)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Author:    %s\n", c.User.Login)
	fmt.Printf("Created:   %s\n", c.CreatedAt.Format("2006-01-02 15:04:05"))
	if !c.UpdatedAt.IsZero() && c.UpdatedAt != c.CreatedAt {
		fmt.Printf("Updated:   %s\n", c.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("URL:       %s\n", c.HTMLURL)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()
	fmt.Println(c.Body)
	fmt.Println()
}
