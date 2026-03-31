package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestReadCommentBody_BodyAndBodyFileConflict(t *testing.T) {
	prevBody, prevFile := commentBody, commentBodyFile
	t.Cleanup(func() {
		commentBody = prevBody
		commentBodyFile = prevFile
	})

	commentBody = "inline"
	commentBodyFile = "comment.md"

	_, err := readCommentBody(nil)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutual exclusivity message, got %v", err)
	}
}

func TestReadCommentBody_EmptyBodyFile(t *testing.T) {
	prevBody, prevFile := commentBody, commentBodyFile
	t.Cleanup(func() {
		commentBody = prevBody
		commentBodyFile = prevFile
	})

	file, err := os.CreateTemp(t.TempDir(), "comment-*.md")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	commentBody = ""
	commentBodyFile = file.Name()

	_, err = readCommentBody(nil)
	if err == nil {
		t.Fatal("expected empty file error")
	}
	if !strings.Contains(err.Error(), "did not contain any comment text") {
		t.Fatalf("expected empty file guidance, got %v", err)
	}
}

func TestReadCommentBody_PositionalConflict(t *testing.T) {
	prevBody, prevFile := commentBody, commentBodyFile
	t.Cleanup(func() {
		commentBody = prevBody
		commentBodyFile = prevFile
	})

	commentBody = "inline"
	commentBodyFile = ""

	_, err := readCommentBody([]string{"argument"})
	if err == nil {
		t.Fatal("expected positional body conflict")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutual exclusivity message, got %v", err)
	}
}

func TestParseCommentLocation(t *testing.T) {
	got, err := parseCommentLocation("bv comment", "cmd/root.go:42")
	if err != nil {
		t.Fatalf("parseCommentLocation() error = %v", err)
	}
	if got.Path != "cmd/root.go" || got.Line != 42 {
		t.Fatalf("unexpected location: %+v", got)
	}
}
