package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/STRRL/gh-pr-comments/internal/github"
	"github.com/spf13/cobra"
)

var reviewsJsonOutput bool

var reviewsCmd = &cobra.Command{
	Use:   "reviews [pr-reference]",
	Short: "List all reviews on a pull request",
	Long: `List all reviews on a pull request with their states.

If no PR reference is given, finds the PR for the current branch.

PR reference can be:
  - Full URL: https://github.com/owner/repo/pull/123
  - Short form: owner/repo/123
  - Just number: 123 (when in a repo context)
  - Omitted: uses current branch's PR

Examples:
  gh pr-comments reviews
  gh pr-comments reviews https://github.com/owner/repo/pull/123
  gh pr-comments reviews owner/repo/123
  gh pr-comments reviews 123`,
	Args: cobra.MaximumNArgs(1),
	RunE: runReviews,
}

func init() {
	reviewsCmd.Flags().BoolVar(&reviewsJsonOutput, "json", false, "Output in JSON format")
}

func runReviews(cmd *cobra.Command, args []string) error {
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

	if reviewsJsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(reviews)
	}

	if len(reviews) == 0 {
		fmt.Println("No reviews found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATE\tAUTHOR\tSUBMITTED\tBODY")
	for _, r := range reviews {
		submitted := ""
		if !r.SubmittedAt.IsZero() {
			submitted = r.SubmittedAt.Format("2006-01-02 15:04")
		}
		body := github.TruncateString(r.Body, 50)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", r.ID, r.State, r.User.Login, submitted, body)
	}
	return w.Flush()
}
