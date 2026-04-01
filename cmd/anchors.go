package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	anchorutil "github.com/may/bad-vibes/internal/anchors"
	"github.com/may/bad-vibes/internal/cache"
	"github.com/may/bad-vibes/internal/github"
	"github.com/spf13/cobra"
)

var anchorsTarget targetFlags

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
		if err != nil && len(localAnchors) == 0 {
			return err
		}

		anchors := anchorutil.Merge(localAnchors, threads)
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
