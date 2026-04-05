package cmd

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/may1a/bad-vibes/internal/github"
	"github.com/may1a/bad-vibes/internal/model"
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

func TestWaitForPostedThreadRetriesUntilExactMatch(t *testing.T) {
	prevLookup := findUnresolvedThreadByAt
	prevSleep := sleepForAnchorRetry
	t.Cleanup(func() {
		findUnresolvedThreadByAt = prevLookup
		sleepForAnchorRetry = prevSleep
	})

	calls := 0
	findUnresolvedThreadByAt = func(_ *github.Client, _ context.Context, _ model.PRRef, path string, line int, body string) (string, bool, error) {
		calls++
		if calls == 3 {
			return "thread-id", true, nil
		}
		return "", false, nil
	}
	sleepForAnchorRetry = func(_ time.Duration) {}

	id, ok, err := waitForPostedThread(context.Background(), model.PRRef{}, "cmd/root.go", 42, "body")
	if err != nil {
		t.Fatalf("waitForPostedThread() error = %v", err)
	}
	if !ok || id != "thread-id" {
		t.Fatalf("expected exact thread match, got ok=%v id=%q", ok, id)
	}
	if calls != 3 {
		t.Fatalf("expected 3 lookup attempts, got %d", calls)
	}
}

func TestStoreAnchorRequiresExactThreadMatch(t *testing.T) {
	prevLookup := findUnresolvedThreadByAt
	prevAddAnchor := addAnchorToCache
	prevSleep := sleepForAnchorRetry
	t.Cleanup(func() {
		findUnresolvedThreadByAt = prevLookup
		addAnchorToCache = prevAddAnchor
		sleepForAnchorRetry = prevSleep
	})

	added := false
	findUnresolvedThreadByAt = func(_ *github.Client, _ context.Context, _ model.PRRef, path string, line int, body string) (string, bool, error) {
		return "", false, nil
	}
	addAnchorToCache = func(model.PRRef, model.Anchor) error {
		added = true
		return nil
	}
	sleepForAnchorRetry = func(_ time.Duration) {}

	err := storeAnchor(context.Background(), model.PRRef{}, "perf", "cmd/root.go", 42, "body")
	if err == nil {
		t.Fatal("expected exact-match error")
	}
	if !strings.Contains(err.Error(), "exact thread") {
		t.Fatalf("expected exact-match guidance, got %v", err)
	}
	if added {
		t.Fatal("did not expect anchor to be cached without an exact thread match")
	}
}
