package cmd

import (
	"fmt"

	"github.com/STRRL/gh-pr-comments/internal/github"
	"github.com/spf13/cobra"
)

func completeCommentIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	client, err := github.NewClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	prRef, err := client.ResolvePRReference(nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string

	reviewComments, err := client.GetReviewComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err == nil {
		for _, c := range reviewComments {
			desc := github.TruncateString(c.Body, 40)
			completion := fmt.Sprintf("%d\t[review] %s", c.ID, desc)
			completions = append(completions, completion)
		}
	}

	issueComments, err := client.GetIssueComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err == nil {
		for _, c := range issueComments {
			desc := github.TruncateString(c.Body, 40)
			completion := fmt.Sprintf("%d\t[issue] %s", c.ID, desc)
			completions = append(completions, completion)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeReviewCommentIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {

	client, err := github.NewClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	prRef, err := client.ResolvePRReference(nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string

	reviewComments, err := client.GetReviewComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err == nil {
		for _, c := range reviewComments {
			desc := github.TruncateString(c.Body, 40)
			completion := fmt.Sprintf("%d\t%s: %s", c.ID, c.Path, desc)
			completions = append(completions, completion)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeReviewIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := github.NewClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	prRef, err := client.ResolvePRReference(nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string

	reviews, err := client.GetReviews(prRef.Owner, prRef.Repo, prRef.Number)
	if err == nil {
		for _, r := range reviews {
			desc := r.State
			if r.Body != "" {
				desc = fmt.Sprintf("%s: %s", r.State, github.TruncateString(r.Body, 30))
			}
			completion := fmt.Sprintf("%d\t[%s] %s", r.ID, r.User.Login, desc)
			completions = append(completions, completion)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
