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

var commentsCmd = &cobra.Command{
	Use:   "comments <PR>",
	Short: "Show unresolved review comments",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := resolvePR(args)
		if err != nil {
			return err
		}
		threads, err := github.FetchReviewThreads(ref)
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
		display.PrintThreads(unresolved, anchors)
		return nil
	},
}
