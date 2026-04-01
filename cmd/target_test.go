package cmd

import (
	"strings"
	"testing"
)

func TestResolveTargetInput_ExplicitPRFlag(t *testing.T) {
	got, err := resolveTargetInput(targetResolutionInput{
		CommandPath: "bv summary",
		RepoFlag:    "owner/repo",
		PRFlag:      "99",
	})
	if err != nil {
		t.Fatalf("resolveTargetInput() error = %v", err)
	}
	if got.Ref.Number != 99 || got.Ref.Owner != "owner" || got.Ref.Repo != "repo" {
		t.Fatalf("unexpected resolution: %+v", got.Ref)
	}
}

func TestResolveTargetInput_ExplicitRepoWithoutGitContext(t *testing.T) {
	got, err := resolveTargetInput(targetResolutionInput{
		CommandPath: "bv comments",
		RepoFlag:    "owner/repo",
		PRFlag:      "42",
	})
	if err != nil {
		t.Fatalf("resolveTargetInput() error = %v", err)
	}
	if got.Ref.Number != 42 || got.Ref.Owner != "owner" || got.Ref.Repo != "repo" {
		t.Fatalf("unexpected resolution: %+v", got.Ref)
	}
}

func TestResolveTargetInput_AutoDetect(t *testing.T) {
	got, err := resolveTargetInput(targetResolutionInput{
		CommandPath:    "bv diff",
		DetectedRepo:   "owner/repo",
		DetectedBranch: "feature/test",
	})
	if err != nil {
		t.Fatalf("resolveTargetInput() error = %v", err)
	}
	if !got.NeedsPRAutoPick {
		t.Fatal("expected auto-pick to be required")
	}
	if got.Branch != "feature/test" || got.Ref.Owner != "owner" || got.Ref.Repo != "repo" {
		t.Fatalf("unexpected resolution: %+v", got)
	}
}

func TestResolveTargetInput_RepoConflict(t *testing.T) {
	_, err := resolveTargetInput(targetResolutionInput{
		CommandPath: "bv comment",
		RepoFlag:    "owner/repo",
		PRFlag:      "other/repo#42",
	})
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected conflict message, got %v", err)
	}
}

func TestResolveTargetInput_MissingRepoContext(t *testing.T) {
	_, err := resolveTargetInput(targetResolutionInput{
		CommandPath: "bv summary",
		PRFlag:      "42",
	})
	if err == nil {
		t.Fatal("expected missing repo error")
	}
	if !strings.Contains(err.Error(), "--repo owner/repo --pr 42") {
		t.Fatalf("expected actionable hint, got %v", err)
	}
}
