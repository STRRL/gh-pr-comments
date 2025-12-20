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
	listJsonOutput bool
	listReviewID   int64
	listOutdated   string
	listResolved   string
	listAll        bool
)

var listCmd = &cobra.Command{
	Use:   "list [pr-reference]",
	Short: "List review comments on a pull request",
	Long: `List all review comments (inline code comments) on a pull request.

By default, resolved comments are hidden. Use --all to show all comments,
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
	listCmd.Flags().Int64Var(&listReviewID, "review-id", 0, "Filter by review ID")
	listCmd.Flags().StringVar(&listOutdated, "outdated", "", "Filter by outdated status (true/false)")
	listCmd.Flags().StringVar(&listResolved, "resolved", "", "Filter by resolved status (true/false)")
	listCmd.Flags().BoolVar(&listAll, "all", false, "Show all comments including resolved")
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

	comments, err := client.GetReviewComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return err
	}

	filtered := filterComments(comments)

	if listJsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(filtered)
	}

	if len(filtered) == 0 {
		fmt.Println("No review comments found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tFILE\tLINE\tOUTDATED\tRESOLVED\tREVIEW ID\tAUTHOR\tBODY")
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
		body := github.TruncateString(c.Body, 40)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			c.ID, c.Path, line, outdated, resolved, c.PullRequestReviewID, c.User.Login, body)
	}
	return w.Flush()
}

func filterComments(comments []github.ReviewComment) []github.ReviewComment {
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
