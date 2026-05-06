package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/may1a/bad-vibes/internal/model"
)

// commentLocation is a parsed <file>:<line> target.
type commentLocation struct {
	Path string
	Line int
}

func parseCommentLocation(commandPath, raw string) (commentLocation, error) {
	raw = strings.TrimSpace(raw)
	idx := strings.LastIndex(raw, ":")
	if idx <= 0 || idx == len(raw)-1 {
		return commentLocation{}, fmt.Errorf("could not build review comment\n  why: expected <file>:<line>, got %q\n  try: %s path/from/diff:42 \"comment\"", raw, commandPath)
	}
	path := strings.TrimSpace(raw[:idx])
	lineText := strings.TrimSpace(raw[idx+1:])
	line, err := strconv.Atoi(lineText)
	if err != nil {
		return commentLocation{}, fmt.Errorf("could not build review comment\n  why: %q must end with a numeric line number\n  try: %s path/from/diff:42 \"comment\"", raw, commandPath)
	}
	if path == "" || line < 1 {
		return commentLocation{}, fmt.Errorf("could not build review comment\n  why: expected <file>:<line>, got %q\n  try: %s path/from/diff:42 \"comment\"", raw, commandPath)
	}
	return commentLocation{Path: path, Line: line}, nil
}

func formatCommentLocation(path string, line int) string {
	return fmt.Sprintf("%s:%d", strings.TrimSpace(path), line)
}

// readCommentBody resolves a comment body from the given positional args, body flag,
// body-file flag, or stdin (checked in that order).
func readCommentBody(args []string, bodyFlag, bodyFileFlag string) (commentBodyInput, error) {
	positionalBody := ""
	if len(args) > 0 {
		positionalBody = strings.TrimSpace(args[0])
	}
	if positionalBody != "" && strings.TrimSpace(bodyFlag) != "" {
		return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: the positional body and --body are mutually exclusive\n  try: pass the comment once, either as the 2nd argument or via --body")
	}
	if positionalBody != "" && strings.TrimSpace(bodyFileFlag) != "" {
		return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: the positional body and --body-file are mutually exclusive\n  try: use either the 2nd argument or --body-file")
	}
	if strings.TrimSpace(bodyFlag) != "" && strings.TrimSpace(bodyFileFlag) != "" {
		return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: --body and --body-file are mutually exclusive\n  try: use only one of the 2nd argument, --body, --body-file, or stdin")
	}

	if positionalBody != "" {
		return commentBodyInput{Body: positionalBody, Source: "argument"}, nil
	}
	if strings.TrimSpace(bodyFlag) != "" {
		return commentBodyInput{Body: strings.TrimSpace(bodyFlag), Source: "--body"}, nil
	}
	if strings.TrimSpace(bodyFileFlag) != "" {
		var (
			data []byte
			err  error
		)
		if bodyFileFlag == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(bodyFileFlag)
		}
		if err != nil {
			return commentBodyInput{}, err
		}
		body := strings.TrimSpace(string(data))
		if body == "" {
			return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: %s did not contain any comment text\n  try: add text to %s or pass --body", bodyFileFlag, bodyFileFlag)
		}
		return commentBodyInput{Body: body, Source: "--body-file=" + bodyFileFlag}, nil
	}

	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return commentBodyInput{}, err
		}
		body := strings.TrimSpace(string(data))
		if body == "" {
			return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: stdin was provided but empty\n  try: pipe text into bv or pass a 2nd argument")
		}
		return commentBodyInput{Body: body, Source: "stdin"}, nil
	}

	return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: no comment body was provided\n  try: use the 2nd argument, --body, --body-file, or pipe stdin")
}

func filterThreadsByAuthor(threads []model.ReviewThread, author, excludeAuthor string) []model.ReviewThread {
	if author == "" && excludeAuthor == "" {
		return threads
	}
	filtered := make([]model.ReviewThread, 0, len(threads))
	for _, t := range threads {
		if author != "" && !threadHasAuthor(t, author) {
			continue
		}
		if excludeAuthor != "" && threadHasAuthor(t, excludeAuthor) {
			continue
		}
		filtered = append(filtered, t)
	}
	return filtered
}

func normalizeAuthor(author string) string {
	author = strings.TrimSpace(strings.TrimPrefix(author, "@"))
	author = strings.ToLower(author)
	return strings.TrimSuffix(author, "[bot]")
}

func threadHasAuthor(t model.ReviewThread, author string) bool {
	wanted := normalizeAuthor(author)
	for _, c := range t.Comments {
		if normalizeAuthor(c.Author) == wanted {
			return true
		}
	}
	return false
}
