package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/may1a/bad-vibes/internal/model"
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
}

func TestCache_SaveAndLoad(t *testing.T) {
	ref := model.PRRef{Owner: "test", Repo: "test", Number: 1}

	original := model.PRCache{
		Owner:   "test",
		Repo:    "test",
		Number:  1,
		PRID:    "PR_kwDOABC123",
		HeadSHA: "abc123def456",
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
