package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/may/bad-vibes/internal/github"
	"github.com/may/bad-vibes/internal/model"
	"github.com/spf13/cobra"
)

var summaryTarget targetFlags

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show a tidy PR overview",
	Long: `Show a tidy PR overview including title, author, state, diff stats, unresolved thread count, and per-file changes.

Targeting:
  Prefer --repo/--pr in scripts or outside a checkout.
  If omitted, bv uses the current repo and the latest open PR on the current branch.

Examples:
  bv summary --repo owner/repo --pr 42
  bv summary --pr 42
  bv summary       # auto-detect from current branch`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		target, err := resolveTarget(cmd, summaryTarget)
		if err != nil {
			return err
		}
		ref := target.Ref

		pr, files, err := github.FetchPR(ghClient, ctx, ref)
		if err != nil {
			return err
		}
		threads, err := github.FetchReviewThreads(ghClient, ctx, ref)
		if err != nil {
			return err
		}

		unresolvedCount := 0
		for _, t := range threads {
			if !t.IsResolved {
				unresolvedCount++
			}
		}

		bold := lipgloss.NewStyle().Bold(true)
		dim := lipgloss.NewStyle().Faint(true)
		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
		yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("#facc15"))

		fmt.Printf("\n%s  %s\n", bold.Render(fmt.Sprintf("PR #%d", pr.Number)), pr.Title)
		fmt.Printf("%s  %s\n\n", dim.Render("by"), pr.Author)

		stateColor := green
		if pr.State != "OPEN" {
			stateColor = dim
		}
		fmt.Printf("  %s  %s  %s  %s\n",
			stateColor.Render(pr.State),
			green.Render(fmt.Sprintf("+%d", pr.Additions)),
			red.Render(fmt.Sprintf("-%d", pr.Deletions)),
			dim.Render(fmt.Sprintf("%d files changed", pr.ChangedFiles)),
		)

		if unresolvedCount > 0 {
			fmt.Printf("  %s unresolved thread(s)\n", yellow.Bold(true).Render(fmt.Sprintf("%d", unresolvedCount)))
		} else {
			fmt.Printf("  %s\n", green.Render("no unresolved threads"))
		}

		fmt.Println()
		if pr.Body != "" {
			fmt.Println(dim.Render("Description:"))
			for _, line := range strings.Split(pr.Body, "\n") {
				fmt.Println("  " + line)
			}
			fmt.Println()
		}

		if len(files) > 0 {
			fmt.Println(dim.Render("Changed files:"))
			for _, f := range files {
				fmt.Printf("  %s %s %s\n", dim.Render("·"), formatSummaryFileStatus(f), formatSummaryFileDelta(f))
			}
			fmt.Println()
		}

		fmt.Println(dim.Render(pr.URL))
		return nil
	},
}

func init() {
	addTargetFlags(summaryCmd, &summaryTarget)
}

func formatSummaryFileStatus(file model.PRFile) string {
	label := "mod"
	switch strings.ToLower(file.Status) {
	case "added":
		label = "new"
	case "removed":
		label = "del"
	case "renamed":
		label = "ren"
	}
	if strings.EqualFold(file.Status, "renamed") && file.PreviousPath != "" {
		return fmt.Sprintf("[%s] %s -> %s", label, file.PreviousPath, file.Path)
	}
	return fmt.Sprintf("[%s] %s", label, file.Path)
}

func formatSummaryFileDelta(file model.PRFile) string {
	return fmt.Sprintf("(+%d/-%d)", file.Additions, file.Deletions)
}
