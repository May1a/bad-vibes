package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bad-vibes/internal/git"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/may1a/bad-vibes/internal/session"
	"github.com/spf13/cobra"
)

var reviewStatusTarget targetFlags

var reviewStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the active review session",
	Long: `Show the current review session and any staged comments.

Examples:
  bv review status`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := requireSession(cmd, reviewStatusTarget)
		if err != nil {
			return err
		}

		bold := lipgloss.NewStyle().Bold(true)
		dim := lipgloss.NewStyle().Faint(true)
		yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("#facc15"))
		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))

		fmt.Printf("\n%s  PR #%d · %s/%s\n", bold.Render("Review session"), s.Number, s.Owner, s.Repo)
		fmt.Printf("  %s  %s\n\n", dim.Render("started:"), s.StartedAt.Format("2006-01-02 15:04"))

		if len(s.PendingComments) == 0 {
			fmt.Printf("  %s\n\n", dim.Render("no staged comments"))
		} else {
			count := fmt.Sprintf("%d staged comment(s)", len(s.PendingComments))
			fmt.Printf("  %s\n", yellow.Render(count))
			for i, c := range s.PendingComments {
				loc := fmt.Sprintf("%s:%d", c.Path, c.Line)
				snippet := c.Body
				if len([]rune(snippet)) > 60 {
					snippet = string([]rune(snippet)[:57]) + "..."
				}
				fmt.Printf("  %s  %s  %s\n",
					dim.Render(fmt.Sprintf("%d.", i+1)),
					green.Render(loc),
					dim.Render(snippet),
				)
			}
			fmt.Println()
		}

		fmt.Printf("  %s\n\n", dim.Render("finish with: bv review finish --approve"))
		return nil
	},
}

// requireSession resolves an active review session, using --pr flag or auto-detecting
// a single session for the current repo.
func requireSession(cmd *cobra.Command, flags targetFlags) (model.ReviewSession, error) {
	if flags.pr != "" || flags.repo != "" {
		target, err := resolveTarget(cmd, flags)
		if err != nil {
			return model.ReviewSession{}, err
		}
		s, ok, err := session.Load(target.Ref)
		if err != nil {
			return model.ReviewSession{}, err
		}
		if !ok {
			return model.ReviewSession{}, fmt.Errorf("no review session active for PR #%d\n  try: bv review start %d", target.Ref.Number, target.Ref.Number)
		}
		return s, nil
	}

	// Auto-detect from git context
	repo, err := git.RemoteRepo()
	if err != nil {
		return model.ReviewSession{}, fmt.Errorf("not in a git repo with a GitHub remote\n  try: bv review start --repo owner/repo --pr 42")
	}
	owner, repoName, err := splitRepo(repo)
	if err != nil {
		return model.ReviewSession{}, err
	}

	s, ok, err := session.Active(owner, repoName)
	if err != nil {
		return model.ReviewSession{}, err
	}
	if !ok {
		return model.ReviewSession{}, fmt.Errorf("no review session active for %s/%s\n  try: bv review start <pr>", owner, repoName)
	}
	return s, nil
}

func init() {
	addTargetFlags(reviewStatusCmd, &reviewStatusTarget)
}
