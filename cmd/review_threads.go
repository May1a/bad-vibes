package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bad-vibes/internal/display"
	"github.com/may1a/bad-vibes/internal/github"
	"github.com/spf13/cobra"
)

var (
	reviewThreadsVerbose       bool
	reviewThreadsPatch         bool
	reviewThreadsTarget        targetFlags
	reviewThreadsAuthor        string
	reviewThreadsExcludeAuthor string
)

var reviewThreadsCmd = &cobra.Command{
	Use:   "threads",
	Short: "Show unresolved review threads",
	Long: `Show unresolved review threads in a compact, readable form.

By default prints one summary per unresolved thread plus a code snippet.
Use --verbose to show every comment in the thread.
Use --patch to include diff hunk context.

Examples:
  bv review threads
  bv review threads --verbose --patch`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		target, err := resolveTarget(cmd, reviewThreadsTarget)
		if err != nil {
			return err
		}
		ref := target.Ref
		threads, err := github.FetchReviewThreads(ghClient, ctx, ref)
		if err != nil {
			return err
		}

		unresolved := github.UnresolvedThreads(threads)
		unresolved = filterThreadsByAuthor(unresolved, reviewThreadsAuthor, reviewThreadsExcludeAuthor)

		if len(unresolved) == 0 {
			fmt.Println(lipgloss.NewStyle().Faint(true).Render("No unresolved threads."))
			return nil
		}

		display.PrintThreads(unresolved, display.ThreadRenderOptions{
			Verbose:     reviewThreadsVerbose,
			ShowDiff:    reviewThreadsPatch,
			ShowSnippet: true,
		})
		return nil
	},
}

func init() {
	addTargetFlags(reviewThreadsCmd, &reviewThreadsTarget)
	reviewThreadsCmd.Flags().BoolVar(&reviewThreadsVerbose, "verbose", false, "Show every comment in each thread")
	reviewThreadsCmd.Flags().BoolVar(&reviewThreadsPatch, "patch", false, "Include diff hunk context")
	reviewThreadsCmd.Flags().StringVar(&reviewThreadsAuthor, "author", "", "Only show threads where a comment is by this author")
	reviewThreadsCmd.Flags().StringVar(&reviewThreadsExcludeAuthor, "exclude-author", "", "Exclude threads where any comment is by this author")
}
