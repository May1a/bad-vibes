package display

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/may/bad-vibes/internal/model"
)

var (
	styleThreadBorder  = lipgloss.NewStyle().Foreground(lipgloss.Color("#475569"))
	styleThreadFile    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#facc15"))
	styleThreadID      = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748b")).Faint(true)
	styleOutdated      = lipgloss.NewStyle().Foreground(lipgloss.Color("#f97316")).Bold(true)
	styleAuthor        = lipgloss.NewStyle().Foreground(lipgloss.Color("#38bdf8"))
	styleDate          = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748b")).Faint(true)
	styleAnchorTag     = lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")).Bold(true)

	reAnchorTag = regexp.MustCompile(`#[\w-]+`)
)

func highlightAnchors(body string) string {
	return reAnchorTag.ReplaceAllStringFunc(body, func(m string) string {
		return styleAnchorTag.Render(m)
	})
}

// PrintThreads renders a slice of review threads to stdout.
// The caller is responsible for pre-filtering (e.g. only unresolved threads).
func PrintThreads(threads []model.ReviewThread, anchors []model.Anchor) {
	// Build a reverse map: threadID → anchor tag
	anchorByThread := map[string]string{}
	for _, a := range anchors {
		anchorByThread[a.ThreadID] = "#" + a.Tag
	}

	border := styleThreadBorder.Render

	for _, t := range threads {
		// Header line
		location := "PR-level comment"
		if t.Path != "" {
			if t.Line > 0 {
				location = fmt.Sprintf("%s:%d", t.Path, t.Line)
			} else {
				location = t.Path
			}
		}

		header := styleThreadFile.Render(location) + "  " + styleThreadID.Render("["+t.ID+"]")
		if t.IsOutdated {
			header += "  " + styleOutdated.Render("[OUTDATED]")
		}
		if tag, ok := anchorByThread[t.ID]; ok {
			header += "  " + styleAnchorTag.Render(tag)
		}

		fmt.Println(border("┌─") + " " + header)

		styleHunk := lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#475569"))
		for i, c := range t.Comments {
			dateFmt := c.CreatedAt.Format("2006-01-02")
			meta := styleAuthor.Render("@"+c.Author) + "  " + styleDate.Render(dateFmt)
			fmt.Println(border("│") + "  " + meta)

			if i == 0 && c.DiffHunk != "" {
				for _, hunkLine := range strings.Split(c.DiffHunk, "\n") {
					fmt.Println(border("│") + "  " + styleHunk.Render(hunkLine))
				}
				fmt.Println(border("│"))
			}

			for _, bodyLine := range strings.Split(highlightAnchors(c.Body), "\n") {
				fmt.Println(border("│") + "  " + bodyLine)
			}

			if i < len(t.Comments)-1 {
				fmt.Println(border("│"))
			}
		}

		fmt.Println(border("└─"))
		fmt.Println()
	}
}
