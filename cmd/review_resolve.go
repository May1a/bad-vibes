package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bad-vibes/internal/github"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/spf13/cobra"
)

var (
	reviewResolveID            string
	reviewResolveTargetCfg     targetFlags
	reviewResolveAuthor        string
	reviewResolveExcludeAuthor string
)

var (
	fetchReviewThreadsForResolve = github.FetchReviewThreads
	resolveThreadForResolve      = github.ResolveThread
)

var reviewResolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Resolve a review thread",
	Long: `Mark a review thread as resolved.

Without --id, resolves the first unresolved thread.
With --id, resolves by GraphQL node ID or numeric index.

Examples:
  bv review resolve
  bv review resolve --id PRRT_abc123
  bv review resolve --id 1
  bv review resolve --id 1 --author coderabbitai`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		target, err := resolveTarget(cmd, reviewResolveTargetCfg)
		if err != nil {
			return err
		}
		ref := target.Ref

		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		dim := lipgloss.NewStyle().Faint(true)

		if isNumericIndex(reviewResolveID) {
			threads, err := fetchReviewThreadsForResolve(ghClient, ctx, ref)
			if err != nil {
				return err
			}
			unresolved := github.UnresolvedThreads(threads)
			unresolved = filterThreadsByAuthor(unresolved, reviewResolveAuthor, reviewResolveExcludeAuthor)
			idx, _ := strconv.Atoi(reviewResolveID)
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

		if reviewResolveID != "" {
			fmt.Println(dim.Render("resolving: ") + reviewResolveID)
			if err := resolveThreadForResolve(ghClient, ctx, reviewResolveID); err != nil {
				return err
			}
			fmt.Println(green.Render("✓") + " Thread resolved.")
			return nil
		}

		// No ID given: resolve first unresolved thread
		threads, err := fetchReviewThreadsForResolve(ghClient, ctx, ref)
		if err != nil {
			return err
		}
		unresolved := github.UnresolvedThreads(threads)
		first, ok := github.FirstUnresolvedThread(unresolved)
		if !ok {
			return fmt.Errorf("no unresolved threads found for PR #%d", ref.Number)
		}
		loc := threadLabelWithTitle(first)
		fmt.Println(dim.Render("resolving: ") + loc)
		if err := resolveThreadForResolve(ghClient, ctx, first.ID); err != nil {
			return err
		}
		fmt.Println(green.Render("✓") + " Resolved " + loc)
		return nil
	},
}

type resolveTargetSelection struct {
	ThreadID    string
	Description string
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

func init() {
	addTargetFlags(reviewResolveCmd, &reviewResolveTargetCfg)
	reviewResolveCmd.Flags().StringVar(&reviewResolveID, "id", "", "Thread ID (GraphQL node ID or numeric index)")
	reviewResolveCmd.Flags().StringVar(&reviewResolveAuthor, "author", "", "Apply author filter (same as bv review threads --author)")
	reviewResolveCmd.Flags().StringVar(&reviewResolveExcludeAuthor, "exclude-author", "", "Apply exclude-author filter")
}
