package display

import (
	"strings"
	"testing"
)

func TestSanitizeCodeRabbitBody_StripsHTMLComments(t *testing.T) {
	input := "Some text\n<!-- fingerprinting:abc123 -->\nMore text\n<!-- This is an auto-generated comment by CodeRabbit -->"
	got := SanitizeCodeRabbitBody(input)
	if strings.Contains(got, "<!--") {
		t.Fatalf("expected HTML comments to be stripped, got %q", got)
	}
	if !strings.Contains(got, "Some text") || !strings.Contains(got, "More text") {
		t.Fatalf("expected body text preserved, got %q", got)
	}
}

func TestSanitizeCodeRabbitBody_StripsDetailsBlocks(t *testing.T) {
	input := "_⚠️ Potential issue_\n\n**Fix the bug**\n\nThe code has a null pointer issue.\n\n<details><summary>🔧 Proposed fix</summary>\n\nfix()\n\n</details>\n\n<!-- suggestion_start -->\n<details><summary>📝 Committable suggestion</summary>\n\nfix()\n\n</details>\n<!-- suggestion_end -->\n\n<details><summary>🤖 Prompt for AI Agents</summary>Fix it</details>\n\n<!-- fingerprinting:abc -->"

	got := SanitizeCodeRabbitBody(input)
	if strings.Contains(got, "<details") {
		t.Fatalf("expected details blocks stripped, got %q", got)
	}
	if strings.Contains(got, "Proposed fix") || strings.Contains(got, "Committable") || strings.Contains(got, "AI Agents") {
		t.Fatalf("expected details content stripped, got %q", got)
	}
	if !strings.Contains(got, "Fix the bug") || !strings.Contains(got, "null pointer") {
		t.Fatalf("expected main body preserved, got %q", got)
	}
}

func TestSanitizeCodeRabbitBody_StripsPromoFooter(t *testing.T) {
	input := "Some review text.\n\n---\n\n<sub>CodeRabbit</sub> [Share on X](https://coderabbit.ai)"
	got := SanitizeCodeRabbitBody(input)
	if strings.Contains(got, "CodeRabbit") {
		t.Fatalf("expected promo footer stripped, got %q", got)
	}
}

func TestExtractCodeRabbitSummary_ParsesSeverityAndTitle(t *testing.T) {
	body := "_⚠️ Potential issue_\n\n**Handle nil pointer dereference**\n\nThe function foo may return nil when the input is empty.\n\n<details><summary>🔧 Proposed fix</summary>fix</details>"

	summary := ExtractCodeRabbitSummary(body)
	if summary.Severity != "⚠️ Potential issue" {
		t.Fatalf("expected severity %q, got %q", "⚠️ Potential issue", summary.Severity)
	}
	if summary.Title != "Handle nil pointer dereference" {
		t.Fatalf("expected title %q, got %q", "Handle nil pointer dereference", summary.Title)
	}
	if summary.Description == "" {
		t.Fatal("expected description to be non-empty")
	}
}

func TestExtractCodeRabbitSummary_MinorSeverity(t *testing.T) {
	body := "_🟡 Minor_\n\n**Use const instead of var**\n\nThis value never changes.\n"
	summary := ExtractCodeRabbitSummary(body)
	if summary.Severity != "🟡 Minor" {
		t.Fatalf("expected severity %q, got %q", "🟡 Minor", summary.Severity)
	}
	if summary.Title != "Use const instead of var" {
		t.Fatalf("expected title %q, got %q", "Use const instead of var", summary.Title)
	}
}

func TestCodeRabbitCompactBody(t *testing.T) {
	body := "_🟠 Major_\n\n**Fix the race condition**\n\nThe concurrent access is not guarded by a mutex.\n\n<details><summary>fix</summary>x</details>"
	got := codeRabbitCompactBody(body)
	if !strings.Contains(got, "[🟠 Major]") {
		t.Fatalf("expected severity badge, got %q", got)
	}
	if !strings.Contains(got, "Fix the race condition") {
		t.Fatalf("expected title, got %q", got)
	}
	if !strings.Contains(got, "concurrent access") {
		t.Fatalf("expected description, got %q", got)
	}
}

func TestCodeRabbitSeverityStyle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"⚠️ Potential issue", "[🟠 Major]"},
		{"🟠 Major", "[🟠 Major]"},
		{"🟡 Minor", "[🟡 Minor]"},
		{"✅ Nitpick", "[✅ Nitpick]"},
		{"Unknown", "[Unknown]"},
	}
	for _, tt := range tests {
		got := codeRabbitSeverityStyle(tt.input)
		if got != tt.expected {
			t.Errorf("codeRabbitSeverityStyle(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestIsCodeRabbit(t *testing.T) {
	if !isCodeRabbit("coderabbitai[bot]") {
		t.Error("expected coderabbitai[bot] to match")
	}
	if !isCodeRabbit("coderabbitai") {
		t.Error("expected coderabbitai to match")
	}
	if isCodeRabbit("human") {
		t.Error("expected human not to match")
	}
}

func TestExtractCodeRabbitSummary_TwoBadgePrefix(t *testing.T) {
	body := "_⚠️ Potential issue_ | _🟠 Major_\n\n**Fix the race condition**\n\nThe concurrent access is not guarded.\n\n<details><summary>fix</summary>x</details>"
	summary := ExtractCodeRabbitSummary(body)
	if summary.Severity != "⚠️ Potential issue | 🟠 Major" {
		t.Fatalf("expected severity %q, got %q", "⚠️ Potential issue | 🟠 Major", summary.Severity)
	}
	if summary.Title != "Fix the race condition" {
		t.Fatalf("expected title %q, got %q", "Fix the race condition", summary.Title)
	}
}

func TestExtractCodeRabbitSummary_TwoBadgePrefixNoTitle(t *testing.T) {
	body := "_⚠️ Potential issue_ | _🟠 Major_\n\n<details><summary>🔧 Proposed fix</summary>fix</details>"
	summary := ExtractCodeRabbitSummary(body)
	if summary.Severity != "⚠️ Potential issue | 🟠 Major" {
		t.Fatalf("expected severity %q, got %q", "⚠️ Potential issue | 🟠 Major", summary.Severity)
	}
	if summary.Title != "" {
		t.Fatalf("expected empty title, got %q", summary.Title)
	}
}

func TestCodeRabbitCompactBody_FallbackOnHTMLCommentPrefix(t *testing.T) {
	body := "<!-- fingerprinting:phantom:medusa:abc123 -->\nPlain text review comment without any markdown structure."
	got := codeRabbitCompactBody(body)
	if got == "" {
		t.Fatal("expected non-empty fallback when no structured summary found")
	}
	if strings.Contains(got, "<!--") {
		t.Fatalf("expected HTML comments stripped from fallback, got %q", got)
	}
}

func TestCodeRabbitCompactBody_FallbackOnPlainText(t *testing.T) {
	body := "This is a plain text bot reply with no severity or title markup."
	got := codeRabbitCompactBody(body)
	if got == "" {
		t.Fatal("expected non-empty fallback for plain text body")
	}
}
