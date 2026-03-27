package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/may/bad-vibes/internal/cache"
	"github.com/may/bad-vibes/internal/display"
	"github.com/may/bad-vibes/internal/github"
	"github.com/may/bad-vibes/internal/model"
	"github.com/spf13/cobra"
)

var (
	commentsVerbose bool
	commentsPatch   bool
)

var commentsCmd = &cobra.Command{
	Use:   "comments [PR]",
	Short: "Show unresolved review comments",
	Long: `Show unresolved review threads in a compact, readable form.

By default this prints a short summary for each unresolved thread.
Use --verbose to show every comment in the thread.
Use --patch to include diff hunk context.

Examples:
  bv comments      # auto-detect PR from current branch
  bv comments 42   # show comments for PR #42
  bv comments --verbose --patch`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		ref, err := resolvePR(args)
		if err != nil {
			return err
		}
		threads, err := github.FetchReviewThreads(ghClient, ctx, ref)
		if err != nil {
			return err
		}

		var unresolved []model.ReviewThread
		for _, t := range threads {
			if !t.IsResolved {
				unresolved = append(unresolved, t)
			}
		}

		if len(unresolved) == 0 {
			fmt.Println(lipgloss.NewStyle().Faint(true).Render("No unresolved threads."))
			return nil
		}

		anchors, _ := cache.ListAnchors(ref)
		display.PrintThreads(unresolved, anchors, display.ThreadRenderOptions{
			Verbose:  commentsVerbose,
			ShowDiff: commentsPatch,
		})
		return nil
	},
}

func init() {
	commentsCmd.Flags().BoolVar(&commentsVerbose, "verbose", false, "Show every comment in each thread")
	commentsCmd.Flags().BoolVar(&commentsPatch, "patch", false, "Include diff hunk context")
}
