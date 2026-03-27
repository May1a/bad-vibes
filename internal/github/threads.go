package github

import (
	"context"
	"fmt"
	"time"

	"github.com/may/bad-vibes/internal/model"
)

const fetchThreadsQuery = `
query FetchThreads($owner: String!, $repo: String!, $number: Int!, $after: String) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      reviewThreads(first: 50, after: $after) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          startLine
          diffSide
          subjectType
          comments(first: 50) {
            nodes {
              id
              body
              diffHunk
              createdAt
              author { login }
            }
          }
        }
      }
    }
  }
}
`

// FetchReviewThreads fetches all review threads for a PR, paginated.
func FetchReviewThreads(client *Client, ctx context.Context, ref model.PRRef) ([]model.ReviewThread, error) {
	type gqlComment struct {
		ID        string `json:"id"`
		Body      string `json:"body"`
		DiffHunk  string `json:"diffHunk"`
		CreatedAt string `json:"createdAt"`
		Author    struct {
			Login string `json:"login"`
		} `json:"author"`
	}
	type gqlThread struct {
		ID          string `json:"id"`
		IsResolved  bool   `json:"isResolved"`
		IsOutdated  bool   `json:"isOutdated"`
		Path        string `json:"path"`
		Line        *int   `json:"line"`
		StartLine   *int   `json:"startLine"`
		DiffSide    string `json:"diffSide"`
		SubjectType string `json:"subjectType"`
		Comments    struct {
			Nodes []gqlComment `json:"nodes"`
		} `json:"comments"`
	}
	type gqlData struct {
		Repository struct {
			PullRequest struct {
				ReviewThreads struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []gqlThread `json:"nodes"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	var allThreads []model.ReviewThread
	var cursor *string

	for {
		vars := map[string]any{
			"owner":  ref.Owner,
			"repo":   ref.Repo,
			"number": ref.Number,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		var data gqlData
		if err := client.graphql(ctx, fetchThreadsQuery, vars, &data); err != nil {
			return nil, fmt.Errorf("fetching threads for PR #%d: %w", ref.Number, err)
		}

		rt := data.Repository.PullRequest.ReviewThreads
		for _, t := range rt.Nodes {
			thread := model.ReviewThread{
				ID:          t.ID,
				Path:        t.Path,
				DiffSide:    t.DiffSide,
				SubjectType: t.SubjectType,
				IsResolved:  t.IsResolved,
				IsOutdated:  t.IsOutdated,
			}
			if t.Line != nil {
				thread.Line = *t.Line
			}
			if t.StartLine != nil {
				thread.StartLine = *t.StartLine
			}
			for _, c := range t.Comments.Nodes {
				ts, _ := time.Parse(time.RFC3339, c.CreatedAt)
				thread.Comments = append(thread.Comments, model.Comment{
					ID:        c.ID,
					Author:    c.Author.Login,
					Body:      c.Body,
					DiffHunk:  c.DiffHunk,
					CreatedAt: ts,
				})
			}
			allThreads = append(allThreads, thread)
		}

		if !rt.PageInfo.HasNextPage {
			break
		}
		c := rt.PageInfo.EndCursor
		cursor = &c
	}

	return allThreads, nil
}

// FindUnresolvedThreadAt returns the GraphQL node ID of the unresolved thread at
// the given file path and line number. Returns ("", false, nil) when no match is
// found. Pass body to disambiguate multiple unresolved threads on the same line.
// Pass path="" to match the first unresolved PR-level thread.
func FindUnresolvedThreadAt(client *Client, ctx context.Context, ref model.PRRef, path string, line int, body string) (string, bool, error) {
	threads, err := FetchReviewThreads(client, ctx, ref)
	if err != nil {
		return "", false, err
	}
	return findUnresolvedThreadID(threads, path, line, body)
}

func findUnresolvedThreadID(threads []model.ReviewThread, path string, line int, body string) (string, bool, error) {
	var matches []model.ReviewThread

	for _, t := range threads {
		if t.IsResolved {
			continue
		}
		if path == "" {
			if t.Path == "" {
				return t.ID, true, nil
			}
			continue
		}
		if t.Path == path && t.Line == line {
			matches = append(matches, t)
		}
	}

	if len(matches) == 0 {
		return "", false, nil
	}

	if body != "" {
		var bodyMatches []model.ReviewThread
		for _, t := range matches {
			if threadHasCommentBody(t, body) {
				bodyMatches = append(bodyMatches, t)
			}
		}
		switch len(bodyMatches) {
		case 0:
			return "", false, nil
		case 1:
			return bodyMatches[0].ID, true, nil
		default:
			return "", false, fmt.Errorf("multiple unresolved threads match %s:%d with the same body", path, line)
		}
	}

	if len(matches) > 1 {
		return "", false, fmt.Errorf("multiple unresolved threads match %s:%d; body required to disambiguate", path, line)
	}

	return matches[0].ID, true, nil
}

func threadHasCommentBody(thread model.ReviewThread, body string) bool {
	for _, c := range thread.Comments {
		if c.Body == body {
			return true
		}
	}
	return false
}

const resolveThreadMutation = `
mutation ResolveThread($threadId: ID!) {
  resolveReviewThread(input: { threadId: $threadId }) {
    thread {
      id
      isResolved
    }
  }
}
`

// ResolveThread marks a review thread as resolved.
func ResolveThread(client *Client, ctx context.Context, threadID string) error {
	var data struct {
		ResolveReviewThread struct {
			Thread struct {
				ID         string `json:"id"`
				IsResolved bool   `json:"isResolved"`
			} `json:"thread"`
		} `json:"resolveReviewThread"`
	}
	if err := client.graphql(ctx, resolveThreadMutation, map[string]any{
		"threadId": threadID,
	}, &data); err != nil {
		return fmt.Errorf("resolving thread %s: %w", threadID, err)
	}
	if !data.ResolveReviewThread.Thread.IsResolved {
		return fmt.Errorf("thread %s was not marked resolved (check permissions)", threadID)
	}
	return nil
}
