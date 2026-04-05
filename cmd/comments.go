package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	anchorutil "github.com/may1a/bv/internal/anchors"
	"github.com/may1a/bv/internal/cache"
	"github.com/may1a/bv/internal/display"
	"github.com/may1a/bv/internal/github"
	"github.com/spf13/cobra"
)

var (
	commentsVerbose bool
	commentsPatch   bool
	commentsTarget  targetFlags
)

var commentsCmd = &cobra.Command{
	Use:   "comments",
	Short: "Show unresolved review comments",
	Long: `Show unresolved review threads in a compact, readable form.

By default this prints one summary per unresolved thread plus a code snippet.
Use --verbose to show every comment in the thread.
Use --patch to include diff hunk context.

Examples:
  bv comments
  bv comments --verbose --patch`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		target, err := resolveTarget(cmd, commentsTarget)
		if err != nil {
			return err
		}
		ref := target.Ref
		threads, err := github.FetchReviewThreads(ghClient, ctx, ref)
		if err != nil {
			return err
		}

		unresolved := github.UnresolvedThreads(threads)

		if len(unresolved) == 0 {
			fmt.Println(lipgloss.NewStyle().Faint(true).Render("No unresolved threads."))
			return nil
		}

		localAnchors, _ := cache.ListAnchors(ref)
		display.PrintThreads(unresolved, anchorutil.Merge(localAnchors, unresolved), display.ThreadRenderOptions{
			Verbose:     commentsVerbose,
			ShowDiff:    commentsPatch,
			ShowSnippet: true,
		})
		return nil
	},
}

func init() {
	addTargetFlags(commentsCmd, &commentsTarget)
	commentsCmd.Flags().BoolVar(&commentsVerbose, "verbose", false, "Show every comment in each thread")
	commentsCmd.Flags().BoolVar(&commentsPatch, "patch", false, "Include diff hunk context")
}
