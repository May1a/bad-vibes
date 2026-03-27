package display

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/may/bad-vibes/internal/model"
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
	Verbose  bool
	ShowDiff bool
}

func highlightAnchors(body string) string {
	return reAnchorTag.ReplaceAllStringFunc(body, func(m string) string {
		return styleAnchorTag.Render(m)
	})
}

// PrintThreads renders a slice of review threads to stdout.
// The caller is responsible for pre-filtering (e.g. only unresolved threads).
func PrintThreads(threads []model.ReviewThread, anchors []model.Anchor, opts ThreadRenderOptions) {
	anchorByThread := map[string]string{}
	for _, a := range anchors {
		anchorByThread[a.ThreadID] = "#" + a.Tag
	}

	for i, t := range threads {
		if i > 0 {
			fmt.Println()
		}
		printThread(t, anchorByThread, opts)
	}
}

func printThread(t model.ReviewThread, anchorByThread map[string]string, opts ThreadRenderOptions) {
	header := styleThreadFile.Render(threadLocation(t)) + "  " + styleThreadID.Render("["+t.ID+"]")
	if t.IsOutdated {
		header += "  " + styleOutdated.Render("[OUTDATED]")
	}
	if tag, ok := anchorByThread[t.ID]; ok {
		header += "  " + styleAnchorTag.Render(tag)
	}
	fmt.Println(header)

	if len(t.Comments) == 0 {
		fmt.Println("  " + styleDate.Render("(no comments)"))
		return
	}

	if opts.ShowDiff && t.Comments[0].DiffHunk != "" {
		for _, hunkLine := range strings.Split(t.Comments[0].DiffHunk, "\n") {
			fmt.Println("  " + styleThreadHunk.Render(hunkLine))
		}
		fmt.Println()
	}

	if !opts.Verbose {
		c := t.Comments[len(t.Comments)-1]
		fmt.Println("  " + styleAuthor.Render("@"+c.Author) + "  " + styleDate.Render(formatDate(c.CreatedAt)))
		for _, bodyLine := range strings.Split(previewBody(c.Body), "\n") {
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
		for _, bodyLine := range strings.Split(highlightAnchors(strings.TrimSpace(c.Body)), "\n") {
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
