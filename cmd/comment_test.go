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

	_, err := readCommentBody()
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

	_, err = readCommentBody()
	if err == nil {
		t.Fatal("expected empty file error")
	}
	if !strings.Contains(err.Error(), "did not contain any comment text") {
		t.Fatalf("expected empty file guidance, got %v", err)
	}
}
