package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/may/bad-vibes/internal/model"
)

func TestMergeAnchorsForDisplay_FallsBackToLocalAnchorsWithWarning(t *testing.T) {
	localAnchors := []model.Anchor{{
		Tag:  "perf",
		Path: "cmd/root.go",
		Line: 42,
		Body: "Needs work",
	}}

	got, warning, err := mergeAnchorsForDisplay(localAnchors, nil, errors.New("boom"))
	if err != nil {
		t.Fatalf("mergeAnchorsForDisplay() error = %v", err)
	}
	if len(got) != 1 || got[0].Tag != "perf" {
		t.Fatalf("expected local anchors to be preserved, got %+v", got)
	}
	if !strings.Contains(warning, "showing local anchors only") {
		t.Fatalf("expected fallback warning, got %q", warning)
	}
}

func TestMergeAnchorsForDisplay_ReturnsErrorWithoutLocalAnchors(t *testing.T) {
	_, _, err := mergeAnchorsForDisplay(nil, nil, errors.New("boom"))
	if err == nil {
		t.Fatal("expected fetch error")
	}
}
