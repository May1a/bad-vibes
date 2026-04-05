package display

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bv/internal/diff"
	"github.com/may1a/bv/internal/model"
)

var (
	styleThreadFile = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#facc15"))
	styleThreadID   = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748b")).Faint(true)
	styleOutdated   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f97316")).Bold(true)
	styleAuthor     = lipgloss.NewStyle().Foreground(lipgloss.Color("#38bdf8"))
	styleDate       = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748b")).Faint(true)
	styleAnchorTag  = lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")).Bold(true)
	styleThreadHunk = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#475569"))

	reAnchorTag = regexp.MustCompile(`#[\w-]+`)
)

type ThreadRenderOptions struct {
	Verbose     bool
	ShowDiff    bool
	ShowSnippet bool
}

func highlightAnchors(body string) string {
	return reAnchorTag.ReplaceAllStringFunc(body, func(m string) string {
		return styleAnchorTag.Render(m)
	})
}

// PrintThreads renders a slice of review threads to stdout.
// The caller is responsible for pre-filtering (e.g. only unresolved threads).
func PrintThreads(threads []model.ReviewThread, anchors []model.Anchor, opts ThreadRenderOptions) {
	anchorByThread := map[string][]string{}
	for _, a := range anchors {
		anchorByThread[a.ThreadID] = append(anchorByThread[a.ThreadID], "#"+a.Tag)
	}

	for i, t := range threads {
		if i > 0 {
			fmt.Println()
		}
		printThread(t, anchorByThread, opts)
	}
}

func printThread(t model.ReviewThread, anchorByThread map[string][]string, opts ThreadRenderOptions) {
	header := styleThreadFile.Render(threadLocation(t)) + "  " + styleThreadID.Render("["+t.ID+"]")
	if t.IsOutdated {
		header += "  " + styleOutdated.Render("[OUTDATED]")
	}
	if tags, ok := anchorByThread[t.ID]; ok {
		header += "  " + styleAnchorTag.Render(strings.Join(tags, " "))
	}
	fmt.Println(header)

	if len(t.Comments) == 0 {
		fmt.Println("  " + styleDate.Render("(no comments)"))
		return
	}

	if opts.ShowDiff && t.Comments[0].DiffHunk != "" {
		for hunkLine := range strings.SplitSeq(t.Comments[0].DiffHunk, "\n") {
			fmt.Println("  " + styleThreadHunk.Render(hunkLine))
		}
		fmt.Println()
	} else if opts.ShowSnippet {
		if snippet := renderThreadSnippet(t); len(snippet) > 0 {
			for _, line := range snippet {
				fmt.Println("  " + line)
			}
			fmt.Println()
		}
	}

	if !opts.Verbose {
		c := t.Comments[len(t.Comments)-1]
		fmt.Println("  " + styleAuthor.Render("@"+c.Author) + "  " + styleDate.Render(formatDate(c.CreatedAt)))
		for bodyLine := range strings.SplitSeq(previewBody(c.Body), "\n") {
			fmt.Println("  " + bodyLine)
		}
		if len(t.Comments) > 1 {
			fmt.Println("  " + styleDate.Render(fmt.Sprintf("(%d comments in thread; use --verbose to show all)", len(t.Comments))))
		}
		return
	}

	for i, c := range t.Comments {
		if i > 0 {
			fmt.Println()
		}
		fmt.Println("  " + styleAuthor.Render("@"+c.Author) + "  " + styleDate.Render(formatDate(c.CreatedAt)))
		for bodyLine := range strings.SplitSeq(highlightAnchors(strings.TrimSpace(c.Body)), "\n") {
			fmt.Println("  " + bodyLine)
		}
	}
}

func threadLocation(t model.ReviewThread) string {
	if t.Path == "" {
		return "PR-level comment"
	}
	if t.Line > 0 {
		return fmt.Sprintf("%s:%d", t.Path, t.Line)
	}
	return t.Path
}

func formatDate(ts time.Time) string {
	if ts.IsZero() {
		return "unknown date"
	}
	return ts.Format("2006-01-02")
}

func previewBody(body string) string {
	body = strings.Join(strings.Fields(strings.TrimSpace(body)), " ")
	body = highlightAnchors(body)
	runes := []rune(body)
	if len(runes) > 180 {
		return string(runes[:177]) + "..."
	}
	return body
}

type snippetLine struct {
	Kind      diff.LineKind
	Content   string
	OldLine   int
	NewLine   int
	Highlight bool
}

func renderThreadSnippet(t model.ReviewThread) []string {
	lines, header, ok := buildThreadSnippet(t, 2)
	if !ok {
		return nil
	}

	rendered := []string{styleThreadHunk.Render(header)}
	for _, line := range lines {
		marker := " "
		if line.Highlight {
			marker = styleThreadFile.Render(">")
		}
		num := styleLineNum.Render(fmt.Sprintf("%4s %4s", formatSnippetLineNumber(line.OldLine), formatSnippetLineNumber(line.NewLine)))

		var content string
		switch line.Kind {
		case diff.LineAdd:
			content = styleAdd.Render(line.Content)
		case diff.LineDelete:
			content = styleDel.Render(line.Content)
		default:
			content = styleContext.Render(line.Content)
		}
		rendered = append(rendered, fmt.Sprintf("%s %s %s", marker, num, content))
	}
	return rendered
}

func buildThreadSnippet(t model.ReviewThread, contextLines int) ([]snippetLine, string, bool) {
	rawHunk := firstDiffHunk(t)
	if rawHunk == "" || t.Path == "" {
		return nil, "", false
	}

	fakePatch := fmt.Sprintf("diff --git a/%[1]s b/%[1]s\n--- a/%[1]s\n+++ b/%[1]s\n%s", t.Path, rawHunk)
	patch, err := diff.ParseUnified(fakePatch)
	if err != nil || len(patch.Files) == 0 || len(patch.Files[0].Hunks) == 0 {
		return nil, "", false
	}

	hunk := patch.Files[0].Hunks[0]
	highlightIndex := findThreadSnippetTarget(t, hunk.Lines)
	centerIndex := max(highlightIndex, 0)

	start := max(centerIndex-contextLines, 0)
	end := min(centerIndex+contextLines+1, len(hunk.Lines))

	lines := make([]snippetLine, 0, end-start)
	for i := start; i < end; i++ {
		line := hunk.Lines[i]
		lines = append(lines, snippetLine{
			Kind:      line.Kind,
			Content:   line.Content,
			OldLine:   line.OldLine,
			NewLine:   line.NewLine,
			Highlight: highlightIndex >= 0 && i == highlightIndex,
		})
	}
	return lines, hunk.Header, true
}

func firstDiffHunk(t model.ReviewThread) string {
	for _, comment := range t.Comments {
		if strings.TrimSpace(comment.DiffHunk) != "" {
			return comment.DiffHunk
		}
	}
	return ""
}

func findThreadSnippetTarget(t model.ReviewThread, lines []diff.Line) int {
	targetLine := t.Line
	if targetLine == 0 {
		targetLine = t.StartLine
	}
	for i, line := range lines {
		switch strings.ToUpper(strings.TrimSpace(t.DiffSide)) {
		case "LEFT":
			if line.OldLine == targetLine && line.Kind != diff.LineAdd {
				return i
			}
		default:
			if line.NewLine == targetLine && line.Kind != diff.LineDelete {
				return i
			}
		}
	}
	return -1
}

func formatSnippetLineNumber(n int) string {
	if n == 0 {
		return "·"
	}
	return fmt.Sprintf("%d", n)
}
