package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bad-vibes/internal/diff"
	"github.com/may1a/bad-vibes/internal/git"
	"github.com/may1a/bad-vibes/internal/github"
	"github.com/may1a/bad-vibes/internal/model"
	"github.com/may1a/bad-vibes/internal/session"
	"github.com/spf13/cobra"
)

var (
	reviewAddBody     string
	reviewAddBodyFile string
	reviewAddSide     string
	reviewAddDryRun   bool
	reviewAddTarget   targetFlags
)

var reviewAddCmd = &cobra.Command{
	Use:   "add <file>:<line> [body]",
	Short: "Stage an inline comment for the current review",
	Long: `Stage an inline review comment for the active review session.

Comments are stored locally and submitted all at once when you run "bv review finish".
Requires an active session started with "bv review start".

Required input:
  <file>:<line>
  body from the optional 2nd argument, --body, --body-file, or stdin

Examples:
  bv review add cmd/root.go:42 "Needs a guard here"
  bv review add cmd/root.go:42 --body-file ./comment.md
  printf 'Needs a guard here\n' | bv review add cmd/root.go:42
  bv review add cmd/root.go:42 "Old code path" --side LEFT
  bv review add cmd/root.go:42 "Needs a guard here" --dry-run`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		location, err := parseCommentLocation(cmd.CommandPath(), args[0])
		if err != nil {
			return err
		}

		bodyInput, err := readCommentBody(args[1:], reviewAddBody, reviewAddBodyFile)
		if err != nil {
			return err
		}

		side := normalizeSide(reviewAddSide)
		if side == "" {
			return fmt.Errorf("--side must be LEFT or RIGHT")
		}

		s, err := requireSession(cmd, reviewAddTarget)
		if err != nil {
			return err
		}

		rawDiff, err := github.FetchDiff(ghClient, cmd.Context(), model.PRRef{Owner: s.Owner, Repo: s.Repo, Number: s.Number})
		if err != nil {
			return err
		}
		patch, err := diff.ParseUnified(rawDiff)
		if err != nil {
			return fmt.Errorf("could not validate comment target\n  why: failed to parse PR diff: %v", err)
		}

		workingDir, _ := os.Getwd()
		repoRoot, _ := git.RepoRoot()
		preflight, err := preflightCommentTarget(commentPreflightInput{
			CommandPath: cmd.CommandPath(),
			RawPath:     location.Path,
			Line:        location.Line,
			Side:        side,
			Patch:       patch,
			WorkingDir:  workingDir,
			RepoRoot:    repoRoot,
		})
		if err != nil {
			return err
		}

		if reviewAddDryRun {
			dim := lipgloss.NewStyle().Faint(true)
			bold := lipgloss.NewStyle().Bold(true)
			fmt.Println(bold.Render("Dry run: comment would be staged"))
			fmt.Printf("  %s %s:%d (%s)\n", dim.Render("location:"), preflight.Path, preflight.Line, preflight.Side)
			fmt.Printf("  %s %s\n", dim.Render("body:"), bodyInput.Source)
			fmt.Println()
			fmt.Println(bodyInput.Body)
			return nil
		}

		s.PendingComments = append(s.PendingComments, model.PendingComment{
			Path: preflight.Path,
			Line: preflight.Line,
			Side: preflight.Side,
			Body: bodyInput.Body,
		})
		if err := session.Save(s); err != nil {
			return fmt.Errorf("saving session: %w", err)
		}

		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		dim := lipgloss.NewStyle().Faint(true)
		total := len(s.PendingComments)
		fmt.Printf("%s Comment staged (%d total). %s\n",
			green.Render("✓"),
			total,
			dim.Render("run: bv review finish --approve"),
		)
		return nil
	},
}

func normalizeSide(s string) string {
	switch s {
	case "", "RIGHT":
		return "RIGHT"
	case "LEFT":
		return "LEFT"
	}
	return ""
}

func init() {
	addTargetFlags(reviewAddCmd, &reviewAddTarget)
	reviewAddCmd.Flags().StringVar(&reviewAddBody, "body", "", "Comment body text")
	reviewAddCmd.Flags().StringVar(&reviewAddBodyFile, "body-file", "", "Read comment body from file ('-' for stdin)")
	reviewAddCmd.Flags().StringVar(&reviewAddSide, "side", "RIGHT", "Diff side to comment on (RIGHT or LEFT)")
	reviewAddCmd.Flags().BoolVar(&reviewAddDryRun, "dry-run", false, "Validate and print without staging")
}
