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
	Use:   "resolve [PR]",
	Short: "Resolve a review thread",
	Long: `Mark a review thread as resolved.

Without --id: launches an interactive list of unresolved threads.
With --id: resolves the given thread ID (GraphQL node ID or #anchor-tag) directly.

Examples:
  bv resolve                    # interactive mode
  bv resolve --id PRRT_abc123   # resolve by GraphQL node ID
  bv resolve --id #perf         # resolve by anchor tag
  bv resolve --id #PR           # resolve first unresolved PR-level thread`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		ref, err := resolvePR(args)
		if err != nil {
			return err
		}

		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		dim := lipgloss.NewStyle().Faint(true)

		if resolveID != "" {
			threadID := resolveID
			var resolveDesc string
			// Support #anchor-tag and the special #PR shorthand for PR-level threads.
			if strings.HasPrefix(threadID, "#") {
				tag := strings.TrimPrefix(threadID, "#")

				// #PR resolves the first unresolved PR-level thread (no path).
				if strings.EqualFold(tag, "PR") {
					id, ok, err := github.FindUnresolvedThreadAt(ghClient, ctx, ref, "", 0, "")
					if err != nil {
						return err
					}
					if !ok {
						return fmt.Errorf("no unresolved PR-level thread found for PR #%d", ref.Number)
					}
					threadID = id
					resolveDesc = "PR-level comment"
				} else {
					// Anchor lookup — symlink style: use path+line to find the live thread ID.
					anchors, err := cache.ListAnchors(ref)
					if err != nil {
						return err
					}
					var anchor *model.Anchor
					for i := range anchors {
						if anchors[i].Tag == tag {
							anchor = &anchors[i]
							break
						}
					}
					if anchor == nil {
						return fmt.Errorf("no anchor %q found for PR #%d", resolveID, ref.Number)
					}
					// Resolve the live thread ID by location (symlink dereference).
					id, ok, err := github.FindUnresolvedThreadAt(ghClient, ctx, ref, anchor.Path, anchor.Line, anchor.Body)
					if err != nil {
						return err
					}
					if ok {
						threadID = id
					} else if anchor.ThreadID != "" {
						// Fallback: use whatever was stored (may fail, but worth trying).
						threadID = anchor.ThreadID
					} else {
						return fmt.Errorf("no unresolved thread found for anchor %q", resolveID)
					}
					resolveDesc = fmt.Sprintf("%s:%d", anchor.Path, anchor.Line)
				}
			} else {
				resolveDesc = threadID
			}
			fmt.Println(dim.Render("resolving: ") + resolveDesc)
			if err := github.ResolveThread(ghClient, ctx, threadID); err != nil {
				return err
			}
			fmt.Println(green.Render("✓") + " Thread resolved.")
			return nil
		}

		// Interactive mode
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

		resolved, err := tui.RunResolveFlow(unresolved)
		if err != nil {
			return err
		}
		if len(resolved) == 0 {
			fmt.Println(lipgloss.NewStyle().Faint(true).Render("No threads resolved."))
		} else {
			for _, r := range resolved {
				fmt.Println(green.Render("✓") + " " + r.Title)
			}
			fmt.Println(dim.Render(fmt.Sprintf("%d thread(s) resolved.", len(resolved))))
		}
		return nil
	},
}

func init() {
	resolveCmd.Flags().StringVar(&resolveID, "id", "", "Thread ID (GraphQL node ID or #anchor-tag) to resolve without TUI")
}
