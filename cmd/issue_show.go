package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bad-vibes/internal/git"
	"github.com/may1a/bad-vibes/internal/issues"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/spf13/cobra"
)

var issueShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show an issue",
	Long: `Show the details of an issue.

Examples:
  bv issue show 0001`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := git.RepoRoot()
		if err != nil {
			return fmt.Errorf("not in a git repository: %w", err)
		}

		issue, err := issues.Load(repoRoot, args[0])
		if err != nil {
			return err
		}

		bold := lipgloss.NewStyle().Bold(true)
		dim := lipgloss.NewStyle().Faint(true)
		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))

		var statusStyle lipgloss.Style
		if issue.Status == model.IssueStatusOpen {
			statusStyle = green
		} else {
			statusStyle = red
		}

		fmt.Printf("\n%s  %s  %s\n", bold.Render(issue.ID), issue.Title, statusStyle.Render(string(issue.Status)))
		fmt.Printf("  %s  %s\n", dim.Render("created:"), issue.CreatedAt.Format("2006-01-02 15:04"))
		if !issue.UpdatedAt.Equal(issue.CreatedAt) {
			fmt.Printf("  %s  %s\n", dim.Render("updated:"), issue.UpdatedAt.Format("2006-01-02 15:04"))
		}
		if issue.Body != "" {
			fmt.Println()
			for _, line := range strings.Split(issue.Body, "\n") {
				fmt.Println("  " + line)
			}
		}
		fmt.Println()
		return nil
	},
}
