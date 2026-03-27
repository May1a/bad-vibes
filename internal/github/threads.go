package github

import (
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
func FetchReviewThreads(ref model.PRRef) ([]model.ReviewThread, error) {
	type gqlComment struct {
		ID       string `json:"id"`
		Body     string `json:"body"`
		DiffHunk string `json:"diffHunk"`
		CreatedAt string `json:"createdAt"`
		Author   struct {
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
		if err := graphql(fetchThreadsQuery, vars, &data); err != nil {
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

// FindUnresolvedThreadAt returns the GraphQL node ID of the first unresolved
// thread at the given file path and line number. Returns ("", false, nil) when
// no match is found. Pass path="" to match PR-level threads (subjectType "PR").
func FindUnresolvedThreadAt(ref model.PRRef, path string, line int) (string, bool, error) {
	threads, err := FetchReviewThreads(ref)
	if err != nil {
		return "", false, err
	}
	for _, t := range threads {
		if t.IsResolved {
			continue
		}
		if path == "" {
			// PR-level thread
			if t.SubjectType == "PR" || t.Path == "" {
				return t.ID, true, nil
			}
		} else if t.Path == path && t.Line == line {
			return t.ID, true, nil
		}
	}
	return "", false, nil
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
func ResolveThread(threadID string) error {
	var data struct {
		ResolveReviewThread struct {
			Thread struct {
				ID         string `json:"id"`
				IsResolved bool   `json:"isResolved"`
			} `json:"thread"`
		} `json:"resolveReviewThread"`
	}
	if err := graphql(resolveThreadMutation, map[string]any{
		"threadId": threadID,
	}, &data); err != nil {
		return fmt.Errorf("resolving thread %s: %w", threadID, err)
	}
	if !data.ResolveReviewThread.Thread.IsResolved {
		return fmt.Errorf("thread %s was not marked resolved (check permissions)", threadID)
	}
	return nil
}
