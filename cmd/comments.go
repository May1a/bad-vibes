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
	commentsTarget  targetFlags
)

var commentsCmd = &cobra.Command{
	Use:   "comments",
	Short: "Show unresolved review comments",
	Long: `Show unresolved review threads in a compact, readable form.

By default this prints a short summary for each unresolved thread plus a code snippet.
Use --verbose to show every comment in the thread.
Use --patch to include diff hunk context.

Targeting:
  Prefer --repo/--pr in scripts or outside a checkout.
  If omitted, bv uses the current repo and the latest open PR on the current branch.

Examples:
  bv comments --repo owner/repo --pr 42
  bv comments --pr 42
  bv comments      # auto-detect from current branch
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
