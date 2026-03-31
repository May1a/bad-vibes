package cmd

import (
	"fmt"

	"github.com/may/bad-vibes/internal/display"
	"github.com/may/bad-vibes/internal/github"
	"github.com/spf13/cobra"
)

var diffTarget targetFlags

var diffCmd = &cobra.Command{
	Use:     "diff",
	Aliases: []string{"show", "review"},
	Short:   "Display the PR diff",
	Long: `Display the PR diff with colored line numbers.

Targeting:
  Prefer --repo/--pr in scripts or outside a checkout.
  If omitted, bv uses the current repo and the latest open PR on the current branch.

Examples:
  bv diff --repo owner/repo --pr 42
  bv diff --pr 42
  bv diff      # auto-detect from current branch
  bv show --pr 42`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		target, err := resolveTarget(cmd, diffTarget)
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
	addTargetFlags(diffCmd, &diffTarget)
}
