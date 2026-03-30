package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/may/bad-vibes/internal/git"
	"github.com/may/bad-vibes/internal/github"
	"github.com/may/bad-vibes/internal/model"
	"github.com/spf13/cobra"
)

var (
	prsAllBranches bool
	prsBranch      string
	prsClosed      bool
)

var prsCmd = &cobra.Command{
	Use:   "prs",
	Short: "List pull requests",
	Long: `List pull requests for the current repo.

By default shows open PRs on the current branch.
Use --all-branches to see PRs across all branches.
Use --closed to see closed and merged PRs instead.

Examples:
  bv prs                    # open PRs on current branch
  bv prs --all-branches     # open PRs across all branches
  bv prs --branch feat/x    # open PRs on a specific branch
  bv prs --closed           # closed and merged PRs`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		repo, err := git.RemoteRepo()
		if err != nil {
			return fmt.Errorf("could not resolve target repository\n  why: %v\n  try: bv prs from inside a GitHub checkout", err)
		}
		parts := strings.SplitN(repo, "/", 2)
		base := model.PRRef{Owner: parts[0], Repo: parts[1]}
		if base.Owner == "" || base.Repo == "" {
			return fmt.Errorf("could not detect repository from git remote; ensure you're in a git repo with a GitHub remote")
		}
		states := github.StatesFromFlags(prsClosed)
		if prsAllBranches && prsBranch != "" {
			return fmt.Errorf("--all-branches and --branch are mutually exclusive")
		}

		branch, err := git.CurrentBranch()
		if err != nil && !prsAllBranches && prsBranch == "" {
			return fmt.Errorf("could not resolve target branch\n  why: %v\n  try: bv prs --all-branches", err)
		}
		if prsAllBranches {
			branch = ""
		} else if prsBranch != "" {
			branch = prsBranch
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

		// Header
		filterDesc := "open · " + branch
		if prsAllBranches {
			filterDesc = "open · all branches"
		} else if prsBranch != "" {
			filterDesc = "open · " + prsBranch
		}
		if prsClosed {
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
			state := stateStyle.Render(fmt.Sprintf("%-7s", github.FormatState(pr.State)))
			branchCol := cyan.Render(pr.HeadRefName)
			author := dim.Render("@" + pr.Author)
			title := pr.Title
			runes := []rune(title)
			if len(runes) > 55 {
				title = string(runes[:54]) + "…"
			}

			fmt.Printf("  %s  %s  %-55s  %s  %s\n",
				num, state, title, branchCol, author,
			)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	prsCmd.Flags().BoolVar(&prsAllBranches, "all-branches", false, "Show PRs from all branches")
	prsCmd.Flags().StringVar(&prsBranch, "branch", "", "Show PRs for a specific branch")
	prsCmd.Flags().BoolVar(&prsClosed, "closed", false, "Show closed and merged PRs instead of open")
}
