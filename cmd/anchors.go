package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	anchorutil "github.com/may1a/bv/internal/anchors"
	"github.com/may1a/bv/internal/cache"
	"github.com/may1a/bv/internal/github"
	"github.com/may1a/bv/internal/model"
	"github.com/spf13/cobra"
)

var anchorsTarget targetFlags

func mergeAnchorsForDisplay(localAnchors []model.Anchor, threads []model.ReviewThread, threadsErr error) ([]model.Anchor, string, error) {
	if threadsErr != nil {
		if len(localAnchors) == 0 {
			return nil, "", threadsErr
		}
		return localAnchors, fmt.Sprintf("warning: could not refresh review threads; showing local anchors only: %v", threadsErr), nil
	}
	return anchorutil.Merge(localAnchors, threads), "", nil
}

var anchorsCmd = &cobra.Command{
	Use:   "anchors",
	Short: "List anchors for a PR",
	Long: `List anchors for a pull request.

This includes locally saved anchors plus tags discovered from unresolved thread bodies.

Examples:
  bv anchors`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := resolveTarget(cmd, anchorsTarget)
		if err != nil {
			return err
		}
		ref := target.Ref
		localAnchors, err := cache.ListAnchors(ref)
		if err != nil {
			return err
		}

		threads, err := github.FetchReviewThreads(ghClient, cmd.Context(), ref)
		anchors, warning, err := mergeAnchorsForDisplay(localAnchors, threads, err)
		if err != nil {
			return err
		}
		if warning != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), lipgloss.NewStyle().Faint(true).Render(warning))
		}
		if len(anchors) == 0 {
			fmt.Println(lipgloss.NewStyle().Faint(true).Render("No anchors defined for this PR."))
			return nil
		}

		tag := lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")).Bold(true)
		file := lipgloss.NewStyle().Foreground(lipgloss.Color("#facc15"))
		dim := lipgloss.NewStyle().Faint(true)

		for _, a := range anchors {
			location := ""
			if a.Path != "" {
				if a.Line > 0 {
					location = fmt.Sprintf("%s:%d", a.Path, a.Line)
				} else {
					location = a.Path
				}
			}
			snippet := strings.Join(strings.Fields(a.Body), " ")
			if len(snippet) > 120 {
				snippet = snippet[:117] + "..."
			}
			fmt.Printf("  %s  %s  %s\n",
				tag.Render("#"+a.Tag),
				file.Render(location),
				dim.Render(snippet),
			)
		}
		return nil
	},
}

func init() {
	addTargetFlags(anchorsCmd, &anchorsTarget)
}
