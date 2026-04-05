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

Surface only unresolved comments, post pointed feedback, and resolve
threads without the garbage that gh dumps by default.

Targeting modes:
  Explicit:    bv summary --repo owner/repo --pr 42
  Auto-detect: bv summary # latest open PR on current branch`,
	Example: `  bv summary
  bv comments
  bv comment cmd/root.go:42 "Needs a guard here"`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !requiresRepoContext(cmd) {
			return nil
		}

		// Resolve GitHub auth token
		token, err := auth.Token()
		if err != nil {
			return err
		}

		// Initialize GitHub client with retry logic and rate limit handling
		ghClient = github.NewClient(token)
		github.SetClient(ghClient)
		return nil
	},
}

func requiresRepoContext(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "diff", "comments", "comment", "resolve", "summary", "anchors", "prs":
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(
		diffCmd,
		commentsCmd,
		commentCmd,
		resolveCmd,
		summaryCmd,
		anchorsCmd,
		prsCmd,
	)
}
