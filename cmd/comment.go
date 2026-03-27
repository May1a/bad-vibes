package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/may/bad-vibes/internal/cache"
	"github.com/may/bad-vibes/internal/github"
	"github.com/may/bad-vibes/internal/model"
	"github.com/may/bad-vibes/internal/tui"
	"github.com/spf13/cobra"
)

var commentCmd = &cobra.Command{
	Use:   "comment [PR]",
	Short: "Leave an inline review comment",
	Long: `Interactive wizard to leave an inline review comment.

Walks through: select file → enter line number → write comment → optionally tag with anchor → confirm.

Examples:
  bv comment      # auto-detect PR from current branch
  bv comment 42   # comment on PR #42`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		ref, err := resolvePR(args)
		if err != nil {
			return err
		}

		pr, files, err := github.FetchPR(ghClient, ctx, ref)
		if err != nil {
			return err
		}

		// Cache HeadSHA for future use
		prCache, _ := cache.Load(ref)
		prCache.PRID = pr.ID
		prCache.HeadSHA = pr.HeadSHA
		prCache.Owner = ref.Owner
		prCache.Repo = ref.Repo
		prCache.Number = ref.Number
		_ = cache.Save(ref, prCache)

		result, err := tui.RunCommentFlow(pr, ref, files)
		if err != nil {
			return err
		}
		if result == nil {
			fmt.Println(lipgloss.NewStyle().Faint(true).Render("Cancelled."))
			return nil
		}

		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		fmt.Println(green.Render("✓") + " Comment posted.")

		// Store anchor if the user tagged one
		if result.AnchorTag != "" {
			// Re-fetch threads to get the real GraphQL node ID for the new thread.
			threadNodeID, ok, err := github.FindUnresolvedThreadAt(ghClient, ctx, ref, result.Path, result.Line, result.Body)
			if err != nil {
				fmt.Println(lipgloss.NewStyle().Faint(true).Render("(anchor not saved: " + err.Error() + ")"))
				return nil
			}
			if !ok {
				fmt.Println(lipgloss.NewStyle().Faint(true).Render("(anchor not saved: could not resolve posted thread ID)"))
				return nil
			}
			anchor := model.Anchor{
				Tag:      result.AnchorTag,
				Path:     result.Path,
				Line:     result.Line,
				Body:     result.Body,
				Created:  time.Now(),
				ThreadID: threadNodeID, // real GraphQL node ID (empty only if lookup fails)
			}
			if err := cache.AddAnchor(ref, anchor); err != nil {
				fmt.Println(lipgloss.NewStyle().Faint(true).Render("(anchor not saved: " + err.Error() + ")"))
			} else {
				fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")).Render(
					"⚓ anchor #" + result.AnchorTag + " saved",
				))
			}
		}

		return nil
	},
}
