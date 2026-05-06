package model

import "time"

// PRRef is the canonical parsed form of any PR reference string.
type PRRef struct {
	Owner  string
	Repo   string
	Number int
}

type PRState string

const (
	PRStateOpen   PRState = "OPEN"
	PRStateClosed PRState = "CLOSED"
	PRStateMerged PRState = "MERGED"
)

// PR holds metadata fetched from the GitHub API.
type PR struct {
	ID           string // GraphQL node ID
	HeadSHA      string // head commit OID — required when posting review comments
	HeadRefName  string // branch name
	Title        string
	Body         string
	State        PRState
	Author       string
	URL          string
	Number       int
	ChangedFiles int
	Additions    int
	Deletions    int
}

// PRFile holds per-file diff stats for a PR.
type PRFile struct {
	Path         string
	PreviousPath string
	Status       string
	Additions    int
	Deletions    int
}

// ReviewThread is a single inline or file-level review thread on a PR.
type ReviewThread struct {
	ID          string // GraphQL node ID (PRRT_...) — used in resolveReviewThread
	Path        string // file path; empty for PR-level threads
	DiffSide    string // LEFT | RIGHT
	SubjectType string // LINE | FILE
	Line        int    // 0 if not set (file-level comment)
	StartLine   int    // for multi-line comments; 0 if single line
	IsResolved  bool
	IsOutdated  bool
	Comments    []Comment
}

// Comment is a single message within a ReviewThread.
type Comment struct {
	ID        string
	Author    string
	Body      string
	DiffHunk  string
	CreatedAt time.Time
}

// PRCache is the on-disk structure stored at
// ~/.cache/bad-vibes/<owner>/<repo>/<number>.json
type PRCache struct {
	Owner   string
	Repo    string
	Number  int
	PRID    string // GraphQL node ID of the PR
	HeadSHA string // cached head commit OID
}

type IssueStatus string

const (
	IssueStatusOpen   IssueStatus = "open"
	IssueStatusClosed IssueStatus = "closed"
)

// Issue is a repo-level work item stored in <repo-root>/.bv/issues/<id>.json.
type Issue struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	Body      string      `json:"body"`
	Status    IssueStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// PendingComment is a staged comment in a review session (not yet submitted to GitHub).
type PendingComment struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Side string `json:"side"`
	Body string `json:"body"`
}

// ReviewSession is the on-disk state for an active review session, stored at
// ~/.bv/<owner>/<repo>/<number>.json
type ReviewSession struct {
	Owner           string           `json:"owner"`
	Repo            string           `json:"repo"`
	Number          int              `json:"number"`
	PRID            string           `json:"pr_id"`
	HeadSHA         string           `json:"head_sha"`
	StartedAt       time.Time        `json:"started_at"`
	PendingComments []PendingComment `json:"pending_comments"`
}
