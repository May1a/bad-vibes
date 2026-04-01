# bad-vibes (`bv`)

A focused CLI for human-in-the-loop AI PR review.

`gh` dumps too much noise. `bv` surfaces only what matters: unresolved review threads, direct commenting, and fast resolution flows — nothing else.

## Features

- 🎯 **Focused** - Only unresolved threads, no noise
- 🧰 **Direct CLI** - Scriptable commenting plus fast thread resolution
- ⚡ **Fast** - Retry logic, rate limit handling, and caching
- 🔗 **Anchors** - Named bookmarks for important threads
- 🛡️ **Robust** - Comprehensive error handling and timeout support

---

## Install

**From source (requires Go 1.21+):**

```sh
git clone https://github.com/May1a/bad-vibes
cd bad-vibes
make install   # installs bv to $GOPATH/bin
```

**Build locally:**

```sh
make build     # produces ./bv
```

**Cross-compile:**

```sh
make build-all # outputs to dist/
```

**As a `skills.sh` skill:**

```sh
npx skills add https://github.com/May1a/bad-vibes --skill bv
```

---

## Auth

`bv` resolves a GitHub token in this order:

1. `GITHUB_TOKEN` environment variable
2. On-disk token cache (`~/.cache/bad-vibes/token`, refreshed hourly)
3. `gh auth token` (requires the [GitHub CLI](https://cli.github.com/))

The result of `gh auth token` is cached to disk so the subprocess only runs when the cache is cold or expired.

---

## Usage

Run `bv` from inside any git repository with a GitHub remote. The repo and current branch are auto-detected, and every repo-scoped command defaults to the newest open PR on the current branch.

### PR reference formats

```
bv summary --pr 42                         # bare number
bv summary --pr owner/repo#42             # cross-repo
bv summary --pr https://github.com/…/pull/42
bv summary                                 # auto-detect from current branch
```

---

## Commands

### `bv prs`

List pull requests for the current repo.

```
bv prs                    # open PRs on current branch
bv prs --all-branches     # open PRs across all branches
bv prs --branch feat/x    # open PRs on a specific branch
bv prs --closed           # closed and merged PRs
```

### `bv summary`

Tidy overview: title, author, state, diff stats, unresolved thread count, description, and changed files with per-file additions/deletions plus created/deleted markers.

```
bv summary
bv summary --pr 42
```

### `bv diff`

Coloured unified diff streamed to stdout.

```
bv diff
bv diff --pr 42
```

Line numbers, additions in green, deletions in red, context in slate.

### `bv comments`

Show only **unresolved** review threads. Resolved ones are silently absent.

```
bv comments
bv comments --pr 42
bv comments --verbose --patch
```

By default `bv comments` prints a compact summary for each unresolved thread plus a code snippet. Use `--verbose` to show every comment in the thread and `--patch` to include the full diff hunk. PR-level threads (no file) are shown as "PR-level comment".

### `bv comment`

Post an inline review comment directly from the CLI.

```
bv comment cmd/root.go:42 "Needs a guard here"
bv comment cmd/root.go:42 --body-file ./comment.md --anchor perf
printf 'Needs a guard here\n' | bv comment cmd/root.go:42
```

Required inputs: `<file>:<line>` and a body via the optional 2nd argument, `--body`, `--body-file`, or stdin. If you set `--anchor`, `bv` retries briefly until the exact posted thread is visible in GitHub and only then saves the anchor locally.

### `bv resolve`

Resolve review threads.

Without `--id`, `bv resolve` resolves the first unresolved thread in the same order shown by `bv comments`.

```sh
bv resolve
bv resolve --id PRRT_abc123          # GraphQL node ID
bv resolve --id #perf                # anchor tag (resolved by path+line lookup)
bv resolve --id #PR                  # first unresolved PR-level thread
```

### `bv anchors`

List saved anchor tags for a PR.

```
bv anchors
```

---

## Anchors

Anchors are named bookmarks for review threads. Tag a comment during `bv comment` and you can reference it later by name:

```sh
bv comment cmd/root.go:42 "Needs work" --anchor perf
bv resolve --id #perf   # resolve that thread by name
```

Anchors work like symlinks: they store the file path and line number, not a raw thread ID. When you dereference `#perf`, `bv` looks up the current live thread at that location — so they stay valid even if the thread ID changes between fetches.

PR-level threads (not attached to a file) can be resolved with the special `#PR` shorthand.

Anchor data is stored locally at `~/.cache/bad-vibes/<owner>/<repo>/<pr>.json`.

---

## Cache

```
~/.cache/bad-vibes/
  token                          # GitHub auth token (1h TTL)
  <owner>/<repo>/<pr>.json       # PR anchor cache
```

---

## Development

```sh
make test          # go test ./...
make test-verbose  # with -race
make lint          # golangci-lint
make tidy          # go mod tidy
```

### Running Tests

```sh
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/github/...
```

### Building

```sh
# Development build
make build

# Install to GOPATH/bin
make install

# Cross-compile for all platforms
make build-all
```

---

## Documentation

- [Troubleshooting](docs/TROUBLESHOOTING.md) - Common issues and solutions
- [API Scopes](docs/API_SCOPES.md) - Required GitHub permissions
- [Contributing](CONTRIBUTING.md) - How to contribute

---

## License

MIT License - see [LICENSE](LICENSE) for details.
