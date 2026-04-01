package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	anchorutil "github.com/may/bad-vibes/internal/anchors"
	"github.com/may/bad-vibes/internal/cache"
	"github.com/may/bad-vibes/internal/github"
	"github.com/may/bad-vibes/internal/model"
	"github.com/may/bad-vibes/internal/tui"
	"github.com/spf13/cobra"
)

var (
	resolveID        string
	resolveTargetCfg targetFlags
)

var resolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Resolve a review thread",
	Long: `Mark a review thread as resolved.

Without --id: launches an interactive list of unresolved threads.
With --id: resolves the given thread ID (GraphQL node ID or #anchor-tag) directly.

Targeting:
  Prefer --repo/--pr in scripts or outside a checkout.
  If omitted, bv uses the current repo and the latest open PR on the current branch.

Examples:
  bv resolve --repo owner/repo --pr 42 --id PRRT_abc123
  bv resolve --pr 42 --id #perf
  bv resolve                    # auto-detect interactive mode`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		target, err := resolveTarget(cmd, resolveTargetCfg)
		if err != nil {
			return err
		}
		ref := target.Ref

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
					// Anchor lookup — use local anchors first, then discover live tags from unresolved threads.
					localAnchors, err := cache.ListAnchors(ref)
					if err != nil {
						return err
					}
					threads, err := github.FetchReviewThreads(ghClient, ctx, ref)
					if err != nil {
						return err
					}
					anchor, err := anchorutil.Resolve(localAnchors, threads, tag)
					if err != nil {
						return fmt.Errorf("%w for PR #%d", err, ref.Number)
					}
					// Resolve the live thread ID by location (symlink dereference).
					id, ok, err := github.FindUnresolvedThreadAt(ghClient, ctx, ref, anchor.Path, anchor.Line, anchor.Body)
					if err != nil {
						return err
					}
					if !ok && anchor.Body != "" {
						id, ok, err = github.FindUnresolvedThreadAt(ghClient, ctx, ref, anchor.Path, anchor.Line, "")
						if err != nil {
							return err
						}
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
	addTargetFlags(resolveCmd, &resolveTargetCfg)
	resolveCmd.Flags().StringVar(&resolveID, "id", "", "Thread ID (GraphQL node ID or #anchor-tag) to resolve without TUI")
}
