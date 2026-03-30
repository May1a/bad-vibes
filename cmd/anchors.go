package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/may/bad-vibes/internal/cache"
	"github.com/spf13/cobra"
)

var anchorsTarget targetFlags

var anchorsCmd = &cobra.Command{
	Use:   "anchors [PR]",
	Short: "List local anchors for a PR",
	Long: `List local anchors for a pull request.

Targeting:
  Prefer --repo/--pr in scripts or outside a checkout.
  If omitted, bv uses the current repo and the latest open PR on the current branch.

Examples:
  bv anchors --repo owner/repo --pr 42
  bv anchors --pr 42
  bv anchors 42   # positional shorthand
  bv anchors`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := resolveTarget(cmd, anchorsTarget, args)
		if err != nil {
			return err
		}
		ref := target.Ref
		anchors, err := cache.ListAnchors(ref)
		if err != nil {
			return err
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
