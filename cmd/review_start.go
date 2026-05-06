package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bad-vibes/internal/github"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/may1a/bad-vibes/internal/session"
	"github.com/spf13/cobra"
)

var reviewStartTarget targetFlags

var reviewStartCmd = &cobra.Command{
	Use:   "start [pr]",
	Short: "Start a review session for a PR",
	Long: `Start a stateful review session for a pull request.

Comments added with "bv review add" are staged locally until you run
"bv review finish" to submit them all as a GitHub review.

Examples:
  bv review start 42
  bv review start --repo owner/repo --pr 42`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if len(args) == 1 && reviewStartTarget.pr == "" {
			reviewStartTarget.pr = args[0]
		}

		target, err := resolveTarget(cmd, reviewStartTarget)
		if err != nil {
			return err
		}
		ref := target.Ref

		existing, ok, err := session.Load(ref)
		if err != nil {
			return err
		}
		if ok {
			dim := lipgloss.NewStyle().Faint(true)
			fmt.Printf("Review session already active for PR #%d (%s/%s)\n", existing.Number, existing.Owner, existing.Repo)
			fmt.Printf("  %s\n", dim.Render("run: bv review status"))
			return nil
		}

		pr, err := github.FetchPRMetadata(ghClient, ctx, ref)
		if err != nil {
			return err
		}

		s := model.ReviewSession{
			Owner:           ref.Owner,
			Repo:            ref.Repo,
			Number:          ref.Number,
			PRID:            pr.ID,
			HeadSHA:         pr.HeadSHA,
			StartedAt:       time.Now(),
			PendingComments: []model.PendingComment{},
		}
		if err := session.Save(s); err != nil {
			return fmt.Errorf("saving session: %w", err)
		}

		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		dim := lipgloss.NewStyle().Faint(true)
		fmt.Printf("%s Review session started: PR #%d — %s\n", green.Render("✓"), pr.Number, pr.Title)
		fmt.Printf("  %s\n", dim.Render("add comments with: bv review add <file>:<line> \"body\""))
		fmt.Printf("  %s\n", dim.Render("submit with:       bv review finish --approve"))
		return nil
	},
}

func init() {
	addTargetFlags(reviewStartCmd, &reviewStartTarget)
}
