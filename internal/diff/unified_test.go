package diff

import "testing"

const sampleUnifiedDiff = `diff --git a/cmd/root.go b/cmd/root.go
index 1111111..2222222 100644
--- a/cmd/root.go
+++ b/cmd/root.go
@@ -10,3 +10,4 @@ func demo() {
-	oldLine()
+	newLine()
 	shared()
+	added()
 }`

func TestParseUnified(t *testing.T) {
	patch, err := ParseUnified(sampleUnifiedDiff)
	if err != nil {
		t.Fatalf("ParseUnified() error = %v", err)
	}
	if len(patch.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(patch.Files))
	}

	file := patch.Files[0]
	if file.Path != "cmd/root.go" {
		t.Fatalf("expected cmd/root.go, got %q", file.Path)
	}
	if !file.HasCommentLine("LEFT", 10) {
		t.Fatal("expected deleted line 10 to be valid on LEFT")
	}
	if !file.HasCommentLine("RIGHT", 10) {
		t.Fatal("expected added line 10 to be valid on RIGHT")
	}
	if !file.HasCommentLine("RIGHT", 11) {
		t.Fatal("expected context line 11 to be valid on RIGHT")
	}
	if file.HasCommentLine("LEFT", 11) {
		t.Fatal("did not expect context line 11 to be valid on LEFT")
	}
}
