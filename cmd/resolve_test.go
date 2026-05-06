package cmd

import (
	"testing"

	"github.com/may1a/bad-vibes/internal/model"
)

func TestThreadLabel(t *testing.T) {
	cases := []struct {
		thread model.ReviewThread
		want   string
	}{
		{model.ReviewThread{Path: "", Line: 0}, "PR-level comment"},
		{model.ReviewThread{Path: "cmd/root.go", Line: 42}, "cmd/root.go:42"},
		{model.ReviewThread{Path: "cmd/root.go", Line: 0}, "cmd/root.go"},
	}
	for _, c := range cases {
		got := threadLabel(c.thread)
		if got != c.want {
			t.Errorf("threadLabel(%+v) = %q, want %q", c.thread, got, c.want)
		}
	}
}

func TestIsNumericIndex(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"", false},
		{"1", true},
		{"42", true},
		{"0", true},
		{"abc", false},
		{"#perf", false},
		{"PRRT_abc", false},
	}
	for _, c := range cases {
		got := isNumericIndex(c.s)
		if got != c.want {
			t.Errorf("isNumericIndex(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}

func TestExtractFirstBoldTitle(t *testing.T) {
	cases := []struct {
		body string
		want string
	}{
		{"**Performance Issue**\nsome text", "Performance Issue"},
		{"no bold here", ""},
		{"**bold**", "bold"},
		{"  **spaced**  ", "spaced"},
	}
	for _, c := range cases {
		got := extractFirstBoldTitle(c.body)
		if got != c.want {
			t.Errorf("extractFirstBoldTitle(%q) = %q, want %q", c.body, got, c.want)
		}
	}
}
