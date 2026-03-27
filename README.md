# bad-vibes (`bv`)

A focused CLI for human-in-the-loop AI PR review.

`gh` dumps too much noise. `bv` surfaces only what matters: unresolved review threads, with tight interactive flows for commenting and resolving — nothing else.

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

---

## Auth

`bv` resolves a GitHub token in this order:

1. `GITHUB_TOKEN` environment variable
2. On-disk token cache (`~/.cache/bad-vibes/token`, refreshed hourly)
3. `gh auth token` (requires the [GitHub CLI](https://cli.github.com/))

The result of `gh auth token` is cached to disk so the subprocess only runs when the cache is cold or expired.

---

## Usage

Run `bv` from inside any git repository with a GitHub remote. The repo and current branch are auto-detected — no config files needed.

The `<PR>` argument is optional on every command. When omitted, `bv` picks the most recent open PR on your current branch.

### PR reference formats

```
bv summary 42                              # bare number
bv summary owner/repo#42                   # cross-repo
bv summary https://github.com/…/pull/42   # full URL
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

### `bv summary [PR]`

Tidy overview: title, author, state, diff stats, unresolved thread count, description, changed files.

```
bv summary
bv summary 42
```

### `bv review [PR]`

Coloured unified diff streamed to stdout.

```
bv review
bv review 42
```

Line numbers, additions in green, deletions in red, context in slate.

### `bv comments [PR]`

Show only **unresolved** review threads. Resolved ones are silently absent.

```
bv comments
bv comments 42
```

Each thread shows file, line, author, timestamp, body, and diff hunk context. PR-level threads (no file) are shown as "PR-level comment". Anchor tags are highlighted in the thread body.

### `bv comment [PR]`

Interactive TUI wizard to post an inline review comment.

```
bv comment
bv comment 42
```

Steps: pick a file → enter line number → write your comment → optionally tag it with an anchor → confirm.

On confirm the comment is posted via the GitHub API. If you set an anchor tag, `bv` immediately re-fetches the thread list to capture the real thread ID and saves the anchor locally.

### `bv resolve [PR]`

Resolve review threads.

**Interactive mode** — pick from a list of unresolved threads:

```
bv resolve
bv resolve 42
```

**Direct mode** — resolve a specific thread without the TUI:

```sh
bv resolve --id PRRT_abc123          # GraphQL node ID
bv resolve --id #perf                # anchor tag (resolved by path+line lookup)
bv resolve --id #PR                  # first unresolved PR-level thread
```

### `bv anchors [PR]`

List saved anchor tags for a PR.

```
bv anchors
bv anchors 42
```

---

## Anchors

Anchors are named bookmarks for review threads. Tag a comment during `bv comment` and you can reference it later by name:

```sh
bv comment          # tag your comment #perf during the wizard
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
