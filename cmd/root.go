package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/may/bad-vibes/internal/auth"
	"github.com/may/bad-vibes/internal/git"
	"github.com/may/bad-vibes/internal/github"
	"github.com/may/bad-vibes/internal/model"
	"github.com/may/bad-vibes/internal/parse"
	"github.com/spf13/cobra"
)

var (
	bvVersion    string
	detectedRepo string // "owner/repo" from git remote origin
	detectedBranch string // current git branch
	ghClient     *github.Client
)

// SetVersion is called from main.go with the ldflags-injected version string.
func SetVersion(v string) {
	bvVersion = v
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
threads — without the garbage that gh dumps by default.`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Detect repo from git remote
		repo, err := git.RemoteRepo()
		if err != nil {
			return err
		}
		if !strings.Contains(repo, "/") {
			return fmt.Errorf("invalid repo format from git remote: %q (expected owner/repo)", repo)
		}
		detectedRepo = repo

		// Detect current branch
		branch, err := git.CurrentBranch()
		if err != nil {
			return err
		}
		detectedBranch = branch

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

func init() {
	rootCmd.AddCommand(
		reviewCmd,
		commentsCmd,
		commentCmd,
		resolveCmd,
		summaryCmd,
		anchorsCmd,
		prsCmd,
	)
}

// repoRef returns a PRRef with just owner/repo populated (no number).
func repoRef() model.PRRef {
	parts := strings.SplitN(detectedRepo, "/", 2)
	if len(parts) == 2 {
		return model.PRRef{Owner: parts[0], Repo: parts[1]}
	}
	return model.PRRef{}
}

// resolvePR resolves a PRRef from CLI args or auto-detects the most recent
// open PR on the current branch when no arg is given.
func resolvePR(args []string) (model.PRRef, error) {
	if len(args) == 1 {
		return parse.ParseRef(args[0], detectedRepo)
	}
	// No arg: find latest open PR on current branch
	base := repoRef()
	pr, err := github.LatestOpenPR(ghClient, context.Background(), base, detectedBranch)
	if err != nil {
		return model.PRRef{}, err
	}
	fmt.Fprintf(os.Stderr, "  → PR #%d: %s\n\n", pr.Number, pr.Title)
	return model.PRRef{Owner: base.Owner, Repo: base.Repo, Number: pr.Number}, nil
}
