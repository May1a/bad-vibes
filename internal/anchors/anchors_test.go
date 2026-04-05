package anchors

import (
	"testing"
	"time"

	"github.com/may1a/bv/internal/model"
)

func TestMergeDiscoversTagsFromUnresolvedThreads(t *testing.T) {
	now := time.Now()
	anchors := Merge(nil, []model.ReviewThread{
		{
			ID:         "thread-1",
			Path:       "cmd/root.go",
			Line:       42,
			IsResolved: false,
			Comments: []model.Comment{
				{Body: "#cleanup\nNeeds cleanup", CreatedAt: now},
			},
		},
	})

	if len(anchors) != 1 {
		t.Fatalf("expected 1 anchor, got %d", len(anchors))
	}
	if anchors[0].Tag != "cleanup" || anchors[0].ThreadID != "thread-1" {
		t.Fatalf("unexpected anchor: %+v", anchors[0])
	}
}

func TestMergeKeepsDuplicateTagsAcrossThreads(t *testing.T) {
	anchors := Merge(nil, []model.ReviewThread{
		{
			ID:         "thread-1",
			Path:       "cmd/root.go",
			Line:       10,
			IsResolved: false,
			Comments:   []model.Comment{{Body: "#shared first"}},
		},
		{
			ID:         "thread-2",
			Path:       "cmd/comment.go",
			Line:       20,
			IsResolved: false,
			Comments:   []model.Comment{{Body: "#shared second"}},
		},
	})

	if len(anchors) != 2 {
		t.Fatalf("expected duplicate tags to be preserved for listing, got %+v", anchors)
	}
}

func TestResolvePrefersLocalAnchor(t *testing.T) {
	local := []model.Anchor{{Tag: "perf", ThreadID: "local-thread", Path: "a.go", Line: 1}}
	threads := []model.ReviewThread{
		{
			ID:         "thread-1",
			Path:       "cmd/root.go",
			Line:       42,
			IsResolved: false,
			Comments:   []model.Comment{{Body: "#perf discovered"}},
		},
	}

	anchor, err := Resolve(local, threads, "perf")
	if err != nil {
		t.Fatalf("expected anchor to resolve, got %v", err)
	}
	if anchor.ThreadID != "local-thread" {
		t.Fatalf("expected local anchor to win, got %+v", anchor)
	}
}

func TestMergeIgnoresInlineMentions(t *testing.T) {
	anchors := Merge(nil, []model.ReviewThread{
		{
			ID:         "thread-1",
			Path:       "cmd/root.go",
			Line:       42,
			IsResolved: false,
			Comments:   []model.Comment{{Body: "This references #cleanup inline."}},
		},
	})

	if len(anchors) != 0 {
		t.Fatalf("expected inline tag mentions to be ignored, got %+v", anchors)
	}
}

func TestResolveErrorsOnAmbiguousTag(t *testing.T) {
	_, err := Resolve(nil, []model.ReviewThread{
		{
			ID:         "thread-1",
			Path:       "cmd/root.go",
			Line:       10,
			IsResolved: false,
			Comments:   []model.Comment{{Body: "#shared first"}},
		},
		{
			ID:         "thread-2",
			Path:       "cmd/comment.go",
			Line:       20,
			IsResolved: false,
			Comments:   []model.Comment{{Body: "#shared second"}},
		},
	}, "shared")
	if err == nil {
		t.Fatal("expected ambiguity error")
	}
}
