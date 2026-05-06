package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bad-vibes/internal/git"
	"github.com/may1a/bad-vibes/internal/issues"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/spf13/cobra"
)

var issueCloseCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Close an issue",
	Long: `Close an open issue.

Examples:
  bv issue close 0001`,
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
		if issue.Status == model.IssueStatusClosed {
			return fmt.Errorf("issue %s is already closed", issue.ID)
		}

		issue.Status = model.IssueStatusClosed
		issue.UpdatedAt = time.Now()
		if err := issues.Save(repoRoot, issue); err != nil {
			return err
		}

		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		fmt.Printf("%s Issue %s closed.\n", green.Render("✓"), issue.ID)
		return nil
	},
}
