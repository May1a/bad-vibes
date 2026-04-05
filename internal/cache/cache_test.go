package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/may1a/bv/internal/model"
)

func TestCache_LoadNonExistent(t *testing.T) {
	ref := model.PRRef{Owner: "test", Repo: "test", Number: 99999}
	c, err := Load(ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Number != 99999 {
		t.Fatalf("expected number 99999, got %d", c.Number)
	}
	if len(c.Anchors) != 0 {
		t.Fatalf("expected no anchors, got %d", len(c.Anchors))
	}
}

func TestCache_SaveAndLoad(t *testing.T) {
	ref := model.PRRef{Owner: "test", Repo: "test", Number: 1}

	original := model.PRCache{
		Owner:   "test",
		Repo:    "test",
		Number:  1,
		PRID:    "PR_kwDOABC123",
		HeadSHA: "abc123def456",
		Anchors: []model.Anchor{
			{
				Tag:      "perf",
				ThreadID: "PRRT_abc123",
				Path:     "cmd/root.go",
				Line:     42,
				Body:     "Performance issue here",
				Created:  time.Now(),
			},
		},
	}

	err := Save(ref, original)
	if err != nil {
		t.Fatalf("unexpected error saving: %v", err)
	}

	loaded, err := Load(ref)
	if err != nil {
		t.Fatalf("unexpected error loading: %v", err)
	}

	if loaded.Owner != original.Owner {
		t.Errorf("owner mismatch: %s != %s", loaded.Owner, original.Owner)
	}
	if loaded.Repo != original.Repo {
		t.Errorf("repo mismatch: %s != %s", loaded.Repo, original.Repo)
	}
	if loaded.Number != original.Number {
		t.Errorf("number mismatch: %d != %d", loaded.Number, original.Number)
	}
	if loaded.PRID != original.PRID {
		t.Errorf("PRID mismatch: %s != %s", loaded.PRID, original.PRID)
	}
	if loaded.HeadSHA != original.HeadSHA {
		t.Errorf("HeadSHA mismatch: %s != %s", loaded.HeadSHA, original.HeadSHA)
	}
	if len(loaded.Anchors) != len(original.Anchors) {
		t.Fatalf("anchors length mismatch: %d != %d", len(loaded.Anchors), len(original.Anchors))
	}

	a := loaded.Anchors[0]
	o := original.Anchors[0]
	if a.Tag != o.Tag {
		t.Errorf("anchor tag mismatch: %s != %s", a.Tag, o.Tag)
	}
	if a.Path != o.Path {
		t.Errorf("anchor path mismatch: %s != %s", a.Path, o.Path)
	}
	if a.Line != o.Line {
		t.Errorf("anchor line mismatch: %d != %d", a.Line, o.Line)
	}
}

func TestCache_AddAnchor(t *testing.T) {
	ref := model.PRRef{Owner: "test", Repo: "test", Number: 2}

	anchor1 := model.Anchor{
		Tag:      "perf",
		ThreadID: "PRRT_abc123",
		Path:     "cmd/root.go",
		Line:     42,
		Body:     "Performance issue",
		Created:  time.Now(),
	}

	anchor2 := model.Anchor{
		Tag:      "security",
		ThreadID: "PRRT_def456",
		Path:     "internal/auth/auth.go",
		Line:     100,
		Body:     "Security concern",
		Created:  time.Now(),
	}

	err := AddAnchor(ref, anchor1)
	if err != nil {
		t.Fatalf("unexpected error adding anchor1: %v", err)
	}

	err = AddAnchor(ref, anchor2)
	if err != nil {
		t.Fatalf("unexpected error adding anchor2: %v", err)
	}

	anchors, err := ListAnchors(ref)
	if err != nil {
		t.Fatalf("unexpected error listing anchors: %v", err)
	}

	if len(anchors) != 2 {
		t.Fatalf("expected 2 anchors, got %d", len(anchors))
	}
}

func TestCache_AddAnchorReplace(t *testing.T) {
	ref := model.PRRef{Owner: "test", Repo: "test", Number: 3}

	anchor1 := model.Anchor{
		Tag:      "perf",
		ThreadID: "PRRT_old",
		Path:     "old.go",
		Line:     10,
		Body:     "Old body",
		Created:  time.Now(),
	}

	anchor2 := model.Anchor{
		Tag:      "perf",
		ThreadID: "PRRT_new",
		Path:     "new.go",
		Line:     20,
		Body:     "New body",
		Created:  time.Now(),
	}

	err := AddAnchor(ref, anchor1)
	if err != nil {
		t.Fatalf("unexpected error adding anchor1: %v", err)
	}

	err = AddAnchor(ref, anchor2)
	if err != nil {
		t.Fatalf("unexpected error adding anchor2: %v", err)
	}

	anchors, err := ListAnchors(ref)
	if err != nil {
		t.Fatalf("unexpected error listing anchors: %v", err)
	}

	if len(anchors) != 1 {
		t.Fatalf("expected 1 anchor (replaced), got %d", len(anchors))
	}

	if anchors[0].ThreadID != "PRRT_new" {
		t.Errorf("expected replaced anchor, got %s", anchors[0].ThreadID)
	}
}

func TestCache_ListAnchorsNonExistent(t *testing.T) {
	ref := model.PRRef{Owner: "test", Repo: "test", Number: 99999}
	anchors, err := ListAnchors(ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(anchors) != 0 {
		t.Fatalf("expected 0 anchors, got %d", len(anchors))
	}
}

func TestCachePath_CreatesDirectory(t *testing.T) {
	ref := model.PRRef{Owner: "testowner", Repo: "testrepo", Number: 42}

	path, err := cachePath(ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("cache directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("cache path is not a directory")
	}
}
