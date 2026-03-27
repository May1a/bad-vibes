package github

import (
	"fmt"

	"github.com/may/bad-vibes/internal/model"
)

const fetchPRQuery = `
query FetchPR($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      id
      number
      title
      body
      state
      headRefOid
      headRefName
      url
      additions
      deletions
      changedFiles
      author { login }
      files(first: 100) {
        nodes { path }
      }
    }
  }
}
`

// FetchPR retrieves PR metadata via GraphQL.
func FetchPR(ref model.PRRef) (model.PR, []string, error) {
	var data struct {
		Repository struct {
			PullRequest struct {
				ID           string `json:"id"`
				Number       int    `json:"number"`
				Title        string `json:"title"`
				Body         string `json:"body"`
				State        string `json:"state"`
				HeadRefOid   string `json:"headRefOid"`
				HeadRefName  string `json:"headRefName"`
				URL          string `json:"url"`
				Additions    int    `json:"additions"`
				Deletions    int    `json:"deletions"`
				ChangedFiles int    `json:"changedFiles"`
				Author       struct {
					Login string `json:"login"`
				} `json:"author"`
				Files struct {
					Nodes []struct {
						Path string `json:"path"`
					} `json:"nodes"`
				} `json:"files"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	err := graphql(fetchPRQuery, map[string]any{
		"owner":  ref.Owner,
		"repo":   ref.Repo,
		"number": ref.Number,
	}, &data)
	if err != nil {
		return model.PR{}, nil, fmt.Errorf("fetching PR #%d: %w", ref.Number, err)
	}

	gql := data.Repository.PullRequest
	pr := model.PR{
		ID:           gql.ID,
		HeadSHA:      gql.HeadRefOid,
		HeadRefName:  gql.HeadRefName,
		Title:        gql.Title,
		Body:         gql.Body,
		State:        gql.State,
		Author:       gql.Author.Login,
		URL:          gql.URL,
		Number:       gql.Number,
		ChangedFiles: gql.ChangedFiles,
		Additions:    gql.Additions,
		Deletions:    gql.Deletions,
	}

	files := make([]string, 0, len(gql.Files.Nodes))
	for _, f := range gql.Files.Nodes {
		files = append(files, f.Path)
	}

	return pr, files, nil
}

// FetchDiff retrieves the unified diff for a PR via REST.
func FetchDiff(ref model.PRRef) (string, error) {
	var raw string
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", ref.Owner, ref.Repo, ref.Number)
	err := rest("GET", path, nil, &raw, map[string]string{
		"Accept": "application/vnd.github.diff",
	})
	if err != nil {
		return "", fmt.Errorf("fetching diff for PR #%d: %w", ref.Number, err)
	}
	return raw, nil
}
