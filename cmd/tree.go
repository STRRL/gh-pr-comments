package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/STRRL/gh-pr-comments/internal/github"
	"github.com/spf13/cobra"
)

var (
	treeJsonOutput bool
	treeAll        bool
)

var treeCmd = &cobra.Command{
	Use:   "tree [pr-reference]",
	Short: "Show hierarchical view of reviews and comments",
	Long: `Show a tree view of all reviews and their comments on a pull request.

By default, resolved comments are hidden. Use --all to show all comments.

If no PR reference is given, finds the PR for the current branch.

PR reference can be:
  - Full URL: https://github.com/owner/repo/pull/123
  - Short form: owner/repo/123
  - Just number: 123 (when in a repo context)
  - Omitted: uses current branch's PR

Examples:
  gh pr-comments tree
  gh pr-comments tree --all
  gh pr-comments tree https://github.com/owner/repo/pull/123
  gh pr-comments tree owner/repo/123
  gh pr-comments tree 123`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTree,
}

func init() {
	treeCmd.Flags().BoolVar(&treeJsonOutput, "json", false, "Output in JSON format")
	treeCmd.Flags().BoolVar(&treeAll, "all", false, "Show all comments including resolved")
}

type TreeOutput struct {
	PullRequest   *github.PullRequest   `json:"pull_request"`
	Reviews       []ReviewWithComments  `json:"reviews"`
	IssueComments []github.IssueComment `json:"issue_comments"`
}

type ReviewWithComments struct {
	Review   github.Review           `json:"review"`
	Comments []github.ReviewComment  `json:"comments"`
}

func runTree(cmd *cobra.Command, args []string) error {
	client, err := github.NewClient()
	if err != nil {
		return err
	}

	prRef, err := client.ResolvePRReference(args)
	if err != nil {
		return err
	}

	pr, err := client.GetPullRequest(prRef.Owner, prRef.Repo, prRef.Number)
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

	issueComments, err := client.GetIssueComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return err
	}

	commentsByReview := make(map[int64][]github.ReviewComment)
	for _, c := range reviewComments {
		if !treeAll && c.IsResolved {
			continue
		}
		commentsByReview[c.PullRequestReviewID] = append(commentsByReview[c.PullRequestReviewID], c)
	}

	var reviewsWithComments []ReviewWithComments
	for _, r := range reviews {
		reviewsWithComments = append(reviewsWithComments, ReviewWithComments{
			Review:   r,
			Comments: commentsByReview[r.ID],
		})
	}

	sort.Slice(reviewsWithComments, func(i, j int) bool {
		return reviewsWithComments[i].Review.SubmittedAt.Before(reviewsWithComments[j].Review.SubmittedAt)
	})

	sort.Slice(issueComments, func(i, j int) bool {
		return issueComments[i].CreatedAt.Before(issueComments[j].CreatedAt)
	})

	if treeJsonOutput {
		output := TreeOutput{
			PullRequest:   pr,
			Reviews:       reviewsWithComments,
			IssueComments: issueComments,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	printTree(pr, reviewsWithComments, issueComments)
	return nil
}

func printTree(pr *github.PullRequest, reviews []ReviewWithComments, issueComments []github.IssueComment) {
	fmt.Printf("PR #%d: %s\n", pr.Number, pr.Title)
	fmt.Println("\u2502")

	for i, r := range reviews {
		isLastReview := i == len(reviews)-1 && len(issueComments) == 0
		prefix := "\u251c\u2500\u2500"
		childPrefix := "\u2502   "
		if isLastReview {
			prefix = "\u2514\u2500\u2500"
			childPrefix = "    "
		}

		submitted := ""
		if !r.Review.SubmittedAt.IsZero() {
			submitted = r.Review.SubmittedAt.Format("2006-01-02")
		}

		fmt.Printf("%s Review %d by %s (%s) - %s\n",
			prefix, r.Review.ID, r.Review.User.Login, r.Review.State, submitted)

		if r.Review.Body != "" {
			body := github.TruncateString(r.Review.Body, 60)
			fmt.Printf("%s\u2502   %s\n", childPrefix, body)
		}

		if len(r.Comments) == 0 {
			fmt.Printf("%s\u2514\u2500\u2500 (no inline comments)\n", childPrefix)
		} else {
			for j, c := range r.Comments {
				isLastComment := j == len(r.Comments)-1
				commentPrefix := "\u251c\u2500\u2500"
				if isLastComment {
					commentPrefix = "\u2514\u2500\u2500"
				}

				line := ""
				if c.OriginalLine != nil {
					line = fmt.Sprintf(":%d", *c.OriginalLine)
				}
				var marks []string
				if c.IsOutdated() {
					marks = append(marks, "outdated")
				}
				if c.IsResolved {
					marks = append(marks, "resolved")
				}
				markStr := ""
				if len(marks) > 0 {
					markStr = " (" + strings.Join(marks, ", ") + ")"
				}

				fmt.Printf("%s%s [%d] %s%s%s\n", childPrefix, commentPrefix, c.ID, c.Path, line, markStr)

				bodyPrefix := childPrefix + "\u2502   "
				if isLastComment {
					bodyPrefix = childPrefix + "    "
				}
				body := github.TruncateString(c.Body, 60)
				fmt.Printf("%s\u2514\u2500\u2500 %s\n", bodyPrefix, body)
			}
		}
		fmt.Printf("%s\n", childPrefix)
	}

	if len(issueComments) > 0 {
		fmt.Printf("\u2514\u2500\u2500 Issue Comments (%d)\n", len(issueComments))
		for i, c := range issueComments {
			isLast := i == len(issueComments)-1
			prefix := "    \u251c\u2500\u2500"
			if isLast {
				prefix = "    \u2514\u2500\u2500"
			}
			fmt.Printf("%s %d by %s - %s\n", prefix, c.ID, c.User.Login, c.CreatedAt.Format("2006-01-02"))
		}
	}
}
