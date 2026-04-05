package github

import (
	"context"
	"fmt"

	"github.com/may1a/bv/internal/model"
)

// PostedComment holds identifiers returned after posting a review comment.
type PostedComment struct {
	ReviewID  int64  // REST review ID
	CommentID int64  // REST comment ID within the review
	ThreadID  string // GraphQL thread node ID (empty — not returned by REST)
}

// PostReviewComment submits a single inline review comment via REST.
func PostReviewComment(client *Client, ctx context.Context, ref model.PRRef, headSHA, path, body, side string, line int) (PostedComment, error) {
	type commentPayload struct {
		Path string `json:"path"`
		Line int    `json:"line"`
		Side string `json:"side"`
		Body string `json:"body"`
	}
	payload := struct {
		CommitID string           `json:"commit_id"`
		Event    string           `json:"event"`
		Comments []commentPayload `json:"comments"`
	}{
		CommitID: headSHA,
		Event:    "COMMENT",
		Comments: []commentPayload{
			{Path: path, Line: line, Side: side, Body: body},
		},
	}

	var result struct {
		ID       int64 `json:"id"`
		Comments []struct {
			ID int64 `json:"id"`
		} `json:"comments"`
	}

	apiPath := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", ref.Owner, ref.Repo, ref.Number)
	if err := client.rest(ctx, "POST", apiPath, payload, &result, nil); err != nil {
		return PostedComment{}, fmt.Errorf("posting review comment: %w", err)
	}

	pc := PostedComment{ReviewID: result.ID}
	if len(result.Comments) > 0 {
		pc.CommentID = result.Comments[0].ID
	}
	return pc, nil
}
