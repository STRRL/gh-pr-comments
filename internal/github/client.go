package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	graphql "github.com/cli/shurcooL-graphql"
)

type Client struct {
	rest    *api.RESTClient
	graphql *api.GraphQLClient
}

func NewClient() (*Client, error) {
	restClient, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}
	graphqlClient, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL client: %w", err)
	}
	return &Client{rest: restClient, graphql: graphqlClient}, nil
}

type PRReference struct {
	Owner  string
	Repo   string
	Number int
}

func ParsePRReference(ref string) (*PRReference, error) {
	urlPattern := regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/pull/(\d+)`)
	if matches := urlPattern.FindStringSubmatch(ref); matches != nil {
		num, _ := strconv.Atoi(matches[3])
		return &PRReference{Owner: matches[1], Repo: matches[2], Number: num}, nil
	}

	shortPattern := regexp.MustCompile(`^([^/]+)/([^/]+)/(\d+)$`)
	if matches := shortPattern.FindStringSubmatch(ref); matches != nil {
		num, _ := strconv.Atoi(matches[3])
		return &PRReference{Owner: matches[1], Repo: matches[2], Number: num}, nil
	}

	if num, err := strconv.Atoi(ref); err == nil {
		return &PRReference{Number: num}, nil
	}

	return nil, fmt.Errorf("invalid PR reference: %s (expected URL, owner/repo/number, or number)", ref)
}

func (c *Client) GetCurrentRepo() (owner, repo string, err error) {
	currentRepo, err := repository.Current()
	if err != nil {
		return "", "", fmt.Errorf("not in a git repository or unable to determine repo: %w", err)
	}
	return currentRepo.Owner, currentRepo.Name, nil
}

func (c *Client) GetPullRequest(owner, repo string, number int) (*PullRequest, error) {
	var pr PullRequest
	path := fmt.Sprintf("repos/%s/%s/pulls/%d", owner, repo, number)
	if err := c.rest.Get(path, &pr); err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}
	return &pr, nil
}

func (c *Client) GetReviews(owner, repo string, number int) ([]Review, error) {
	var reviews []Review
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", owner, repo, number)
	if err := c.rest.Get(path, &reviews); err != nil {
		return nil, fmt.Errorf("failed to get reviews: %w", err)
	}
	return reviews, nil
}

func (c *Client) GetReviewComments(owner, repo string, number int) ([]ReviewComment, error) {
	var comments []ReviewComment
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/comments", owner, repo, number)
	if err := c.rest.Get(path, &comments); err != nil {
		return nil, fmt.Errorf("failed to get review comments: %w", err)
	}

	resolvedMap, err := c.getResolvedStatus(owner, repo, number)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to fetch resolved status: %v\n", err)
	} else {
		for i := range comments {
			if resolved, ok := resolvedMap[comments[i].ID]; ok {
				comments[i].IsResolved = resolved
			}
		}
	}

	return comments, nil
}

func (c *Client) getResolvedStatus(owner, repo string, number int) (map[int64]bool, error) {
	var query struct {
		Repository struct {
			PullRequest struct {
				ReviewThreads struct {
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					}
					Nodes []struct {
						IsResolved bool
						Comments   struct {
							Nodes []struct {
								DatabaseId int64
							}
						} `graphql:"comments(first: 100)"`
					}
				} `graphql:"reviewThreads(first: 100)"`
			} `graphql:"pullRequest(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]interface{}{
		"owner":  graphql.String(owner),
		"repo":   graphql.String(repo),
		"number": graphql.Int(number),
	}

	if err := c.graphql.Query("GetReviewThreads", &query, variables); err != nil {
		return nil, err
	}

	result := make(map[int64]bool)
	for _, thread := range query.Repository.PullRequest.ReviewThreads.Nodes {
		for _, comment := range thread.Comments.Nodes {
			result[comment.DatabaseId] = thread.IsResolved
		}
	}

	return result, nil
}

