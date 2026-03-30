package cmd

import (
	"fmt"

	"github.com/may/bad-vibes/internal/display"
	"github.com/may/bad-vibes/internal/github"
	"github.com/spf13/cobra"
)

var reviewTarget targetFlags

var reviewCmd = &cobra.Command{
	Use:   "review [PR]",
	Short: "Display the PR diff",
	Long: `Display the PR diff with colored line numbers.

Targeting:
  Prefer --repo/--pr in scripts or outside a checkout.
  If omitted, bv uses the current repo and the latest open PR on the current branch.

Examples:
  bv review --repo owner/repo --pr 42
  bv review --pr 42
  bv review 42   # positional shorthand
  bv review      # auto-detect from current branch`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		target, err := resolveTarget(cmd, reviewTarget, args)
		if err != nil {
			return err
		}
		ref := target.Ref
		diff, err := github.FetchDiff(ghClient, ctx, ref)
		if err != nil {
			return err
		}
		if diff == "" {
			fmt.Println("No diff available.")
			return nil
		}
		display.PrintDiff(diff)
		return nil
	},
}

func init() {
	addTargetFlags(reviewCmd, &reviewTarget)
}
