package github

import (
	"context"

	"github.com/may1a/bad-vibes/internal/model"
)

// MockClient is a test double for Client that returns predefined responses.
type MockClient struct {
	// PRs to return from FetchPRs
	MockPRs []model.PR
	// PR to return from FetchPR
	MockPR model.PR
	// Files to return from FetchPR
	MockFiles []model.PRFile
	// Threads to return from FetchReviewThreads
	MockThreads []model.ReviewThread
	// Diff to return from FetchDiff
	MockDiff string

	// Errors to return
	FetchPRErr       error
	FetchPRsErr      error
	FetchThreadsErr  error
	FetchDiffErr     error
	ResolveThreadErr error
	PostCommentErr   error

	// Call tracking
	FetchPRCalls       int
	FetchPRsCalls      int
	FetchThreadsCalls  int
	FetchDiffCalls     int
	ResolveThreadCalls int
	PostCommentCalls   int

	// PostedComment to return from PostReviewComment
	MockPostedComment PostedComment
}

// Ensure MockClient satisfies the same interface patterns as real client functions
// Note: We use package-level functions that accept *Client, so we provide test helpers

// MockFetchPR is a test helper that calls FetchPR with the mock client.
func (m *MockClient) MockFetchPR(ctx context.Context, ref model.PRRef) (model.PR, []model.PRFile, error) {
	m.FetchPRCalls++
	if m.FetchPRErr != nil {
		return model.PR{}, nil, m.FetchPRErr
	}
	return m.MockPR, m.MockFiles, nil
}

// MockFetchPRs is a test helper that calls FetchPRs with the mock client.
func (m *MockClient) MockFetchPRs(ctx context.Context, ref model.PRRef, branch string, states []model.PRState) ([]model.PR, error) {
	m.FetchPRsCalls++
	if m.FetchPRsErr != nil {
		return nil, m.FetchPRsErr
	}
	return m.MockPRs, nil
}

// MockFetchReviewThreads is a test helper that calls FetchReviewThreads with the mock client.
func (m *MockClient) MockFetchReviewThreads(ctx context.Context, ref model.PRRef) ([]model.ReviewThread, error) {
	m.FetchThreadsCalls++
	if m.FetchThreadsErr != nil {
		return nil, m.FetchThreadsErr
	}
	return m.MockThreads, nil
}

// MockFetchDiff is a test helper that calls FetchDiff with the mock client.
func (m *MockClient) MockFetchDiff(ctx context.Context, ref model.PRRef) (string, error) {
	m.FetchDiffCalls++
	if m.FetchDiffErr != nil {
		return "", m.FetchDiffErr
	}
	return m.MockDiff, nil
}

// MockResolveThread is a test helper that calls ResolveThread with the mock client.
func (m *MockClient) MockResolveThread(ctx context.Context, threadID string) error {
	m.ResolveThreadCalls++
	if m.ResolveThreadErr != nil {
		return m.ResolveThreadErr
	}
	return nil
}

// MockPostReviewComment is a test helper that calls PostReviewComment with the mock client.
func (m *MockClient) MockPostReviewComment(ctx context.Context, ref model.PRRef, headSHA, path, body, side string, line int) (PostedComment, error) {
	m.PostCommentCalls++
	if m.PostCommentErr != nil {
		return PostedComment{}, m.PostCommentErr
	}
	return m.MockPostedComment, nil
}

// NewMockClient creates a new MockClient with default values.
func NewMockClient() *MockClient {
	return &MockClient{
		MockPRs:     []model.PR{},
		MockThreads: []model.ReviewThread{},
		MockFiles:   []model.PRFile{},
		MockPostedComment: PostedComment{
			ReviewID:  1,
			CommentID: 1,
		},
	}
}
