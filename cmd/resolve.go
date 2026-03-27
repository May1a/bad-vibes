package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/may/bad-vibes/internal/cache"
	"github.com/may/bad-vibes/internal/github"
	"github.com/may/bad-vibes/internal/model"
	"github.com/may/bad-vibes/internal/tui"
	"github.com/spf13/cobra"
)

var resolveID string

var resolveCmd = &cobra.Command{
	Use:   "resolve <PR>",
	Short: "Resolve a review thread",
	Long: `Mark a review thread as resolved.

Without --id: launches an interactive list of unresolved threads.
With --id: resolves the given thread ID (GraphQL node ID or #anchor-tag) directly.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := resolvePR(args)
		if err != nil {
			return err
		}

		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))

		if resolveID != "" {
			threadID := resolveID
			// Support #anchor-tag
			if strings.HasPrefix(threadID, "#") {
				tag := strings.TrimPrefix(threadID, "#")
				anchors, err := cache.ListAnchors(ref)
				if err != nil {
					return err
				}
				found := false
				for _, a := range anchors {
					if a.Tag == tag {
						threadID = a.ThreadID
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("no anchor %q found for PR #%d", resolveID, ref.Number)
				}
			}
			if err := github.ResolveThread(threadID); err != nil {
				return err
			}
			fmt.Println(green.Render("✓") + " Thread resolved.")
			return nil
		}

		// Interactive mode
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

		resolved, err := tui.RunResolveFlow(unresolved)
		if err != nil {
			return err
		}
		if len(resolved) == 0 {
			fmt.Println(lipgloss.NewStyle().Faint(true).Render("No threads resolved."))
		} else {
			for _, id := range resolved {
				fmt.Println(green.Render("✓") + " " + id)
			}
			fmt.Println(red.Bold(false).Faint(true).Render(fmt.Sprintf("%d thread(s) resolved.", len(resolved))))
		}
		return nil
	},
}

func init() {
	resolveCmd.Flags().StringVar(&resolveID, "id", "", "Thread ID (GraphQL node ID or #anchor-tag) to resolve without TUI")
}
