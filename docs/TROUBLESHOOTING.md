# Troubleshooting Guide

This guide helps diagnose and resolve common issues with bad-vibes (`bv`).

## Authentication Issues

### "no GitHub token found"

**Error:**
```
no GitHub token found
  • run `gh auth login` to authenticate via the GitHub CLI, or
  • set the GITHUB_TOKEN environment variable
```

**Solutions:**

1. **Using GitHub CLI (recommended):**
   ```sh
   gh auth login
   # Follow the prompts
   ```

2. **Using environment variable:**
   ```sh
   export GITHUB_TOKEN=ghp_your_token_here
   # Add to your shell profile for persistence
   ```

3. **Verify token is working:**
   ```sh
   gh auth status
   ```

### "bad credentials"

**Causes:**
- Token has expired
- Token was revoked
- Incorrect token copied

**Solutions:**
1. Clear the token cache:
   ```sh
   rm ~/.cache/bad-vibes/token
   ```
2. Re-authenticate:
   ```sh
   gh auth login
   ```

### "rate limit exceeded"

**Error:**
```
GitHub API rate limit exceeded (rate limit resets at ...)
```

**Solutions:**
1. Wait for the rate limit to reset (shown in error message)
2. Use a personal access token (higher limits than unauthenticated)
3. Reduce frequency of API calls

**Check your rate limit:**
```sh
curl -H "Authorization: Bearer $GITHUB_TOKEN" https://api.github.com/rate_limit
```

## Git Repository Issues

### "could not read git remote: not inside a git repo"

**Causes:**
- Not in a git repository
- Git not installed

**Solutions:**
1. Navigate to a git repository:
   ```sh
   cd /path/to/your/repo
   ```
2. Initialize git if needed:
   ```sh
   git init
   git remote add origin https://github.com/owner/repo.git
   ```

### "origin remote is not a GitHub URL"

**Causes:**
- Remote points to GitLab, Bitbucket, or self-hosted Git
- Remote URL format not recognized

**Solutions:**
1. Check your remote:
   ```sh
   git remote -v
   ```
2. Update to GitHub URL:
   ```sh
   git remote set-url origin https://github.com/owner/repo.git
   ```

### "not on a branch (detached HEAD?)"

**Causes:**
- Checked out a specific commit, not a branch

**Solutions:**
1. Switch to a branch:
   ```sh
   git checkout your-branch
   # or
   git checkout -b new-branch
   ```

## PR Detection Issues

### "no open PR found for branch"

**Causes:**
- No PR exists for current branch
- PR is on a different branch
- PR is closed/merged

**Solutions:**
1. List all PRs:
   ```sh
   bv prs --all-branches
   ```
2. Specify PR explicitly:
   ```sh
   bv summary --pr 42
   ```
3. Create a PR if none exists

## Comment/Thread Issues

### "anchor not saved: could not resolve posted thread ID"

**Causes:**
- Thread not yet visible in GraphQL API (eventual consistency)
- Race condition between posting and fetching

**Solutions:**
1. Wait a few seconds and try again
2. Manually add anchor to cache file:
   ```sh
   # Edit ~/.cache/bad-vibes/owner/repo/pr-number.json
   ```

### "multiple unresolved threads match file.go:42"

**Causes:**
- Multiple threads on same line
- Body required to disambiguate

**Solutions:**
1. Include unique text in comment body when posting
2. Use interactive mode to select correct thread:
   ```sh
   bv resolve
   ```

### "no unresolved thread found for anchor #perf"

**Causes:**
- Thread was already resolved
- Thread location changed (force push)
- Anchor cache stale

**Solutions:**
1. List anchors to verify:
   ```sh
   bv anchors
   ```
2. Remove stale anchor (edit cache file manually)
3. Resolve thread directly by ID:
   ```sh
   bv comments  # Find thread ID
   bv resolve --id PRRT_abc123
   ```

## TUI Issues

### "terminal not supported" or display issues

**Causes:**
- Terminal doesn't support ANSI colors
- Terminal too small

**Solutions:**
1. Use a modern terminal (iTerm2, Alacritty, Kitty, etc.)
2. Increase terminal size
3. Set `TERM` environment variable:
   ```sh
   export TERM=xterm-256color
   ```

### TUI freezes or becomes unresponsive

**Causes:**
- Network timeout
- Large PR with many threads

**Solutions:**
1. Press `Ctrl+C` to exit
2. Check network connectivity
3. Use non-interactive commands instead:
   ```sh
   bv comments --json  # If implemented
   ```

## Performance Issues

### Slow command execution

**Causes:**
- Large PR with many files/threads
- Slow network connection
- GitHub API latency

**Solutions:**
1. Check your network connection
2. Use specific PR number instead of auto-detect
3. Filter threads in interactive mode

### High memory usage

**Causes:**
- Very large diff
- Many threads loaded

**Solutions:**
1. View diff in pager:
   ```sh
   bv diff | less -R
   ```
2. Close and reopen terminal

## Cache Issues

### Corrupted cache

**Symptoms:**
- Unexpected JSON errors
- Missing anchors

**Solutions:**
1. Clear cache:
   ```sh
   rm -rf ~/.cache/bad-vibes
   ```
2. Cache will be recreated on next use

### Token cache not refreshing

**Symptoms:**
- Authentication errors after token should have refreshed

**Solutions:**
1. Manually clear token cache:
   ```sh
   rm ~/.cache/bad-vibes/token
   ```
2. Re-authenticate:
   ```sh
   gh auth login
   ```

## Build/Installation Issues

### "command not found: bv"

**Causes:**
- Not installed
- Not in PATH

**Solutions:**
1. Install:
   ```sh
   make install
   ```
2. Ensure GOPATH/bin is in PATH:
   ```sh
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

### Build fails with dependency errors

**Solutions:**
1. Clean module cache:
   ```sh
   go clean -modcache
   ```
2. Re-download dependencies:
   ```sh
   go mod download
   ```
3. Tidy dependencies:
   ```sh
   go mod tidy
   ```

### "go version too old"

**Required:** Go 1.25 or later

**Solutions:**
1. Check version:
   ```sh
   go version
   ```
2. Update Go:
   ```sh
   # macOS
   brew upgrade go
   # Linux
   # Download from https://golang.org/dl/
   ```

## Getting More Help

### Enable verbose output

Check if `--verbose` flag is available:
```sh
bv --help
```

### Check version

```sh
bv --version
```

### View logs

If structured logging is implemented:
```sh
BV_DEBUG=1 bv summary
```

### Report a bug

1. Check existing issues: https://github.com/May1a/bad-vibes/issues
2. Create new issue with:
   - `bv --version` output
   - Steps to reproduce
   - Expected vs actual behavior
   - Error messages

### Community support

- GitHub Discussions (if enabled)
- GitHub Issues for bugs

## Quick Reference

| Problem | Command |
|---------|---------|
| Clear all cache | `rm -rf ~/.cache/bad-vibes` |
| Check auth status | `gh auth status` |
| List PRs | `bv prs --all-branches` |
| View anchors | `bv anchors` |
| Check git remote | `git remote -v` |
| Check current branch | `git branch --show-current` |
