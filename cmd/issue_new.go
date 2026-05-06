package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bad-vibes/internal/git"
	"github.com/may1a/bad-vibes/internal/issues"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/spf13/cobra"
)

var issueNewBody string

var issueNewCmd = &cobra.Command{
	Use:   "new <title>",
	Short: "Create a new issue",
	Long: `Create a new issue in .bv/issues/.

Examples:
  bv issue new "Cache invalidation is too aggressive"
  bv issue new "Auth tokens are not rotated" --body "Found in internal/auth/auth.go"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.TrimSpace(args[0])
		if title == "" {
			return fmt.Errorf("title cannot be empty")
		}

		repoRoot, err := git.RepoRoot()
		if err != nil {
			return fmt.Errorf("not in a git repository: %w", err)
		}

		id, err := issues.NextID(repoRoot)
		if err != nil {
			return err
		}

		now := time.Now()
		issue := model.Issue{
			ID:        id,
			Title:     title,
			Body:      strings.TrimSpace(issueNewBody),
			Status:    model.IssueStatusOpen,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := issues.Save(repoRoot, issue); err != nil {
			return err
		}

		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		dim := lipgloss.NewStyle().Faint(true)
		fmt.Printf("%s Issue %s created: %s\n", green.Render("✓"), issue.ID, issue.Title)
		fmt.Printf("  %s\n", dim.Render("commit .bv/issues/ to track this issue in version control"))
		return nil
	},
}

func init() {
	issueNewCmd.Flags().StringVar(&issueNewBody, "body", "", "Issue body / description")
}
