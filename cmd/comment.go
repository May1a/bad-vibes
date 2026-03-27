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
	Use:   "comment <PR>",
	Short: "Leave an inline review comment",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := resolvePR(args)
		if err != nil {
			return err
		}

		pr, files, err := github.FetchPR(ref)
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
			// The REST review endpoint doesn't return it, but the thread is visible
			// immediately via GraphQL after posting.
			threadNodeID, _, _ := github.FindUnresolvedThreadAt(ref, result.Path, result.Line)
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
