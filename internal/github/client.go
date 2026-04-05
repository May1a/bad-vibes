package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	apiBase         = "https://api.github.com"
	graphqlEndpoint = "https://api.github.com/graphql"

	// Retry configuration
	maxRetries     = 3
	initialBackoff = 250 * time.Millisecond
	maxBackoff     = 5 * time.Second
	defaultTimeout = 30 * time.Second

	// Rate limit headers
	headerRateLimit     = "X-RateLimit-Limit"
	headerRateRemaining = "X-RateLimit-Remaining"
	headerRateReset     = "X-RateLimit-Reset"
	headerRetryAfter    = "Retry-After"
)

var (
	// ErrRateLimited is returned when the API rate limit is exceeded.
	ErrRateLimited = errors.New("GitHub API rate limit exceeded")

	// ErrTimeout is returned when a request times out.
	ErrTimeout = errors.New("request timed out")

	// defaultClient is the package-level client used by legacy functions.
	defaultClient *Client
)

// SetClient sets the default client for package-level functions.
// This is used for backward compatibility with code that doesn't pass client explicitly.
func SetClient(c *Client) {
	defaultClient = c
}

// GetClient returns the default client.
// Panics if no client has been set.
func GetClient() *Client {
	if defaultClient == nil {
		panic("github: no client set; call SetClient first")
	}
	return defaultClient
}

// Client is a GitHub API client with retry logic and rate limit handling.
type Client struct {
	token      string
	httpClient *http.Client
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// NewClient creates a new GitHub API client.
func NewClient(token string, opts ...ClientOption) *Client {
	c := &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// graphqlRequest is the JSON body for a GraphQL request.
type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphqlErrors captures the top-level errors array in a GraphQL response.
type graphqlErrors []struct {
	Message string `json:"message"`
}

func (e graphqlErrors) Error() string {
	if len(e) == 0 {
		return "unknown GraphQL error"
	}
	var msg strings.Builder
	msg.WriteString(e[0].Message)
	for _, err := range e[1:] {
		msg.WriteString("; " + err.Message)
	}
	return msg.String()
}

// RateLimitInfo contains rate limit information from API responses.
type RateLimitInfo struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

// APIError represents a GitHub API error with context.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Message    string
	RateLimit  *RateLimitInfo
}

// RateLimitError wraps APIError so callers can match both ErrRateLimited and APIError.
type RateLimitError struct {
	APIError *APIError
}

func (e *RateLimitError) Error() string {
	if e.APIError == nil {
		return ErrRateLimited.Error()
	}
	return ErrRateLimited.Error() + ": " + e.APIError.Error()
}

func (e *RateLimitError) Unwrap() []error {
	if e.APIError == nil {
		return []error{ErrRateLimited}
	}
	return []error{ErrRateLimited, e.APIError}
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("GitHub API %s %s returned %d", e.Method, e.Path, e.StatusCode)
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if e.RateLimit != nil && e.RateLimit.Remaining == 0 {
		msg += fmt.Sprintf(" (rate limit resets at %s)", e.RateLimit.Reset.Format(time.RFC822))
	}
	return msg
}

// Unwrap returns the underlying error message for errors.Is/As.
func (e *APIError) Unwrap() error {
	return errors.New(e.Message)
}

// graphql sends a GraphQL query/mutation with retry logic and rate limit handling.
func (c *Client) graphql(ctx context.Context, query string, variables map[string]any, v any) error {
	body, err := json.Marshal(graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("marshaling GraphQL request: %w", err)
	}

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			backoff := calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := c.doGraphQL(ctx, body, v)
		if err == nil {
			return nil
		}

		lastErr = err

		if errors.Is(err, ErrRateLimited) {
			continue
		}

		// Check if error is retryable.
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			if apiErr.StatusCode >= 500 {
				continue // Retry
			}
		}

		// Non-retryable error
		return err
	}

	return fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}