func (c *Client) GetIssueComments(owner, repo string, number int) ([]IssueComment, error) {
	var comments []IssueComment
	path := fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, number)
	if err := c.rest.Get(path, &comments); err != nil {
		return nil, fmt.Errorf("failed to get issue comments: %w", err)
	}
	return comments, nil
}

func (c *Client) ReplyToReviewComment(owner, repo string, prNumber int, commentID int64, body string) (*ReviewComment, error) {
	var reply ReviewComment
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/comments/%d/replies", owner, repo, prNumber, commentID)
	payload := map[string]string{"body": body}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request body: %w", err)
	}
	if err := c.rest.Post(path, bytes.NewBuffer(jsonData), &reply); err != nil {
		return nil, fmt.Errorf("failed to reply to comment: %w", err)
	}
	return &reply, nil
}

func (pr *PRReference) ResolveOwnerRepo(c *Client) error {
	if pr.Owner != "" && pr.Repo != "" {
		return nil
	}
	owner, repo, err := c.GetCurrentRepo()
	if err != nil {
		return err
	}
	pr.Owner = owner
	pr.Repo = repo
	return nil
}

func TruncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

type PRSearchResult struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Head   struct {
		Ref string `json:"ref"`
	} `json:"head"`
}

func (c *Client) FindPRForBranch(owner, repo, branch string) (*PRReference, error) {
	var prs []PRSearchResult
	path := fmt.Sprintf("repos/%s/%s/pulls?head=%s:%s&state=all", owner, repo, url.QueryEscape(owner), url.QueryEscape(branch))
	if err := c.rest.Get(path, &prs); err != nil {
		return nil, fmt.Errorf("failed to search PRs: %w", err)
	}

	if len(prs) == 0 {
		return nil, fmt.Errorf("no pull request found for branch '%s'", branch)
	}

	return &PRReference{
		Owner:  owner,
		Repo:   repo,
		Number: prs[0].Number,
	}, nil
}

func (c *Client) ResolvePRReference(args []string) (*PRReference, error) {
	if len(args) > 0 && args[0] != "" {
		prRef, err := ParsePRReference(args[0])
		if err != nil {
			return nil, err
		}
		if err := prRef.ResolveOwnerRepo(c); err != nil {
			return nil, err
		}
		return prRef, nil
	}

	owner, repo, err := c.GetCurrentRepo()
	if err != nil {
		return nil, fmt.Errorf("no PR specified and %w", err)
	}

	branch, err := GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("no PR specified and %w", err)
	}

	prRef, err := c.FindPRForBranch(owner, repo, branch)
	if err != nil {
		return nil, fmt.Errorf("no PR specified and %w", err)
	}

	return prRef, nil
}

func (c *Client) MinimizeComment(nodeID string, classifier CommentClassifier) error {
	var mutation struct {
		MinimizeComment struct {
			MinimizedComment struct {
				IsMinimized bool
			}
		} `graphql:"minimizeComment(input: $input)"`
	}

	type MinimizeCommentInput struct {
		SubjectID  graphql.ID     `json:"subjectId"`
		Classifier graphql.String `json:"classifier"`
	}

	variables := map[string]interface{}{
		"input": MinimizeCommentInput{
			SubjectID:  graphql.ID(nodeID),
			Classifier: graphql.String(classifier),
		},
	}

	if err := c.graphql.Mutate("MinimizeComment", &mutation, variables); err != nil {
		return fmt.Errorf("failed to minimize comment: %w", err)
	}

	return nil
}

func (c *Client) UnminimizeComment(nodeID string) error {
	var mutation struct {
		UnminimizeComment struct {
			UnminimizedComment struct {
				IsMinimized bool
			}
		} `graphql:"unminimizeComment(input: $input)"`
	}

	type UnminimizeCommentInput struct {
		SubjectID graphql.ID `json:"subjectId"`
	}

	variables := map[string]interface{}{
		"input": UnminimizeCommentInput{
			SubjectID: graphql.ID(nodeID),
		},
	}

	if err := c.graphql.Mutate("UnminimizeComment", &mutation, variables); err != nil {
		return fmt.Errorf("failed to unminimize comment: %w", err)
	}

	return nil
}
