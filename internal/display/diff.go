package display

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleFileHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#334155")).Padding(0, 1)
	styleHunk       = lipgloss.NewStyle().Foreground(lipgloss.Color("#67e8f9")).Faint(true)
	styleAdd        = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	styleDel        = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
	styleContext    = lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8"))
	styleLineNum    = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748b")).Faint(true)

	reHunk = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)
)

// PrintDiff writes a coloured unified diff to stdout.
func PrintDiff(raw string) {
	var oldLine, newLine int

	for _, line := range strings.Split(raw, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "new file") || strings.HasPrefix(line, "deleted file") ||
			strings.HasPrefix(line, "similarity") || strings.HasPrefix(line, "rename"):
			// skip git metadata lines

		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ "):
			// file header pair — print the +++ line as the file header
			if strings.HasPrefix(line, "+++ ") {
				file := strings.TrimPrefix(line, "+++ b/")
				file = strings.TrimPrefix(file, "+++ ")
				fmt.Println(styleFileHeader.Render("  " + file + "  "))
			}

		case strings.HasPrefix(line, "@@"):
			m := reHunk.FindStringSubmatch(line)
			if m != nil {
				oldLine, _ = strconv.Atoi(m[1])
				newLine, _ = strconv.Atoi(m[2])
			}
			fmt.Println(styleHunk.Render(line))

		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			num := styleLineNum.Render(fmt.Sprintf("    %4s  %4d  ", "·", newLine))
			fmt.Println(num + styleAdd.Render(line))
			newLine++

		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			num := styleLineNum.Render(fmt.Sprintf("    %4d  %4s  ", oldLine, "·"))
			fmt.Println(num + styleDel.Render(line))
			oldLine++

		case line == "\\ No newline at end of file":
			fmt.Println(styleContext.Faint(true).Render(line))

		default:
			num := styleLineNum.Render(fmt.Sprintf("    %4d  %4d  ", oldLine, newLine))
			fmt.Println(num + styleContext.Render(line))
			if oldLine > 0 {
				oldLine++
				newLine++
			}
		}
	}
}
