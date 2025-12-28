package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/STRRL/gh-pr-comments/internal/github"
	"github.com/spf13/cobra"
)

var (
	listJsonOutput  bool
	listReviewID    int64
	listOutdated    string
	listResolved    string
	listAll         bool
	listCommentType string
)

var listCmd = &cobra.Command{
	Use:   "list [pr-reference]",
	Short: "List all comments on a pull request",
	Long: `List all comments on a pull request, including both review comments
(inline code comments) and issue comments (general PR comments).

By default, resolved review comments are hidden. Use --all to show all comments,
or --resolved=true to show only resolved comments.

If no PR reference is given, finds the PR for the current branch.

PR reference can be:
  - Full URL: https://github.com/owner/repo/pull/123
  - Short form: owner/repo/123
  - Just number: 123 (when in a repo context)
  - Omitted: uses current branch's PR

Examples:
  gh pr-comments list
  gh pr-comments list --all
  gh pr-comments list --type=review
  gh pr-comments list --type=issue
  gh pr-comments list --resolved=true
  gh pr-comments list https://github.com/owner/repo/pull/123
  gh pr-comments list owner/repo/123 --review-id=3581523351
  gh pr-comments list 123 --outdated
  gh pr-comments list 123 --outdated=false`,
	Args: cobra.MaximumNArgs(1),
	RunE: runList,
}

func init() {
	listCmd.Flags().BoolVar(&listJsonOutput, "json", false, "Output in JSON format")
	listCmd.Flags().Int64Var(&listReviewID, "review-id", 0, "Filter by review ID (review comments only)")
	listCmd.Flags().StringVar(&listOutdated, "outdated", "", "Filter by outdated status (true/false, review comments only)")
	listCmd.Flags().StringVar(&listResolved, "resolved", "", "Filter by resolved status (true/false, review comments only)")
	listCmd.Flags().BoolVar(&listAll, "all", false, "Show all comments including resolved")
	listCmd.Flags().StringVar(&listCommentType, "type", "", "Filter by comment type (review/issue)")

	listCmd.RegisterFlagCompletionFunc("review-id", completeReviewIDs)
	listCmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"review\tInline code comments", "issue\tGeneral PR comments"}, cobra.ShellCompDirectiveNoFileComp
	})
	listCmd.RegisterFlagCompletionFunc("outdated", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true\tShow only outdated comments", "false\tShow only non-outdated comments"}, cobra.ShellCompDirectiveNoFileComp
	})
	listCmd.RegisterFlagCompletionFunc("resolved", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true\tShow only resolved comments", "false\tShow only unresolved comments"}, cobra.ShellCompDirectiveNoFileComp
	})
}

type unifiedComment struct {
	Type       string `json:"type"`
	ID         int64  `json:"id"`
	Author     string `json:"author"`
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
	File       string `json:"file,omitempty"`
	Line       string `json:"line,omitempty"`
	Outdated   string `json:"outdated,omitempty"`
	Resolved   string `json:"resolved,omitempty"`
	ReviewID   int64  `json:"review_id,omitempty"`
}

func runList(cmd *cobra.Command, args []string) error {
	client, err := github.NewClient()
	if err != nil {
		return err
	}

	prRef, err := client.ResolvePRReference(args)
	if err != nil {
		return err
	}

	var allComments []unifiedComment

	if listCommentType == "" || listCommentType == "review" {
		reviewComments, err := client.GetReviewComments(prRef.Owner, prRef.Repo, prRef.Number)
		if err != nil {
			return err
		}
		filtered := filterReviewComments(reviewComments)
		for _, c := range filtered {
			line := ""
			if c.OriginalLine != nil {
				line = fmt.Sprintf("%d", *c.OriginalLine)
			}
			outdated := "false"
			if c.IsOutdated() {
				outdated = "true"
			}
			resolved := "false"
			if c.IsResolved {
				resolved = "true"
			}
			allComments = append(allComments, unifiedComment{
				Type:      "review",
				ID:        c.ID,
				Author:    c.User.Login,
				Body:      c.Body,
				CreatedAt: c.CreatedAt.Format("2006-01-02 15:04"),
				File:      c.Path,
				Line:      line,
				Outdated:  outdated,
				Resolved:  resolved,
				ReviewID:  c.PullRequestReviewID,
			})
		}
	}

	if listCommentType == "" || listCommentType == "issue" {
		issueComments, err := client.GetIssueComments(prRef.Owner, prRef.Repo, prRef.Number)
		if err != nil {
			return err
		}
		for _, c := range issueComments {
			allComments = append(allComments, unifiedComment{
				Type:      "issue",
				ID:        c.ID,
				Author:    c.User.Login,
				Body:      c.Body,
				CreatedAt: c.CreatedAt.Format("2006-01-02 15:04"),
			})
		}
	}

	if listJsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(allComments)
	}

	if len(allComments) == 0 {
		fmt.Println("No comments found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TYPE\tID\tFILE\tLINE\tOUTDATED\tRESOLVED\tAUTHOR\tBODY")
	for _, c := range allComments {
		body := github.TruncateString(c.Body, 40)
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			c.Type, c.ID, c.File, c.Line, c.Outdated, c.Resolved, c.Author, body)
	}
	return w.Flush()
}

func filterReviewComments(comments []github.ReviewComment) []github.ReviewComment {
	var result []github.ReviewComment
	for _, c := range comments {
		if listReviewID != 0 && c.PullRequestReviewID != listReviewID {
			continue
		}

		if listOutdated != "" {
			isOutdated := c.IsOutdated()
			if listOutdated == "true" && !isOutdated {
				continue
			}
			if listOutdated == "false" && isOutdated {
				continue
			}
		}

		if !listAll {
			if listResolved != "" {
				if listResolved == "true" && !c.IsResolved {
					continue
				}
				if listResolved == "false" && c.IsResolved {
					continue
				}
			} else {
				if c.IsResolved {
					continue
				}
			}
		}

		result = append(result, c)
	}
	return result
}
