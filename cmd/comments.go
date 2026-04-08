package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	anchorutil "github.com/may1a/bad-vibes/internal/anchors"
	"github.com/may1a/bad-vibes/internal/cache"
	"github.com/may1a/bad-vibes/internal/display"
	"github.com/may1a/bad-vibes/internal/github"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/spf13/cobra"
)

var (
	commentsVerbose       bool
	commentsPatch         bool
	commentsTarget        targetFlags
	commentsAuthor        string
	commentsExcludeAuthor string
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
		unresolved = filterThreadsByAuthor(unresolved, commentsAuthor, commentsExcludeAuthor)

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
	commentsCmd.Flags().StringVar(&commentsAuthor, "author", "", "Only show threads where a comment is by this author")
	commentsCmd.Flags().StringVar(&commentsExcludeAuthor, "exclude-author", "", "Exclude threads where any comment is by this author")
}

func filterThreadsByAuthor(threads []model.ReviewThread, author, excludeAuthor string) []model.ReviewThread {
	if author == "" && excludeAuthor == "" {
		return threads
	}
	filtered := make([]model.ReviewThread, 0, len(threads))
	for _, t := range threads {
		if author != "" && !threadHasAuthor(t, author) {
			continue
		}
		if excludeAuthor != "" && threadHasAuthor(t, excludeAuthor) {
			continue
		}
		filtered = append(filtered, t)
	}
	return filtered
}

func normalizeAuthor(author string) string {
	author = strings.TrimSpace(strings.TrimPrefix(author, "@"))
	author = strings.ToLower(author)
	return strings.TrimSuffix(author, "[bot]")
}

func threadHasAuthor(t model.ReviewThread, author string) bool {
	wanted := normalizeAuthor(author)
	for _, c := range t.Comments {
		if normalizeAuthor(c.Author) == wanted {
			return true
		}
	}
	return false
}
