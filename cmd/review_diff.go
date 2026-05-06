package cmd

import (
	"fmt"

	"github.com/may1a/bad-vibes/internal/display"
	"github.com/may1a/bad-vibes/internal/github"
	"github.com/spf13/cobra"
)

var reviewDiffTarget targetFlags

var reviewDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Display the PR diff",
	Long: `Display the PR diff with colored line numbers.

Examples:
  bv review diff
  bv review diff --pr 42`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		target, err := resolveTarget(cmd, reviewDiffTarget)
		if err != nil {
			return err
		}
		ref := target.Ref
		d, err := github.FetchDiff(ghClient, ctx, ref)
		if err != nil {
			return err
		}
		if d == "" {
			fmt.Println("No diff available.")
			return nil
		}
		display.PrintDiff(d)
		return nil
	},
}

func init() {
	addTargetFlags(reviewDiffCmd, &reviewDiffTarget)
}
