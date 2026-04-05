package cmd

import (
	"fmt"

	"github.com/may1a/bad-vibes/internal/display"
	"github.com/may1a/bad-vibes/internal/github"
	"github.com/spf13/cobra"
)

var diffTarget targetFlags

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Display the PR diff",
	Long: `Display the PR diff with colored line numbers.

Examples:
  bv diff
  bv diff --pr 42`,
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
