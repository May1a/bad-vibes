package github

import (
	"context"
	"fmt"

	"github.com/may1a/bad-vibes/internal/model"
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
    }
  }
}
`

// FetchPRMetadata retrieves PR metadata via GraphQL.
func FetchPRMetadata(client *Client, ctx context.Context, ref model.PRRef) (model.PR, error) {
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
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	err := client.graphql(ctx, fetchPRQuery, map[string]any{
		"owner":  ref.Owner,
		"repo":   ref.Repo,
		"number": ref.Number,
	}, &data)
	if err != nil {
		return model.PR{}, fmt.Errorf("fetching PR #%d metadata: %w", ref.Number, err)
	}

	gql := data.Repository.PullRequest
	return model.PR{
		ID:           gql.ID,
		HeadSHA:      gql.HeadRefOid,
		HeadRefName:  gql.HeadRefName,
		Title:        gql.Title,
		Body:         gql.Body,
		State:        model.PRState(gql.State),
		Author:       gql.Author.Login,
		URL:          gql.URL,
		Number:       gql.Number,
		ChangedFiles: gql.ChangedFiles,
		Additions:    gql.Additions,
		Deletions:    gql.Deletions,
	}, nil
}

// FetchPR retrieves PR metadata and changed-file stats.
func FetchPR(client *Client, ctx context.Context, ref model.PRRef) (model.PR, []model.PRFile, error) {
	pr, err := FetchPRMetadata(client, ctx, ref)
	if err != nil {
		return model.PR{}, nil, err
	}
	files, err := FetchPRFiles(client, ctx, ref)
	if err != nil {
		return model.PR{}, nil, err
	}
	return pr, files, nil
}

func FetchPRFiles(client *Client, ctx context.Context, ref model.PRRef) ([]model.PRFile, error) {
	type restFile struct {
		Filename         string `json:"filename"`
		PreviousFilename string `json:"previous_filename"`
		Status           string `json:"status"`
		Additions        int    `json:"additions"`
		Deletions        int    `json:"deletions"`
	}

	var files []model.PRFile
	for page := 1; ; page++ {
		var payload []restFile
		path := fmt.Sprintf("/repos/%s/%s/pulls/%d/files?per_page=100&page=%d", ref.Owner, ref.Repo, ref.Number, page)
		if err := client.rest(ctx, "GET", path, nil, &payload, nil); err != nil {
			return nil, fmt.Errorf("fetching PR #%d files: %w", ref.Number, err)
		}
		for _, file := range payload {
			files = append(files, model.PRFile{
				Path:         file.Filename,
				PreviousPath: file.PreviousFilename,
				Status:       file.Status,
				Additions:    file.Additions,
				Deletions:    file.Deletions,
			})
		}
		if len(payload) < 100 {
			break
		}
	}
	return files, nil
}

// FetchDiff retrieves the unified diff for a PR via REST.
func FetchDiff(client *Client, ctx context.Context, ref model.PRRef) (string, error) {
	var raw string
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", ref.Owner, ref.Repo, ref.Number)
	err := client.rest(ctx, "GET", path, nil, &raw, map[string]string{
		"Accept": "application/vnd.github.diff",
	})
	if err != nil {
		return "", fmt.Errorf("fetching diff for PR #%d: %w", ref.Number, err)
	}
	return raw, nil
}
