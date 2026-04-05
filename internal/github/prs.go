package github

import (
	"context"
	"fmt"

	"github.com/may1a/bv/internal/model"
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
        author { login }
      }
    }
  }
}
`

// FetchPRs lists PRs for the given repo.
// branch: "" means all branches.
func FetchPRs(client *Client, ctx context.Context, ref model.PRRef, branch string, states []model.PRState) ([]model.PR, error) {
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
					ID          string `json:"id"`
					Number      int    `json:"number"`
					Title       string `json:"title"`
					State       string `json:"state"`
					HeadRefName string `json:"headRefName"`
					Author      struct {
						Login string `json:"login"`
					} `json:"author"`
				} `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	}

	if err := client.graphql(ctx, listPRsQuery, vars, &data); err != nil {
		return nil, fmt.Errorf("listing PRs: %w", err)
	}

	nodes := data.Repository.PullRequests.Nodes
	prs := make([]model.PR, 0, len(nodes))
	for _, n := range nodes {
		prs = append(prs, model.PR{
			ID:          n.ID,
			HeadRefName: n.HeadRefName,
			Title:       n.Title,
			State:       model.PRState(n.State),
			Author:      n.Author.Login,
			Number:      n.Number,
		})
	}
	return prs, nil
}

// LatestOpenPR returns the most recent open PR for the given branch.
func LatestOpenPR(client *Client, ctx context.Context, ref model.PRRef, branch string) (model.PR, error) {
	prs, err := FetchPRs(client, ctx, ref, branch, []model.PRState{model.PRStateOpen})
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

func ListStates(closed bool) []model.PRState {
	if closed {
		return []model.PRState{model.PRStateClosed, model.PRStateMerged}
	}
	return []model.PRState{model.PRStateOpen}
}
