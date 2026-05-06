package github

import (
	"context"
	"fmt"

	"github.com/may1a/bad-vibes/internal/model"
)

// PostedComment holds identifiers returned after posting a review comment.
type PostedComment struct {
	ReviewID  int64  // REST review ID
	CommentID int64  // REST comment ID within the review
	ThreadID  string // GraphQL thread node ID (empty — not returned by REST)
}

// ReviewEvent is the GitHub review submission event type.
type ReviewEvent string

const (
	ReviewEventApprove        ReviewEvent = "APPROVE"
	ReviewEventRequestChanges ReviewEvent = "REQUEST_CHANGES"
	ReviewEventComment        ReviewEvent = "COMMENT"
)

// ReviewCommentInput is a single comment to include in a submitted review.
type ReviewCommentInput struct {
	Path string
	Line int
	Side string
	Body string
}

// SubmitReview submits a full GitHub review (with optional inline comments) via REST.
func SubmitReview(client *Client, ctx context.Context, ref model.PRRef, headSHA string, event ReviewEvent, body string, comments []ReviewCommentInput) error {
	type commentPayload struct {
		Path string `json:"path"`
		Line int    `json:"line"`
		Side string `json:"side"`
		Body string `json:"body"`
	}
	cs := make([]commentPayload, len(comments))
	for i, c := range comments {
		cs[i] = commentPayload{Path: c.Path, Line: c.Line, Side: c.Side, Body: c.Body}
	}
	payload := struct {
		CommitID string           `json:"commit_id"`
		Event    string           `json:"event"`
		Body     string           `json:"body,omitempty"`
		Comments []commentPayload `json:"comments,omitempty"`
	}{
		CommitID: headSHA,
		Event:    string(event),
		Body:     body,
		Comments: cs,
	}

	apiPath := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", ref.Owner, ref.Repo, ref.Number)
	if err := client.rest(ctx, "POST", apiPath, payload, nil, nil); err != nil {
		return fmt.Errorf("submitting review: %w", err)
	}
	return nil
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
