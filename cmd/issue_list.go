package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bad-vibes/internal/git"
	"github.com/may1a/bad-vibes/internal/issues"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/spf13/cobra"
)

var issueListAll bool

var issueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	Long: `List open issues stored in .bv/issues/.

Use --all to include closed issues.

Examples:
  bv issue list
  bv issue list --all`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, err := git.RepoRoot()
		if err != nil {
			return fmt.Errorf("not in a git repository: %w", err)
		}

		all, err := issues.List(repoRoot, issueListAll)
		if err != nil {
			return err
		}

		dim := lipgloss.NewStyle().Faint(true)
		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
		bold := lipgloss.NewStyle().Bold(true)

		if len(all) == 0 {
			fmt.Println(dim.Render("No issues found."))
			return nil
		}

		fmt.Println()
		for _, issue := range all {
			var statusStyle lipgloss.Style
			if issue.Status == model.IssueStatusOpen {
				statusStyle = green
			} else {
				statusStyle = red
			}
			status := statusStyle.Render(fmt.Sprintf("%-6s", string(issue.Status)))
			id := bold.Render(issue.ID)
			title := issue.Title
			runes := []rune(title)
			if len(runes) > 60 {
				title = string(runes[:57]) + "..."
			}
			fmt.Printf("  %s  %s  %s\n", id, status, title)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	issueListCmd.Flags().BoolVar(&issueListAll, "all", false, "Include closed issues")
}
