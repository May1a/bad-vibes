package display

import (
	"testing"

	"github.com/may1a/bad-vibes/internal/model"
)

func TestBuildThreadSnippet_HighlightsTargetLine(t *testing.T) {
	thread := model.ReviewThread{
		Path:     "cmd/root.go",
		Line:     11,
		DiffSide: "RIGHT",
		Comments: []model.Comment{
			{
				DiffHunk: "@@ -10,3 +10,4 @@ func demo() {\n- oldLine()\n+ newLine()\n  shared()\n+ added()\n }",
			},
		},
	}

	lines, header, ok := buildThreadSnippet(thread, 1)
	if !ok {
		t.Fatal("expected snippet to be built")
	}
	if header == "" {
		t.Fatal("expected hunk header")
	}
	highlighted := 0
	for _, line := range lines {
		if line.Highlight {
			highlighted++
			if line.NewLine != 11 {
				t.Fatalf("expected highlighted new line 11, got %d", line.NewLine)
			}
		}
	}
	if highlighted != 1 {
		t.Fatalf("expected exactly one highlighted line, got %d", highlighted)
	}
}

func TestBuildThreadSnippet_DoesNotHighlightMissingTarget(t *testing.T) {
	thread := model.ReviewThread{
		Path:     "cmd/root.go",
		Line:     99,
		DiffSide: "RIGHT",
		Comments: []model.Comment{
			{
				DiffHunk: "@@ -10,3 +10,4 @@ func demo() {\n- oldLine()\n+ newLine()\n  shared()\n+ added()\n }",
			},
		},
	}

	lines, _, ok := buildThreadSnippet(thread, 1)
	if !ok {
		t.Fatal("expected snippet to be built")
	}
	for _, line := range lines {
		if line.Highlight {
			t.Fatal("did not expect any highlighted line when target is missing")
		}
	}
}
