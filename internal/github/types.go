package github

import "time"

type User struct {
	Login string `json:"login"`
}

type Review struct {
	ID          int64     `json:"id"`
	NodeID      string    `json:"node_id"`
	User        User      `json:"user"`
	Body        string    `json:"body"`
	State       string    `json:"state"`
	HTMLURL     string    `json:"html_url"`
	SubmittedAt time.Time `json:"submitted_at"`
}

type ReviewComment struct {
	ID                    int64     `json:"id"`
	NodeID                string    `json:"node_id"`
	PullRequestReviewID   int64     `json:"pull_request_review_id"`
	DiffHunk              string    `json:"diff_hunk"`
	Path                  string    `json:"path"`
	Position              *int      `json:"position"`
	OriginalPosition      *int      `json:"original_position"`
	CommitID              string    `json:"commit_id"`
	OriginalCommitID      string    `json:"original_commit_id"`
	User                  User      `json:"user"`
	Body                  string    `json:"body"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	HTMLURL               string    `json:"html_url"`
	Line                  *int      `json:"line"`
	OriginalLine          *int      `json:"original_line"`
	StartLine             *int      `json:"start_line"`
	OriginalStartLine     *int      `json:"original_start_line"`
	Side                  string    `json:"side"`
	StartSide             string    `json:"start_side"`
	SubjectType           string    `json:"subject_type"`
	IsResolved            bool      `json:"is_resolved"`
}

func (rc *ReviewComment) IsOutdated() bool {
	return rc.Position == nil || rc.Line == nil
}

type IssueComment struct {
	ID        int64     `json:"id"`
	NodeID    string    `json:"node_id"`
	User      User      `json:"user"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	HTMLURL   string    `json:"html_url"`
}

type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	User   User   `json:"user"`
}
