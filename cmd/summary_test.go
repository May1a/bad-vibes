package cmd

import (
	"strings"
	"testing"

	"github.com/may1a/bad-vibes/internal/model"
)

func TestFormatSummaryFileStatus(t *testing.T) {
	got := formatSummaryFileStatus(model.PRFile{
		Path:   "cmd/root.go",
		Status: "added",
	})
	if !strings.Contains(got, "[new]") || !strings.Contains(got, "cmd/root.go") {
		t.Fatalf("unexpected formatted status: %q", got)
	}
}

func TestFormatSummaryFileStatus_Rename(t *testing.T) {
	got := formatSummaryFileStatus(model.PRFile{
		Path:         "cmd/new.go",
		PreviousPath: "cmd/old.go",
		Status:       "renamed",
		Additions:    3,
		Deletions:    1,
	})
	if !strings.Contains(got, "cmd/old.go -> cmd/new.go") {
		t.Fatalf("expected rename detail, got %q", got)
	}
}
