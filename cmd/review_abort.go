package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/may1a/bad-vibes/internal/session"
	"github.com/spf13/cobra"
)

var reviewAbortTarget targetFlags

var reviewAbortCmd = &cobra.Command{
	Use:   "abort",
	Short: "Discard the review session without submitting",
	Long: `Discard the active review session and all staged comments without submitting.

Examples:
  bv review abort`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := requireSession(cmd, reviewAbortTarget)
		if err != nil {
			return err
		}
		ref := model.PRRef{Owner: s.Owner, Repo: s.Repo, Number: s.Number}

		if err := session.Delete(ref); err != nil {
			return fmt.Errorf("deleting session: %w", err)
		}

		dim := lipgloss.NewStyle().Faint(true)
		fmt.Printf("Review session for PR #%d discarded (%d staged comment(s) lost).\n",
			ref.Number,
			len(s.PendingComments),
		)
		fmt.Printf("  %s\n", dim.Render("start a new session with: bv review start <pr>"))
		return nil
	},
}

func init() {
	addTargetFlags(reviewAbortCmd, &reviewAbortTarget)
}
