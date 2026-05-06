package cmd

import (
	"os"

	"github.com/may1a/bad-vibes/internal/auth"
	"github.com/may1a/bad-vibes/internal/github"
	"github.com/spf13/cobra"
)

var (
	ghClient *github.Client
)

// SetVersion is called from main.go with the ldflags-injected version string.
func SetVersion(v string) {
	rootCmd.Version = v
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "bv",
	Short: "bad vibes — focused AI-assisted PR review",
	Long: `bad vibes cuts through the noise of PR review.

Surface unresolved threads, post pointed feedback, and track issues
without the noise that GitHub's own UI buries them in.

Targeting modes:
  Explicit:    bv review summary --repo owner/repo --pr 42
  Auto-detect: bv review summary # latest open PR on current branch`,
	Example: `  bv review start 42
  bv review threads
  bv review add cmd/root.go:42 "Needs a guard here"
  bv review finish --approve
  bv issue new "Cache invalidation is too aggressive"`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !requiresRepoContext(cmd) {
			return nil
		}

		token, err := auth.Token()
		if err != nil {
			return err
		}

		ghClient = github.NewClient(token)
		github.SetClient(ghClient)
		return nil
	},
}

func requiresRepoContext(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "review" {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(
		reviewCmd,
		issueCmd,
	)
}
