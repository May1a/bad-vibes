# Contributing to bad-vibes

Thank you for your interest in contributing to bad-vibes (`bv`)! This document provides guidelines and instructions for contributing.

## Getting Started

### Prerequisites

- Go 1.25 or later
- GitHub CLI (`gh`) for authentication (optional but recommended)
- Git

### Setting Up Your Development Environment

1. **Fork and clone the repository:**
   ```sh
   git clone https://github.com/YOUR_USERNAME/bad-vibes.git
   cd bad-vibes
   ```

2. **Install dependencies:**
   ```sh
   go mod download
   ```

3. **Build the CLI:**
   ```sh
   make build
   ```

4. **Run tests:**
   ```sh
   make test
   ```

## Development Workflow

### Making Changes

1. **Create a branch:**
   ```sh
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the coding guidelines below.

3. **Run tests and linting:**
   ```sh
   make test
   make lint
   make tidy
   ```

4. **Commit your changes** with clear, descriptive commit messages.

5. **Push and open a pull request.**

### Coding Guidelines

#### Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines.
- Use `gofmt` or `goimports` for formatting (enforced by linter).
- Keep functions small and focused on a single responsibility.
- Prefer early returns over deep nesting.

#### Error Handling

- Wrap errors with context using `fmt.Errorf("context: %w", err)`.
- Define sentinel errors for expected error conditions.
- Never ignore errors explicitly (no `_ = func()` unless documented).

#### Testing

- Write tests for all new functionality.
- Use table-driven tests where appropriate.
- Mock external dependencies (GitHub API) using `MockClient`.
- Aim for high test coverage, but prioritize meaningful tests.

#### Documentation

- Add godoc comments for exported types and functions.
- Update README.md for user-facing changes.
- Include examples in documentation where helpful.

## Pull Request Process

1. **Ensure your PR description clearly describes:**
   - What problem is being solved
   - How the changes solve it
   - Any breaking changes or migration notes

2. **Link related issues** using `Fixes #123` or `Closes #456`.

3. **Request review** from maintainers.

4. **Address feedback** promptly and push updates.

5. **Squash commits** if requested by maintainers.

## Architecture Overview

```
bad-vibes/
├── cmd/              # CLI command definitions (cobra commands)
├── internal/
│   ├── auth/         # GitHub token resolution
│   ├── cache/        # Local cache for anchors and tokens
│   ├── display/      # Terminal output formatting
│   ├── git/          # Git operations (remote, branch detection)
│   ├── github/       # GitHub API client (GraphQL + REST)
│   ├── model/        # Data structures
│   ├── parse/        # PR reference parsing
│   └── tui/          # Interactive TUI components (bubbletea)
└── main.go           # Entry point
```

### Key Design Decisions

- **GitHub API Client**: Uses both GraphQL (for threads, PRs) and REST (for diff, posting comments) APIs.
- **Retry Logic**: All API calls include exponential backoff and rate limit handling.
- **Anchors**: Local symlinks to review threads, stored in `~/.cache/bad-vibes/`.
- **TUI**: Built with `bubbletea` for interactive flows.

## Reporting Issues

### Bug Reports

Include:
- `bv --version` output
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or error messages

### Feature Requests

Include:
- Problem statement
- Proposed solution
- Use cases
- Alternatives considered

## Release Process

Releases are managed by maintainers. The process:

1. Version bump in code (if applicable)
2. Update CHANGELOG.md
3. Create git tag
4. GitHub Actions builds and publishes binaries

## Code of Conduct

- Be respectful and inclusive.
- Focus on constructive feedback.
- Welcome contributors of all experience levels.

## Questions?

- Open an issue for general questions.
- Check existing issues and documentation first.

---

Thank you for contributing to bad-vibes! 🎉
