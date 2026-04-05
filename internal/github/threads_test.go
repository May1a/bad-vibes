package github

import (
	"strings"
	"testing"

	"github.com/may1a/bv/internal/model"
)

func TestFindUnresolvedThreadIDPRLevel(t *testing.T) {
	id, ok, err := findUnresolvedThreadID([]model.ReviewThread{
		{ID: "resolved", Path: "", IsResolved: true},
		{ID: "pr", Path: ""},
		{ID: "line", Path: "cmd/root.go", Line: 10},
	}, "", 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected a PR-level thread match")
	}
	if id != "pr" {
		t.Fatalf("expected PR-level thread id %q, got %q", "pr", id)
	}
}

func TestFindUnresolvedThreadIDPRLevelMatchesBody(t *testing.T) {
	id, ok, err := findUnresolvedThreadID([]model.ReviewThread{
		{
			ID:   "first",
			Path: "",
			Comments: []model.Comment{
				{Body: "first body"},
			},
		},
		{
			ID:   "second",
			Path: "",
			Comments: []model.Comment{
				{Body: "second body"},
			},
		},
	}, "", 0, "second body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected a PR-level thread match")
	}
	if id != "second" {
		t.Fatalf("expected PR-level thread id %q, got %q", "second", id)
	}
}

func TestFindUnresolvedThreadIDAmbiguousWithoutBody(t *testing.T) {
	_, ok, err := findUnresolvedThreadID([]model.ReviewThread{
		{ID: "a", Path: "cmd/root.go", Line: 10},
		{ID: "b", Path: "cmd/root.go", Line: 10},
	}, "cmd/root.go", 10, "")
	if err == nil {
		t.Fatal("expected ambiguity error")
	}
	if ok {
		t.Fatal("expected ok to be false when ambiguous")
	}
	if !strings.Contains(err.Error(), "body required to disambiguate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindUnresolvedThreadIDPRLevelAmbiguousWithoutBody(t *testing.T) {
	_, ok, err := findUnresolvedThreadID([]model.ReviewThread{
		{ID: "a", Path: ""},
		{ID: "b", Path: ""},
	}, "", 0, "")
	if err == nil {
		t.Fatal("expected ambiguity error")
	}
	if ok {
		t.Fatal("expected ok to be false when ambiguous")
	}
	if !strings.Contains(err.Error(), "body required to disambiguate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindUnresolvedThreadIDMatchesBody(t *testing.T) {
	id, ok, err := findUnresolvedThreadID([]model.ReviewThread{
		{
			ID:   "a",
			Path: "cmd/root.go",
			Line: 10,
			Comments: []model.Comment{
				{Body: "first"},
			},
		},
		{
			ID:   "b",
			Path: "cmd/root.go",
			Line: 10,
			Comments: []model.Comment{
				{Body: "second"},
			},
		},
	}, "cmd/root.go", 10, "second")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected a unique body match")
	}
	if id != "b" {
		t.Fatalf("expected thread id %q, got %q", "b", id)
	}
}

func TestFindUnresolvedThreadIDBodyMiss(t *testing.T) {
	_, ok, err := findUnresolvedThreadID([]model.ReviewThread{
		{
			ID:   "a",
			Path: "cmd/root.go",
			Line: 10,
			Comments: []model.Comment{
				{Body: "first"},
			},
		},
	}, "cmd/root.go", 10, "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected no match")
	}
}
