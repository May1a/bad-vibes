package cmd

import "github.com/spf13/cobra"

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage repo-level issues",
	Long: `Track issues found during code review in the repo's .bv/issues/ directory.

Issues are plain JSON files stored in the repository and meant to be committed.

Examples:
  bv issue new "Cache invalidation is too aggressive"
  bv issue list
  bv issue show 0001
  bv issue close 0001`,
}

func init() {
	issueCmd.AddCommand(
		issueListCmd,
		issueNewCmd,
		issueShowCmd,
		issueCloseCmd,
		issueReopenCmd,
	)
}
