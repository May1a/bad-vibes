package display

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bv/internal/diff"
)

var (
	styleFileHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#334155")).Padding(0, 1)
	styleHunk       = lipgloss.NewStyle().Foreground(lipgloss.Color("#67e8f9")).Faint(true)
	styleAdd        = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	styleDel        = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
	styleContext    = lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8"))
	styleLineNum    = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748b")).Faint(true)
)

// PrintDiff writes a coloured unified diff to stdout.
func PrintDiff(raw string) {
	patch, err := diff.ParseUnified(raw)
	if err != nil {
		fmt.Print(raw)
		return
	}

	for _, file := range patch.Files {
		if file.Path == "" {
			continue
		}
		fmt.Println(styleFileHeader.Render("  " + file.Path + "  "))
		for _, hunk := range file.Hunks {
			fmt.Println(styleHunk.Render(hunk.Header))
			for _, line := range hunk.Lines {
				switch line.Kind {
				case diff.LineAdd:
					num := styleLineNum.Render(fmt.Sprintf("    %4s  %4d  ", "·", line.NewLine))
					fmt.Println(num + styleAdd.Render(line.Content))
				case diff.LineDelete:
					num := styleLineNum.Render(fmt.Sprintf("    %4d  %4s  ", line.OldLine, "·"))
					fmt.Println(num + styleDel.Render(line.Content))
				case diff.LineMeta:
					fmt.Println(styleContext.Faint(true).Render(line.Content))
				default:
					num := styleLineNum.Render(fmt.Sprintf("    %4d  %4d  ", line.OldLine, line.NewLine))
					fmt.Println(num + styleContext.Render(line.Content))
				}
			}
		}
	}
}
