package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bad-vibes/internal/git"
	"github.com/may1a/bad-vibes/internal/github"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/spf13/cobra"
)

var (
	reviewPrsAllBranches bool
	reviewPrsBranch      string
	reviewPrsClosed      bool
)

var reviewPrsCmd = &cobra.Command{
	Use:   "prs",
	Short: "List pull requests",
	Long: `List pull requests for the current repo.

By default shows open PRs on the current branch.
Use --all-branches to see PRs across all branches.
Use --closed to see closed and merged PRs instead.

Examples:
  bv review prs
  bv review prs --all-branches
  bv review prs --branch feat/x
  bv review prs --closed`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		repo, err := git.RemoteRepo()
		if err != nil {
			return fmt.Errorf("could not resolve target repository\n  why: %v\n  try: bv review prs from inside a GitHub checkout", err)
		}
		owner, repoName, err := splitRepo(repo)
		if err != nil {
			return fmt.Errorf("could not detect repository from git remote; ensure you're in a git repo with a GitHub remote")
		}
		base := model.PRRef{Owner: owner, Repo: repoName}
		states := github.ListStates(reviewPrsClosed)
		if reviewPrsAllBranches && reviewPrsBranch != "" {
			return fmt.Errorf("--all-branches and --branch are mutually exclusive")
		}

		branch, err := git.CurrentBranch()
		if err != nil && !reviewPrsAllBranches && reviewPrsBranch == "" {
			return fmt.Errorf("could not resolve target branch\n  why: %v\n  try: bv review prs --all-branches", err)
		}
		if reviewPrsAllBranches {
			branch = ""
		} else if reviewPrsBranch != "" {
			branch = reviewPrsBranch
		}

		prs, err := github.FetchPRs(ghClient, ctx, base, branch, states)
		if err != nil {
			return err
		}

		dim := lipgloss.NewStyle().Faint(true)
		bold := lipgloss.NewStyle().Bold(true)
		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
		purple := lipgloss.NewStyle().Foreground(lipgloss.Color("#a78bfa"))
		yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("#facc15"))
		cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#38bdf8"))

		filterDesc := "open · " + branch
		if reviewPrsAllBranches {
			filterDesc = "open · all branches"
		} else if reviewPrsBranch != "" {
			filterDesc = "open · " + reviewPrsBranch
		}
		if reviewPrsClosed {
			filterDesc = strings.Replace(filterDesc, "open", "closed+merged", 1)
		}
		fmt.Printf("\n  %s  %s\n\n", bold.Render(base.Owner+"/"+base.Repo), dim.Render("("+filterDesc+")"))

		if len(prs) == 0 {
			fmt.Println("  " + dim.Render("no PRs found"))
			fmt.Println()
			return nil
		}

		for _, pr := range prs {
			var stateStyle lipgloss.Style
			switch pr.State {
			case "OPEN":
				stateStyle = green
			case "MERGED":
				stateStyle = purple
			default:
				stateStyle = red
			}

			num := yellow.Render(fmt.Sprintf("#%-4d", pr.Number))
			state := stateStyle.Render(fmt.Sprintf("%-7s", strings.ToLower(string(pr.State))))
			branchCol := cyan.Render(pr.HeadRefName)
			author := dim.Render("@" + pr.Author)
			title := pr.Title
			runes := []rune(title)
			if len(runes) > 55 {
				title = string(runes[:54]) + "…"
			}
			fmt.Printf("  %s  %s  %-55s  %s  %s\n", num, state, title, branchCol, author)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	reviewPrsCmd.Flags().BoolVar(&reviewPrsAllBranches, "all-branches", false, "Show PRs from all branches")
	reviewPrsCmd.Flags().StringVar(&reviewPrsBranch, "branch", "", "Show PRs for a specific branch")
	reviewPrsCmd.Flags().BoolVar(&reviewPrsClosed, "closed", false, "Show closed and merged PRs instead of open")
}