// doGraphQL performs a single GraphQL request.
func (c *Client) doGraphQL(ctx context.Context, body []byte, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %v", ErrTimeout, ctx.Err())
		}
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	rateLimit := extractRateLimit(resp)

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Method:     "POST",
			Path:       "/graphql",
			Message:    string(raw),
			RateLimit:  rateLimit,
		}
		if isRateLimitedResponse(resp, raw) {
			return &RateLimitError{APIError: apiErr}
		}
		return apiErr
	}

	// GraphQL always returns 200; errors live in the errors field.
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors graphqlErrors   `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return envelope.Errors
	}
	if v != nil {
		return json.Unmarshal(envelope.Data, v)
	}
	return nil
}

// rest performs a REST API call with retry logic and rate limit handling.
// body may be nil for GET requests.
// extraHeaders is optional additional HTTP headers.
// If v is a *string, the raw response body is written to it (useful for diff).
func (c *Client) rest(ctx context.Context, method, path string, body any, v any, extraHeaders map[string]string) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyBytes = b
	}

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			backoff := calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		err := c.doREST(ctx, method, path, bodyReader, v, extraHeaders)
		if err == nil {
			return nil
		}

		lastErr = err

		if errors.Is(err, ErrRateLimited) {
			continue
		}

		// Check if error is retryable.
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			if apiErr.StatusCode >= 500 {
				continue // Retry
			}
		}

		// Non-retryable error
		return err
	}

	return fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}

// doREST performs a single REST request.
func (c *Client) doREST(ctx context.Context, method, path string, bodyReader io.Reader, v any, extraHeaders map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, method, apiBase+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, val := range extraHeaders {
		req.Header.Set(k, val)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %v", ErrTimeout, ctx.Err())
		}
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	rateLimit := extractRateLimit(resp)

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Method:     method,
			Path:       path,
			Message:    string(raw),
			RateLimit:  rateLimit,
		}
		if isRateLimitedResponse(resp, raw) {
			return &RateLimitError{APIError: apiErr}
		}
		return apiErr
	}

	if v == nil {
		return nil
	}
	if s, ok := v.(*string); ok {
		*s = string(raw)
		return nil
	}
	return json.Unmarshal(raw, v)
}

// calculateBackoff returns the backoff duration for a given attempt using exponential backoff.
func calculateBackoff(attempt int) time.Duration {
	backoff := min(initialBackoff*time.Duration(1<<uint(attempt-1)), maxBackoff)

	// Add symmetric jitter (±10%) to avoid synchronized retries.
	jitter := backoff / 10
	if jitter > 0 {
		delta := rand.Int64N(int64(jitter)*2+1) - int64(jitter)
		backoff += time.Duration(delta)
		if backoff < 0 {
			backoff = 0
		}
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
	return backoff
}

func isRateLimitedResponse(resp *http.Response, raw []byte) bool {
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	if resp.StatusCode != http.StatusForbidden {
		return false
	}
	if resp.Header.Get(headerRetryAfter) != "" {
		return true
	}
	if resp.Header.Get(headerRateRemaining) == "0" {
		return true
	}

	msg := strings.ToLower(string(raw))
	return strings.Contains(msg, "rate limit")
}

// extractRateLimit extracts rate limit information from response headers.
func extractRateLimit(resp *http.Response) *RateLimitInfo {
	info := &RateLimitInfo{}

	if limit := resp.Header.Get(headerRateLimit); limit != "" {
		info.Limit, _ = strconv.Atoi(limit)
	}
	if remaining := resp.Header.Get(headerRateRemaining); remaining != "" {
		info.Remaining, _ = strconv.Atoi(remaining)
	}
	if reset := resp.Header.Get(headerRateReset); reset != "" {
		if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
			info.Reset = time.Unix(ts, 0)
		}
	}

	// Handle Retry-After header for 429 responses
	if resp.StatusCode == 429 {
		if retryAfter := resp.Header.Get(headerRetryAfter); retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				info.Reset = time.Now().Add(time.Duration(seconds) * time.Second)
			}
		}
	}

	return info
}

// GetRateLimit returns the current rate limit status (GraphQL-only).
func (c *Client) GetRateLimit(ctx context.Context) (*RateLimitInfo, error) {
	query := `query { rateLimit { limit remaining resetAt } }`
	var data struct {
		RateLimit struct {
			Limit     int    `json:"limit"`
			Remaining int    `json:"remaining"`
			ResetAt   string `json:"resetAt"`
		} `json:"rateLimit"`
	}

	if err := c.graphql(ctx, query, nil, &data); err != nil {
		return nil, err
	}

	resetAt, _ := time.Parse(time.RFC3339, data.RateLimit.ResetAt)
	return &RateLimitInfo{
		Limit:     data.RateLimit.Limit,
		Remaining: data.RateLimit.Remaining,
		Reset:     resetAt,
	}, nil
}
