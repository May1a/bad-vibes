package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	anchorutil "github.com/may1a/bad-vibes/internal/anchors"
	"github.com/may1a/bad-vibes/internal/cache"
	"github.com/may1a/bad-vibes/internal/github"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/spf13/cobra"
)

var (
	resolveID        string
	resolveTargetCfg targetFlags
)

var (
	fetchReviewThreadsForResolve = github.FetchReviewThreads
	resolveThreadForResolve      = github.ResolveThread
	listAnchorsForResolve        = cache.ListAnchors
)

var resolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Resolve a review thread",
	Long: `Mark a review thread as resolved.

Without --id, resolves the first unresolved thread shown by bv comments.
With --id, resolves the given thread ID (GraphQL node ID, #anchor-tag, or numeric index) directly.

Examples:
  bv resolve
  bv resolve --pr 42 --id #perf
  bv resolve --id PRRT_abc123
  bv resolve --id 1`,
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

		if isNumericIndex(resolveID) {
			threads, err := fetchReviewThreadsForResolve(ghClient, ctx, ref)
			if err != nil {
				return err
			}
			unresolved := github.UnresolvedThreads(threads)
			idx, _ := strconv.Atoi(resolveID)
			if idx < 1 || idx > len(unresolved) {
				return fmt.Errorf("thread index #%d out of range (1–%d)", idx, len(unresolved))
			}
			t := unresolved[idx-1]
			loc := threadLabelWithTitle(t)
			fmt.Println(dim.Render("resolving: ") + loc)
			if err := resolveThreadForResolve(ghClient, ctx, t.ID); err != nil {
				return err
			}
			fmt.Println(green.Render("✓") + " Resolved " + loc)
			return nil
		}

		if resolveID != "" && !strings.HasPrefix(resolveID, "#") {
			fmt.Println(dim.Render("resolving: ") + resolveID)
			if err := resolveThreadForResolve(ghClient, ctx, resolveID); err != nil {
				return err
			}
			fmt.Println(green.Render("✓") + " Thread resolved.")
			return nil
		}

		var localAnchors []model.Anchor
		if strings.HasPrefix(resolveID, "#") {
			localAnchors, err = listAnchorsForResolve(ref)
			if err != nil {
				return err
			}
		}

		threads, err := fetchReviewThreadsForResolve(ghClient, ctx, ref)
		if err != nil {
			return err
		}

		selection, err := resolveSelection(ref, resolveID, localAnchors, threads)
		if err != nil {
			return err
		}
		if selection.ThreadID == "" {
			return fmt.Errorf("no unresolved threads found for PR #%d", ref.Number)
		}

		fmt.Println(dim.Render("resolving: ") + selection.Description)
		if err := resolveThreadForResolve(ghClient, ctx, selection.ThreadID); err != nil {
			return err
		}
		fmt.Println(green.Render("✓") + " Resolved " + selection.Description)
		return nil
	},
}

type resolveTargetSelection struct {
	ThreadID    string
	Description string
}

func resolveSelection(ref model.PRRef, rawID string, localAnchors []model.Anchor, threads []model.ReviewThread) (resolveTargetSelection, error) {
	unresolved := github.UnresolvedThreads(threads)
	if rawID == "" {
		first, ok := github.FirstUnresolvedThread(unresolved)
		if !ok {
			return resolveTargetSelection{}, nil
		}
		return resolveTargetSelection{
			ThreadID:    first.ID,
			Description: threadLabelWithTitle(first),
		}, nil
	}

	if !strings.HasPrefix(rawID, "#") {
		return resolveTargetSelection{
			ThreadID:    rawID,
			Description: rawID,
		}, nil
	}

	tag := strings.TrimPrefix(rawID, "#")
	if strings.EqualFold(tag, "PR") {
		id, ok, err := github.LookupUnresolvedThreadID(unresolved, "", 0, "")
		if err != nil {
			return resolveTargetSelection{}, err
		}
		if !ok {
			return resolveTargetSelection{}, fmt.Errorf("no unresolved PR-level thread found for PR #%d", ref.Number)
		}
		return resolveTargetSelection{
			ThreadID:    id,
			Description: "PR-level comment",
		}, nil
	}

	anchor, err := anchorutil.Resolve(localAnchors, unresolved, tag)
	if err != nil {
		return resolveTargetSelection{}, fmt.Errorf("%w for PR #%d", err, ref.Number)
	}

	id, ok, err := github.LookupUnresolvedThreadID(unresolved, anchor.Path, anchor.Line, anchor.Body)
	if err != nil {
		return resolveTargetSelection{}, err
	}
	if !ok && anchor.ThreadID != "" && hasThreadID(unresolved, anchor.ThreadID) {
		id = anchor.ThreadID
		ok = true
	}
	if !ok {
		return resolveTargetSelection{}, fmt.Errorf("no unresolved thread found for anchor %q", rawID)
	}
	return resolveTargetSelection{
		ThreadID:    id,
		Description: formatAnchorLocation(anchor),
	}, nil
}

func hasThreadID(threads []model.ReviewThread, threadID string) bool {
	for _, thread := range threads {
		if thread.ID == threadID && !thread.IsResolved {
			return true
		}
	}
	return false
}

func threadLabel(thread model.ReviewThread) string {
	if thread.Path == "" {
		return "PR-level comment"
	}
	if thread.Line > 0 {
		return fmt.Sprintf("%s:%d", thread.Path, thread.Line)
	}
	return thread.Path
}

func formatAnchorLocation(anchor model.Anchor) string {
	if anchor.Path == "" {
		return "PR-level comment"
	}
	if anchor.Line > 0 {
		return fmt.Sprintf("%s:%d", anchor.Path, anchor.Line)
	}
	return anchor.Path
}

func init() {
	addTargetFlags(resolveCmd, &resolveTargetCfg)
	resolveCmd.Flags().StringVar(&resolveID, "id", "", "Thread ID (GraphQL node ID, #anchor-tag, or numeric index)")
}

func isNumericIndex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func threadLabelWithTitle(t model.ReviewThread) string {
	loc := threadLabel(t)
	if len(t.Comments) > 0 {
		body := t.Comments[0].Body
		if title := extractFirstBoldTitle(body); title != "" {
			return fmt.Sprintf("%s — %q", loc, title)
		}
	}
	return loc
}

func extractFirstBoldTitle(body string) string {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 4 && strings.HasPrefix(trimmed, "**") && strings.HasSuffix(trimmed, "**") {
			return strings.TrimSuffix(strings.TrimPrefix(trimmed, "**"), "**")
		}
	}
	return ""
}
