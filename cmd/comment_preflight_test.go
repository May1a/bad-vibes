package cmd

import (
	"strings"
	"testing"

	"github.com/may/bad-vibes/internal/diff"
)

const sampleCommentDiff = `diff --git a/cmd/root.go b/cmd/root.go
index 1111111..2222222 100644
--- a/cmd/root.go
+++ b/cmd/root.go
@@ -10,3 +10,4 @@ func demo() {
-	oldLine()
+	newLine()
 	shared()
+	added()
 }`

func mustPatch(t *testing.T) diff.Patch {
	t.Helper()
	patch, err := diff.ParseUnified(sampleCommentDiff)
	if err != nil {
		t.Fatalf("ParseUnified() error = %v", err)
	}
	return patch
}

func TestPreflightCommentTarget_NormalizesAbsolutePath(t *testing.T) {
	patch := mustPatch(t)
	got, err := preflightCommentTarget(commentPreflightInput{
		CommandPath: "bv comment",
		RawPath:     "/repo/cmd/root.go",
		Line:        10,
		Side:        "RIGHT",
		Patch:       patch,
		WorkingDir:  "/repo",
		RepoRoot:    "/repo",
	})
	if err != nil {
		t.Fatalf("preflightCommentTarget() error = %v", err)
	}
	if got.Path != "cmd/root.go" {
		t.Fatalf("expected normalized path cmd/root.go, got %q", got.Path)
	}
}

func TestPreflightCommentTarget_SuggestsCloseMatch(t *testing.T) {
	patch := mustPatch(t)
	_, err := preflightCommentTarget(commentPreflightInput{
		CommandPath: "bv comment",
		RawPath:     "root.go",
		Line:        10,
		Side:        "RIGHT",
		Patch:       patch,
	})
	if err == nil {
		t.Fatal("expected suggestion error")
	}
	if !strings.Contains(err.Error(), "cmd/root.go") {
		t.Fatalf("expected close-match suggestion, got %v", err)
	}
}

func TestPreflightCommentTarget_SuggestsOtherSide(t *testing.T) {
	patch := mustPatch(t)
	_, err := preflightCommentTarget(commentPreflightInput{
		CommandPath: "bv comment",
		RawPath:     "cmd/root.go",
		Line:        11,
		Side:        "LEFT",
		Patch:       patch,
	})
	if err == nil {
		t.Fatal("expected side guidance error")
	}
	if !strings.Contains(err.Error(), "--side RIGHT") {
		t.Fatalf("expected alternate-side hint, got %v", err)
	}
}
