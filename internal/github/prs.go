package github

import (
	"fmt"
	"strings"
	"time"

	"github.com/may/bad-vibes/internal/model"
)

const listPRsQuery = `
query ListPRs($owner: String!, $repo: String!, $branch: String, $states: [PullRequestState!]!) {
  repository(owner: $owner, name: $repo) {
    pullRequests(
      first: 50,
      headRefName: $branch,
      states: $states,
      orderBy: { field: CREATED_AT, direction: DESC }
    ) {
      nodes {
        id
        number
        title
        state
        headRefName
        headRefOid
        url
        createdAt
        additions
        deletions
        changedFiles
        author { login }
      }
    }
  }
}
`

// FetchPRs lists PRs for the given repo.
// branch: "" means all branches.
// states: e.g. []string{"OPEN"} or []string{"CLOSED", "MERGED"}.
func FetchPRs(ref model.PRRef, branch string, states []string) ([]model.PR, error) {
	vars := map[string]any{
		"owner":  ref.Owner,
		"repo":   ref.Repo,
		"states": states,
	}
	if branch != "" {
		vars["branch"] = branch
	} else {
		vars["branch"] = nil
	}

	var data struct {
		Repository struct {
			PullRequests struct {
				Nodes []struct {
					ID           string `json:"id"`
					Number       int    `json:"number"`
					Title        string `json:"title"`
					State        string `json:"state"`
					HeadRefName  string `json:"headRefName"`
					HeadRefOid   string `json:"headRefOid"`
					URL          string `json:"url"`
					CreatedAt    string `json:"createdAt"`
					Additions    int    `json:"additions"`
					Deletions    int    `json:"deletions"`
					ChangedFiles int    `json:"changedFiles"`
					Author       struct {
						Login string `json:"login"`
					} `json:"author"`
				} `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	}

	if err := graphql(listPRsQuery, vars, &data); err != nil {
		return nil, fmt.Errorf("listing PRs: %w", err)
	}

	nodes := data.Repository.PullRequests.Nodes
	prs := make([]model.PR, 0, len(nodes))
	for _, n := range nodes {
		ts, _ := time.Parse(time.RFC3339, n.CreatedAt)
		_ = ts
		prs = append(prs, model.PR{
			ID:           n.ID,
			HeadSHA:      n.HeadRefOid,
			HeadRefName:  n.HeadRefName,
			Title:        n.Title,
			State:        n.State,
			Author:       n.Author.Login,
			URL:          n.URL,
			Number:       n.Number,
			ChangedFiles: n.ChangedFiles,
			Additions:    n.Additions,
			Deletions:    n.Deletions,
		})
	}
	return prs, nil
}

// LatestOpenPR returns the most recent open PR for the given branch.
func LatestOpenPR(ref model.PRRef, branch string) (model.PR, error) {
	prs, err := FetchPRs(ref, branch, []string{"OPEN"})
	if err != nil {
		return model.PR{}, err
	}
	if len(prs) == 0 {
		return model.PR{}, fmt.Errorf(
			"no open PR found for branch %q in %s/%s\n"+
				"  run `bv prs` to see all PRs, or pass a PR number explicitly",
			branch, ref.Owner, ref.Repo,
		)
	}
	return prs[0], nil
}

// StatesFromFlags converts --closed flag into a GraphQL states slice.
func StatesFromFlags(closed bool) []string {
	if closed {
		return []string{"CLOSED", "MERGED"}
	}
	return []string{"OPEN"}
}

// FormatState returns a short display string for a PR state.
func FormatState(state string) string {
	switch strings.ToUpper(state) {
	case "OPEN":
		return "open"
	case "CLOSED":
		return "closed"
	case "MERGED":
		return "merged"
	default:
		return strings.ToLower(state)
	}
}
