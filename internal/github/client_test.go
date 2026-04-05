package github

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type rewriteTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = t.target.Scheme
	clone.URL.Host = t.target.Host
	return t.base.RoundTrip(clone)
}

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parsing test server URL: %v", err)
	}

	return NewClient("test-token", WithHTTPClient(&http.Client{
		Timeout: 5 * time.Second,
		Transport: rewriteTransport{
			target: target,
			base:   http.DefaultTransport,
		},
	}))
}

func TestClient_RetryLogic(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading request body: %v", err)
		}
		if got, want := string(body), `{"value":"ok"}`; got != want {
			t.Fatalf("attempt %d: expected body %q, got %q", attempts, want, got)
		}
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server)

	err := client.rest(context.Background(), http.MethodPost, "/test", map[string]string{"value": "ok"}, nil, nil)
	if err != nil {
		t.Fatalf("expected retries to succeed, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestClient_RateLimitHandling(t *testing.T) {
	// We need to test the rate limit extraction, not the full REST call
	// Create a mock response
	w := httptest.NewRecorder()
	w.Header().Set("X-RateLimit-Limit", "5000")
	w.Header().Set("X-RateLimit-Remaining", "0")
	w.Header().Set("X-RateLimit-Reset", "9999999999")
	w.WriteHeader(http.StatusForbidden)
	if _, err := w.Write([]byte(`{"message": "rate limited"}`)); err != nil {
		t.Fatalf("writing response body: %v", err)
	}

	resp := w.Result()
	info := extractRateLimit(resp)

	if info.Limit != 5000 {
		t.Fatalf("expected limit 5000, got %d", info.Limit)
	}
	if info.Remaining != 0 {
		t.Fatalf("expected remaining 0, got %d", info.Remaining)
	}
	if info.Reset.Unix() != 9999999999 {
		t.Fatalf("expected reset 9999999999, got %d", info.Reset.Unix())
	}
}

func TestClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	client := newTestClient(t, server)

	err := client.rest(ctx, "GET", "/test", nil, nil, nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
}

func TestClient_RetryOnRateLimited403(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", "9999999999")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"secondary rate limit"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.rest(context.Background(), http.MethodGet, "/test", nil, nil, nil)
	if err != nil {
		t.Fatalf("expected rate-limited request to succeed after retries, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		min     time.Duration
		max     time.Duration
	}{
		{1, 225 * time.Millisecond, 275 * time.Millisecond},
		{2, 450 * time.Millisecond, 550 * time.Millisecond},
		{3, 900 * time.Millisecond, 1100 * time.Millisecond},
		{10, 4500 * time.Millisecond, 5 * time.Second}, // capped at maxBackoff
	}

	for _, tt := range tests {
		backoff := calculateBackoff(tt.attempt)
		if backoff < tt.min || backoff > tt.max {
			t.Errorf("attempt %d: backoff %v not in range [%v, %v]", tt.attempt, backoff, tt.min, tt.max)
		}
	}
}

func TestExtractRateLimit(t *testing.T) {
	tests := []struct {
		name               string
		headers            map[string]string
		statusCode         int
		wantLimit          int
		wantRemaining      int
		wantResetApprox    int64
		wantResetFromRetry bool
	}{
		{
			name: "normal response",
			headers: map[string]string{
				"X-RateLimit-Limit":     "5000",
				"X-RateLimit-Remaining": "4999",
				"X-RateLimit-Reset":     "9999999999",
			},
			statusCode:      200,
			wantLimit:       5000,
			wantRemaining:   4999,
			wantResetApprox: 9999999999,
		},
		{
			name: "429 with Retry-After",
			headers: map[string]string{
				"Retry-After": "60",
			},
			statusCode:         429,
			wantResetFromRetry: true,
		},
		{
			name:       "missing headers",
			headers:    map[string]string{},
			statusCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			for k, v := range tt.headers {
				w.Header().Set(k, v)
			}
			w.WriteHeader(tt.statusCode)

			resp := w.Result()
			info := extractRateLimit(resp)

			if tt.wantLimit > 0 && info.Limit != tt.wantLimit {
				t.Errorf("expected limit %d, got %d", tt.wantLimit, info.Limit)
			}
			if tt.wantRemaining > 0 && info.Remaining != tt.wantRemaining {
				t.Errorf("expected remaining %d, got %d", tt.wantRemaining, info.Remaining)
			}
			if tt.wantResetApprox > 0 {
				expectedReset := time.Unix(tt.wantResetApprox, 0)
				if info.Reset.Unix() != expectedReset.Unix() {
					t.Errorf("expected reset %v, got %v", expectedReset, info.Reset)
				}
			}
			if tt.wantResetFromRetry && info.Reset.IsZero() {
				t.Error("expected reset time from Retry-After header")
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name       string
		err        *APIError
		want       string
		wantPrefix bool
	}{
		{
			name: "basic error",
			err: &APIError{
				StatusCode: 404,
				Method:     "GET",
				Path:       "/repos/owner/repo/pulls/1",
				Message:    "not found",
			},
			want: "GitHub API GET /repos/owner/repo/pulls/1 returned 404: not found",
		},
		{
			name: "rate limited error",
			err: &APIError{
				StatusCode: 403,
				Method:     "POST",
				Path:       "/graphql",
				Message:    "rate limited",
				RateLimit: &RateLimitInfo{
					Remaining: 0,
					Reset:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
			},
			want:       "GitHub API POST /graphql returned 403: rate limited (rate limit resets at",
			wantPrefix: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if tt.wantPrefix {
				if !strings.HasPrefix(got, tt.want) {
					t.Errorf("expected prefix %q, got %q", tt.want, got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestGraphQLErrors_Error(t *testing.T) {
	errs := graphqlErrors{
		{Message: "error 1"},
		{Message: "error 2"},
	}

	got := errs.Error()
	want := "error 1; error 2"

	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestGraphQLErrors_Empty(t *testing.T) {
	errs := graphqlErrors{}

	got := errs.Error()
	want := "unknown GraphQL error"

	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}
