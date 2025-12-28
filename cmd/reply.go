package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/STRRL/gh-pr-comments/internal/github"
	"github.com/spf13/cobra"
)

var (
	replyBody       string
	replyPR         string
	replyJsonOutput bool
)

var replyCmd = &cobra.Command{
	Use:   "reply <comment-id>",
	Short: "Reply to a review comment",
	Long: `Reply to a review comment on a pull request.

The reply will be added as a threaded response to the specified review comment.
The comment-id can be found from the 'list', 'view', or 'tree' command output.

Note: Only review comments (inline code comments) support threaded replies.
Issue comments (general PR comments) do not support threading.

Examples:
  # Reply using --body flag
  gh pr-comments reply 2621968472 --body "Thanks for the feedback!"

  # Reply using stdin (useful for multi-line messages)
  echo "Will fix!" | gh pr-comments reply 2621968472

  # Specify PR explicitly
  gh pr-comments reply 2621968472 --pr owner/repo/99 --body "Fixed"

  # Reply with JSON output
  gh pr-comments reply 2621968472 --body "Done" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runReply,
}

func init() {
	replyCmd.Flags().StringVar(&replyBody, "body", "", "Reply message body (reads from stdin if not provided)")
	replyCmd.Flags().StringVar(&replyPR, "pr", "", "PR reference (e.g., owner/repo/123 or just 123)")
	replyCmd.Flags().BoolVar(&replyJsonOutput, "json", false, "Output in JSON format")
	rootCmd.AddCommand(replyCmd)
}

func runReply(cmd *cobra.Command, args []string) error {
	client, err := github.NewClient()
	if err != nil {
		return err
	}

	commentIDStr := args[0]
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid comment ID: %s", commentIDStr)
	}

	body, err := getReplyBody()
	if err != nil {
		return err
	}

	var prArgs []string
	if replyPR != "" {
		prArgs = []string{replyPR}
	}

	prRef, err := client.ResolvePRReference(prArgs)
	if err != nil {
		return fmt.Errorf("could not determine PR: %w\nPlease specify a PR with --pr or run from a branch with an associated PR", err)
	}

	found, err := findReviewComment(client, prRef, commentID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("review comment with ID %d not found in PR %d\nNote: Only review comments support threaded replies", commentID, prRef.Number)
	}

	reply, err := client.ReplyToReviewComment(prRef.Owner, prRef.Repo, prRef.Number, commentID, body)
	if err != nil {
		return err
	}

	if replyJsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(reply)
	}

	printReplySuccess(reply, body)
	return nil
}

func getReplyBody() (string, error) {
	if replyBody != "" {
		return replyBody, nil
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to check stdin: %w", err)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		body := strings.TrimSpace(string(data))
		if body != "" {
			return body, nil
		}
	}

	return "", fmt.Errorf("reply body required: use --body flag or pipe content via stdin")
}

func findReviewComment(client *github.Client, prRef *github.PRReference, commentID int64) (bool, error) {
	comments, err := client.GetReviewComments(prRef.Owner, prRef.Repo, prRef.Number)
	if err != nil {
		return false, err
	}

	for _, c := range comments {
		if c.ID == commentID {
			return true, nil
		}
	}

	return false, nil
}

func printReplySuccess(reply *github.ReviewComment, body string) {
	fmt.Println("Reply created successfully!")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("ID:      %d\n", reply.ID)
	fmt.Printf("Author:  %s\n", reply.User.Login)
	fmt.Printf("Created: %s\n", reply.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("URL:     %s\n", reply.HTMLURL)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()
	fmt.Println(body)
	fmt.Println()
}
