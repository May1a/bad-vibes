package cmd

import (
	"fmt"

	"github.com/may/bad-vibes/internal/display"
	"github.com/may/bad-vibes/internal/github"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review [PR]",
	Short: "Display the PR diff",
	Long: `Display the PR diff with colored line numbers.

Examples:
  bv review      # auto-detect PR from current branch
  bv review 42   # show diff for PR #42`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		ref, err := resolvePR(args)
		if err != nil {
			return err
		}
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
