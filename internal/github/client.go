package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiBase      = "https://api.github.com"
	graphqlEndpoint = "https://api.github.com/graphql"
)

var (
	token  string
	client = &http.Client{Timeout: 30 * time.Second}
)

// SetToken stores the GitHub auth token for all subsequent API calls.
func SetToken(t string) { token = t }

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
	msg := e[0].Message
	for _, err := range e[1:] {
		msg += "; " + err.Message
	}
	return msg
}

// graphql sends a GraphQL query/mutation and unmarshals the data field into v.
func graphql(query string, variables map[string]any, v any) error {
	body, err := json.Marshal(graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, raw)
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

// rest performs a REST API call. body may be nil for GET requests.
// extraHeaders is optional additional HTTP headers.
// If v is a *string, the raw response body is written to it (useful for diff).
func rest(method, path string, body any, v any, extraHeaders map[string]string) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, apiBase+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, val := range extraHeaders {
		req.Header.Set(k, val)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, raw)
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
