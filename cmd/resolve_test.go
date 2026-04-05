package cmd

import (
	"testing"

	"github.com/may1a/bv/internal/model"
)

func TestResolveSelectionWithoutIDUsesFirstUnresolvedThread(t *testing.T) {
	selection, err := resolveSelection(model.PRRef{Number: 5}, "", nil, []model.ReviewThread{
		{ID: "resolved", IsResolved: true, Path: "ignored.go", Line: 1},
		{ID: "first", Path: "cmd/root.go", Line: 10},
		{ID: "second", Path: "cmd/comment.go", Line: 20},
	})
	if err != nil {
		t.Fatalf("resolveSelection() error = %v", err)
	}
	if selection.ThreadID != "first" {
		t.Fatalf("expected first unresolved thread, got %+v", selection)
	}
	if selection.Description != "cmd/root.go:10" {
		t.Fatalf("unexpected selection description: %+v", selection)
	}
}

func TestResolveSelectionAnchorUsesProvidedThreads(t *testing.T) {
	selection, err := resolveSelection(model.PRRef{Number: 5}, "#perf", nil, []model.ReviewThread{
		{
			ID:   "thread-1",
			Path: "cmd/root.go",
			Line: 42,
			Comments: []model.Comment{
				{Body: "#perf tighten this up"},
			},
		},
	})
	if err != nil {
		t.Fatalf("resolveSelection() error = %v", err)
	}
	if selection.ThreadID != "thread-1" {
		t.Fatalf("expected provided thread snapshot to resolve anchor, got %+v", selection)
	}
}

func TestResolveSelectionPRLevelUsesFirstPRLevelThread(t *testing.T) {
	selection, err := resolveSelection(model.PRRef{Number: 5}, "#PR", nil, []model.ReviewThread{
		{ID: "line", Path: "cmd/root.go", Line: 10},
		{ID: "pr", Path: ""},
	})
	if err != nil {
		t.Fatalf("resolveSelection() error = %v", err)
	}
	if selection.ThreadID != "pr" || selection.Description != "PR-level comment" {
		t.Fatalf("unexpected PR-level selection: %+v", selection)
	}
}
