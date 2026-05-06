package cmd

import "github.com/spf13/cobra"

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Manage a code review session",
	Long: `Start, work in, and finish a stateful PR review session.

A review session stages your comments locally and submits them all at once
as a proper GitHub review when you run "bv review finish".

Examples:
  bv review start 42
  bv review threads
  bv review add cmd/root.go:55 "Needs a guard here"
  bv review status
  bv review finish --approve`,
}

func init() {
	reviewCmd.AddCommand(
		reviewStartCmd,
		reviewStatusCmd,
		reviewThreadsCmd,
		reviewAddCmd,
		reviewResolveCmd,
		reviewFinishCmd,
		reviewAbortCmd,
		reviewDiffCmd,
		reviewSummaryCmd,
		reviewPrsCmd,
	)
}
