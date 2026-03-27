# Architecture Documentation

This document describes the internal architecture of bad-vibes (`bv`).

## Overview

bad-vibes is a CLI tool for focused AI-assisted PR review. It provides a streamlined interface for viewing PR diffs, managing review comments, and resolving threads.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         User CLI                                 │
│                        (cobra commands)                          │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Command Layer (cmd/)                        │
│  • root.go      - Initialization, auth, repo detection          │
│  • prs.go       - List PRs                                       │
│  • summary.go   - PR overview                                    │
│  • review.go    - Display diff                                   │
│  • comments.go  - Show unresolved threads                        │
│  • comment.go   - Interactive comment wizard                     │
│  • resolve.go   - Resolve threads                                │
│  • anchors.go   - List anchors                                   │
└─────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│   TUI Layer      │  │  Display Layer   │  │   Git Layer      │
│   (tui/)         │  │  (display/)      │  │   (git/)         │
│  • bubbletea     │  │  • lipgloss      │  │  • git commands  │
│    components    │  │  • formatting    │  │  • remote parse  │
└──────────────────┘  └──────────────────┘  └──────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    GitHub API Client (github/)                   │
│  • client.go    - HTTP client with retry & rate limit handling  │
│  • pr.go        - Fetch PR metadata (GraphQL)                   │
│  • prs.go       - List PRs (GraphQL)                            │
│  • threads.go   - Fetch/resolve threads (GraphQL)               │
│  • post.go      - Post comments (REST)                          │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      External Services                           │
│  • GitHub GraphQL API  - api.github.com/graphql                 │
│  • GitHub REST API     - api.github.com                         │
└─────────────────────────────────────────────────────────────────┘
```

## Component Details

### Command Layer (`cmd/`)

Uses [cobra](https://github.com/spf13/cobra) for CLI structure. Each command:
- Parses flags and arguments
- Calls appropriate internal packages
- Handles output formatting

**Key patterns:**
- `resolvePR()` - Common function to resolve PR reference from args or auto-detect
- `repoRef()` - Get owner/repo from git remote
- Persistent pre-run hook for auth and repo detection

### GitHub API Client (`internal/github/`)

**Client Structure:**
```go
type Client struct {
    token      string
    httpClient *http.Client
}
```

**Features:**
- Exponential backoff retry logic (max 3 retries)
- Rate limit detection and handling
- Context support for cancellation
- Unified error types (`APIError`)

**API Endpoints Used:**

| Operation | API Type | Endpoint |
|-----------|----------|----------|
| Fetch PR | GraphQL | `repository.pullRequest` |
| List PRs | GraphQL | `repository.pullRequests` |
| Review Threads | GraphQL | `pullRequest.reviewThreads` |
| Resolve Thread | GraphQL | `resolveReviewThread` mutation |
| Post Comment | REST | `POST /repos/:owner/:repo/pulls/:number/reviews` |
| Fetch Diff | REST | `GET /repos/:owner/:repo/pulls/:number` (Accept: application/vnd.github.diff) |

### TUI Layer (`internal/tui/`)

Built with [bubbletea](https://github.com/charmbracelet/bubbletea) and [bubbles](https://github.com/charmbracelet/bubbles).

**Components:**
- `CommentModel` - Multi-step comment wizard
- `ResolveModel` - Interactive thread resolver

**State Machine (Comment Flow):**
```
stepFile → stepLine → stepBody → stepAnchor → stepConfirm → Done
```

### Cache Layer (`internal/cache/`)

**Storage Location:** `~/.cache/bad-vibes/`

**Structure:**
```
~/.cache/bad-vibes/
├── token                          # GitHub token (1h TTL)
└── <owner>/<repo>/<pr-number>.json  # PR anchor data
```

**PR Anchor Cache Schema:**
```json
{
  "Owner": "may1a",
  "Repo": "bad-vibes",
  "Number": 42,
  "PRID": "PR_kwDOABC123",
  "HeadSHA": "abc123...",
  "Anchors": [
    {
      "Tag": "perf",
      "ThreadID": "PRRT_xyz789",
      "Path": "cmd/root.go",
      "Line": 42,
      "Body": "Performance concern...",
      "Created": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### Auth Layer (`internal/auth/`)

**Token Resolution Order:**
1. `GITHUB_TOKEN` environment variable (immediate)
2. Disk cache (`~/.cache/bad-vibes/token`, 1h TTL)
3. `gh auth token` subprocess (cached on success)

**Security:**
- Token file permissions: `0600`
- Cache directory permissions: `0700`

### Display Layer (`internal/display/`)

**Responsibilities:**
- Colored diff output (additions green, deletions red)
- Thread rendering with anchor highlighting
- Line number formatting

**Styling:**
- Uses [lipgloss](https://github.com/charmbracelet/lipgloss) for terminal styling
- Consistent color palette across commands

### Parse Layer (`internal/parse/`)

**Supported PR Reference Formats:**
- `42` - Bare number (requires default repo)
- `owner/repo#42` - Short form
- `https://github.com/owner/repo/pull/42` - Full URL

## Data Flow Examples

### `bv comments` Flow

```
User runs: bv comments
    │
    ▼
cmd/comments.go:RunE
    │
    ├─► resolvePR() ─► github.LatestOpenPR() ─► FetchPRs()
    │                                             │
    │                                             ▼
    │                                    GitHub GraphQL API
    │
    ▼
FetchReviewThreads()
    │
    ▼
GitHub GraphQL API (paginated)
    │
    ▼
Filter unresolved threads
    │
    ▼
display.PrintThreads()
    │
    ▼
Formatted terminal output
```

### `bv comment` Flow

```
User runs: bv comment
    │
    ▼
tui.RunCommentFlow()
    │
    ├─► stepFile: Select file (bubbletea list)
    ├─► stepLine: Enter line number
    ├─► stepBody: Write comment
    ├─► stepAnchor: Optional anchor tag
    └─► stepConfirm: Review and post
            │
            ▼
    github.PostReviewComment() (REST)
            │
            ▼
    GitHub API creates comment
            │
            ▼
    FindUnresolvedThreadAt() (GraphQL)
            │
            ▼
    cache.AddAnchor() (if tagged)
```

## Error Handling Strategy

**Error Types:**
- `APIError` - GitHub API errors with status code and rate limit info
- `ErrRateLimited` - Sentinel error for rate limit exceeded
- `ErrTimeout` - Sentinel error for request timeout

**Wrapping Pattern:**
```go
if err != nil {
    return fmt.Errorf("fetching PR #%d: %w", ref.Number, err)
}
```

**Retry Logic:**
```go
for attempt := 0; attempt < maxRetries; attempt++ {
    if attempt > 0 {
        time.Sleep(calculateBackoff(attempt))
    }
    err := c.doRequest(...)
    if err == nil {
        return nil
    }
    if !isRetryable(err) {
        return err
    }
}
```

## Testing Strategy

**Test Layers:**
1. **Unit tests** - Individual functions (e.g., `parse.ParseRef`)
2. **Integration tests** - API client with mock server
3. **Mock client** - `MockClient` for testing without API calls

**Test Files:**
- `internal/github/client_test.go` - Client retry, rate limit, timeout
- `internal/github/threads_test.go` - Thread resolution logic
- `internal/cache/cache_test.go` - Cache operations
- `internal/auth/auth_test.go` - Token resolution
- `internal/parse/ref_test.go` - PR reference parsing

## Performance Considerations

**Optimization Strategies:**
- Pagination handled automatically for large result sets
- Context-based cancellation for long-running operations
- Token caching reduces `gh` subprocess calls
- Local anchor cache avoids repeated API lookups

**Known Limitations:**
- PR file list truncated at 100 files
- Review threads fetched with 50 per page
- No concurrent fetching (sequential API calls)

## Security Considerations

**Token Handling:**
- Tokens stored with restrictive file permissions
- 1-hour cache TTL limits exposure window
- Token never logged or printed

**Input Validation:**
- PR references validated before API calls
- File paths and line numbers validated in TUI

## Future Architecture Improvements

See GitHub issues for planned improvements including:
- Dependency injection for easier testing
- Structured logging
- Configuration file support
- Concurrent API fetching
- Plugin architecture for extensibility
