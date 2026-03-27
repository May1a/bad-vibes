package model

import "time"

// PRRef is the canonical parsed form of any PR reference string.
type PRRef struct {
	Owner  string
	Repo   string
	Number int
}

// PR holds metadata fetched from the GitHub API.
type PR struct {
	ID           string // GraphQL node ID
	HeadSHA      string // head commit OID — required when posting review comments
	HeadRefName  string // branch name
	Title        string
	Body         string
	State        string // OPEN | CLOSED | MERGED
	Author       string
	URL          string
	Number       int
	ChangedFiles int
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

// Anchor is a user-defined local alias for a review thread.
type Anchor struct {
	Tag      string    // e.g. "perf" (without the #)
	ThreadID string    // GraphQL node ID of the ReviewThread
	Path     string    // file path for display convenience
	Line     int
	Body     string    // first comment body snippet
	Created  time.Time
}

// PRCache is the on-disk structure stored at
// ~/.cache/bad-vibes/<owner>/<repo>/<number>.json
type PRCache struct {
	Owner   string
	Repo    string
	Number  int
	PRID    string   // GraphQL node ID of the PR
	HeadSHA string   // cached head commit OID
	Anchors []Anchor
}
