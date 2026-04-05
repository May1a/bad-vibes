package cmd

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/may1a/bv/internal/diff"
)

type commentBodyInput struct {
	Body   string
	Source string
}

type commentPreflightInput struct {
	CommandPath string
	RawPath     string
	Line        int
	Side        string
	Patch       diff.Patch
	WorkingDir  string
	RepoRoot    string
}

type commentPreflightResult struct {
	Path string
	Line int
	Side string
}

func preflightCommentTarget(input commentPreflightInput) (commentPreflightResult, error) {
	path, err := normalizeCommentPath(input.CommandPath, input.RawPath, input.Line, input.Patch.Paths(), input.WorkingDir, input.RepoRoot)
	if err != nil {
		return commentPreflightResult{}, err
	}

	file, ok := input.Patch.File(path)
	if !ok {
		return commentPreflightResult{}, fmt.Errorf("could not validate comment target\n  why: %q was not found in the pull request diff\n  try: %s", path, formatPathSuggestions(input.CommandPath, path, input.Line, input.Patch.Paths()))
	}

	if file.HasCommentLine(input.Side, input.Line) {
		return commentPreflightResult{
			Path: path,
			Line: input.Line,
			Side: input.Side,
		}, nil
	}

	if file.HasCommentLine(oppositeSide(input.Side), input.Line) {
		return commentPreflightResult{}, fmt.Errorf("could not validate comment target\n  why: %s:%d is only available on the %s side of the diff\n  try: %s %s --side %s \"comment\"", path, input.Line, oppositeSide(input.Side), input.CommandPath, formatCommentLocation(path, input.Line), oppositeSide(input.Side))
	}

	validLines := file.ValidLines(input.Side)
	return commentPreflightResult{}, fmt.Errorf("could not validate comment target\n  why: %s:%d is not part of the pull request diff on the %s side\n  try: choose one of %s", path, input.Line, input.Side, summarizeLines(validLines))
}

func normalizeCommentPath(commandPath, raw string, line int, changedFiles []string, workingDir, repoRoot string) (string, error) {
	candidates := pathCandidates(raw, workingDir, repoRoot)
	fileSet := map[string]struct{}{}
	for _, changed := range changedFiles {
		fileSet[changed] = struct{}{}
	}
	for _, candidate := range candidates {
		if _, ok := fileSet[candidate]; ok {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not validate comment target\n  why: %q is not part of the pull request diff\n  try: %s", raw, formatPathSuggestions(commandPath, raw, line, changedFiles))
}

func pathCandidates(raw, workingDir, repoRoot string) []string {
	candidates := []string{}
	add := func(path string) {
		path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
		path = strings.TrimPrefix(path, "./")
		if path == "." || path == "" {
			return
		}
		if slices.Contains(candidates, path) {
			return
		}
		candidates = append(candidates, path)
	}

	add(raw)

	if repoRoot == "" {
		return candidates
	}

	rawAbs := raw
	if !filepath.IsAbs(rawAbs) && workingDir != "" {
		rawAbs = filepath.Join(workingDir, raw)
	}
	if rel, err := filepath.Rel(repoRoot, rawAbs); err == nil && rel != "" && !strings.HasPrefix(rel, "..") {
		add(rel)
	}

	return candidates
}

func formatPathSuggestions(commandPath, raw string, line int, changedFiles []string) string {
	matches := suggestPaths(raw, changedFiles)
	if len(matches) == 0 {
		return fmt.Sprintf("%s path/from/pr/diff:%d \"comment\"", commandPath, line)
	}
	if len(matches) > 3 {
		matches = matches[:3]
	}
	targets := make([]string, 0, len(matches))
	for _, match := range matches {
		targets = append(targets, formatCommentLocation(match, line))
	}
	return "use one of: " + strings.Join(targets, ", ")
}

func suggestPaths(raw string, changedFiles []string) []string {
	raw = filepath.ToSlash(filepath.Clean(strings.TrimSpace(raw)))
	base := filepath.Base(raw)
	type scored struct {
		path  string
		score int
	}
	var matches []scored
	for _, changed := range changedFiles {
		score := 0
		switch {
		case changed == raw:
			score = 100
		case filepath.Base(changed) == base:
			score = 80
		case strings.HasSuffix(changed, "/"+raw):
			score = 70
		case strings.Contains(changed, raw) || strings.Contains(raw, changed):
			score = 60
		case strings.Contains(filepath.Base(changed), base):
			score = 50
		}
		if score > 0 {
			matches = append(matches, scored{path: changed, score: score})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score == matches[j].score {
			return matches[i].path < matches[j].path
		}
		return matches[i].score > matches[j].score
	})
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		duplicate := slices.Contains(paths, match.path)
		if !duplicate {
			paths = append(paths, match.path)
		}
	}
	return paths
}

func summarizeLines(lines []int) string {
	if len(lines) == 0 {
		return "a line that appears in the diff"
	}
	if len(lines) > 6 {
		lines = lines[:6]
	}
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		parts = append(parts, fmt.Sprintf("%d", line))
	}
	return strings.Join(parts, ", ")
}

func oppositeSide(side string) string {
	if strings.EqualFold(side, "LEFT") {
		return "RIGHT"
	}
	return "LEFT"
}
