---
name: bv
description: This skill should be used when the user asks to "use bv", "review a PR with bv", "post a comment with bv", "show comments with bv", "resolve a thread with bv", "list PRs with bv", "show unresolved comments", "use bad-vibes", "create an anchor", "resolve by anchor", or asks how any `bv` command works.
version: 0.2.0
---

# bv — bad-vibes CLI

`bv` is a focused CLI for human-in-the-loop GitHub PR review. It surfaces unresolved review threads, supports direct inline commenting from the shell, and keeps anchor-based resolution fast.

## Core Concepts

- **Auto-detection**: Every command auto-detects the current repo and branch from git. The `[PR]` argument is optional.
- **PR reference formats**: Commands accept a bare number (`42`), cross-repo reference (`owner/repo#42`), or full GitHub URL.
- **Unresolved-only**: `bv comments` and `bv resolve` operate on unresolved threads. Resolved threads are silently excluded.
- **Anchors**: Named bookmarks for threads (for example `#perf`) persist locally and resolve by file+line, not only by raw GraphQL ID.
- **Direct commenting**: `bv comment` is a normal CLI command now, not a TUI flow. Pass file, line, and body explicitly.

## Commands at a Glance

| Command | Purpose |
|---|---|
| `bv prs` | List PRs for this repo |
| `bv summary [PR]` | Show PR metadata plus unresolved thread count |
| `bv review [PR]` | Show the colored unified diff |
| `bv comments [PR]` | Show unresolved review threads |
| `bv comment [PR]` | Post an inline comment directly from flags/stdin |
| `bv resolve [PR]` | Resolve threads interactively or directly by ID/anchor |
| `bv anchors [PR]` | List saved anchor tags for a PR |

## Auth

Token resolution order:
1. `GITHUB_TOKEN` env var
2. `~/.cache/bad-vibes/token` (1-hour TTL, auto-refreshed)
3. `gh auth token` (GitHub CLI)

## Command Usage

### `bv comments`

Default output is compact. It prints one summary per unresolved thread instead of dumping every comment body and diff hunk by default.

Use:

```bash
bv comments
bv comments 42
bv comments --verbose
bv comments --verbose --patch
```

Interpret flags as follows:

- `--verbose`: show every comment in each thread
- `--patch`: include diff hunk context

When asked to inspect unresolved feedback, prefer starting with plain `bv comments` and only add `--verbose` or `--patch` if the compact output is insufficient.

### `bv comment`

`bv comment` is shell-friendly and non-interactive.

Required inputs:

- `--file PATH`
- `--line N`
- comment body via `--body TEXT`, `--body-file FILE`, or stdin

Examples:

```bash
bv comment --file cmd/root.go --line 42 --body "Needs a guard here"
bv comment 42 --file cmd/root.go --line 42 --body-file ./comment.md --anchor perf
printf 'Needs a guard here\n' | bv comment --file cmd/root.go --line 42
```

Optional flags:

- `--anchor TAG`: save a local anchor for the new thread
- `--side LEFT|RIGHT`: choose diff side, default `RIGHT`

When asked to post a comment with `bv`, provide all required flags explicitly. Do not describe or expect a wizard.

### `bv resolve`

Two modes exist:

```bash
bv resolve                    # interactive picker
bv resolve --id PRRT_abc123   # resolve by GraphQL node ID
bv resolve --id #perf         # resolve by anchor tag
bv resolve --id #PR           # resolve first unresolved PR-level thread
```

Prefer direct `--id` resolution when the user already knows the target thread or anchor. Use interactive resolve only when selection from a list is actually needed.

## Anchors Workflow

Anchors act like named symlinks to review threads:

```bash
# 1. Post a comment and tag it
bv comment --file cmd/root.go --line 42 --body "Needs work" --anchor perf

# 2. Later, resolve it by name
bv resolve --id #perf

# 3. Special: resolve the first unresolved PR-level thread
bv resolve --id #PR

# 4. List all anchors for the current PR
bv anchors
```

Anchors are stored at `~/.cache/bad-vibes/<owner>/<repo>/<pr>.json` as file path + line number, so they remain valid even if the underlying thread ID changes.

## Common Workflows

### Review a PR end-to-end

```bash
bv summary
bv review
bv comments
```

### Respond to review threads

```bash
bv comments
bv comment --file path/to/file.go --line 42 --body "reply text"
bv resolve --id PRRT_abc123
```

### Inspect a noisy thread in more detail

```bash
bv comments --verbose
bv comments --verbose --patch
```
