package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bad-vibes/internal/github"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/may1a/bad-vibes/internal/session"
	"github.com/spf13/cobra"
)

var (
	reviewFinishApprove        bool
	reviewFinishRequestChanges bool
	reviewFinishBody           string
	reviewFinishTarget         targetFlags
)

var reviewFinishCmd = &cobra.Command{
	Use:   "finish",
	Short: "Submit the review and end the session",
	Long: `Submit all staged comments as a GitHub review and end the review session.

Pass --approve or --request-changes to set the review event type.
Without either flag the review is submitted as a plain COMMENT.

Examples:
  bv review finish --approve
  bv review finish --request-changes --body "Several issues to address"
  bv review finish --body "Looks good, just left a few notes"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if reviewFinishApprove && reviewFinishRequestChanges {
			return fmt.Errorf("--approve and --request-changes are mutually exclusive")
		}

		s, err := requireSession(cmd, reviewFinishTarget)
		if err != nil {
			return err
		}
		ref := model.PRRef{Owner: s.Owner, Repo: s.Repo, Number: s.Number}

		event := github.ReviewEventComment
		switch {
		case reviewFinishApprove:
			event = github.ReviewEventApprove
		case reviewFinishRequestChanges:
			event = github.ReviewEventRequestChanges
		}

		comments := make([]github.ReviewCommentInput, len(s.PendingComments))
		for i, c := range s.PendingComments {
			comments[i] = github.ReviewCommentInput{
				Path: c.Path,
				Line: c.Line,
				Side: c.Side,
				Body: c.Body,
			}
		}

		dim := lipgloss.NewStyle().Faint(true)
		fmt.Printf("%s Submitting review (%s, %d comment(s))…\n",
			dim.Render("→"),
			string(event),
			len(comments),
		)

		if err := github.SubmitReview(ghClient, cmd.Context(), ref, s.HeadSHA, event, reviewFinishBody, comments); err != nil {
			return err
		}

		if err := session.Delete(ref); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: review submitted but could not clean up session: %v\n", err)
		}

		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		fmt.Printf("%s Review submitted for PR #%d (%s/%s)\n", green.Render("✓"), ref.Number, ref.Owner, ref.Repo)
		return nil
	},
}

func init() {
	addTargetFlags(reviewFinishCmd, &reviewFinishTarget)
	reviewFinishCmd.Flags().BoolVar(&reviewFinishApprove, "approve", false, "Approve the pull request")
	reviewFinishCmd.Flags().BoolVar(&reviewFinishRequestChanges, "request-changes", false, "Request changes on the pull request")
	reviewFinishCmd.Flags().StringVar(&reviewFinishBody, "body", "", "Overall review body text")
}
