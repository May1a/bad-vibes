package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/may/bad-vibes/internal/cache"
	"github.com/may/bad-vibes/internal/diff"
	"github.com/may/bad-vibes/internal/git"
	"github.com/may/bad-vibes/internal/github"
	"github.com/may/bad-vibes/internal/model"
	"github.com/spf13/cobra"
)

var (
	commentFile     string
	commentLine     int
	commentBody     string
	commentBodyFile string
	commentAnchor   string
	commentSide     string
	commentDryRun   bool
	commentTarget   targetFlags
)

var (
	fetchPRForComment        = github.FetchPR
	fetchDiffForComment      = github.FetchDiff
	postReviewComment        = github.PostReviewComment
	findUnresolvedThreadByAt = github.FindUnresolvedThreadAt
)

var commentCmd = &cobra.Command{
	Use:   "comment [PR]",
	Short: "Leave an inline review comment",
	Long: `Post an inline review comment directly from the CLI.

Required flags:
  --file PATH
  --line N
  body from --body TEXT, --body-file FILE, or stdin

Targeting:
  Prefer --repo/--pr in scripts or outside a checkout.
  If omitted, bv uses the current repo and the latest open PR on the current branch.

Examples:
  bv comment --repo owner/repo --pr 42 --file cmd/root.go --line 42 --body "Needs a guard here"
  bv comment --pr 42 --file cmd/root.go --line 42 --body-file ./comment.md --anchor perf
  printf 'Needs a guard here\n' | bv comment 42 --file cmd/root.go --line 42
  bv comment --pr 42 --file cmd/root.go --line 42 --side LEFT --body "Old code path"
  bv comment --pr 42 --file cmd/root.go --line 42 --body "Needs a guard here" --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if strings.TrimSpace(commentFile) == "" {
			return fmt.Errorf("could not build review comment\n  why: --file is required\n  try: %s --pr 42 --file path/from/diff --line N --body \"comment\"", cmd.CommandPath())
		}
		if commentLine < 1 {
			return fmt.Errorf("could not build review comment\n  why: --line must be >= 1\n  try: %s --pr 42 --file %s --line 1 --body \"comment\"", cmd.CommandPath(), strings.TrimSpace(commentFile))
		}

		bodyInput, err := readCommentBody()
		if err != nil {
			return err
		}

		side := strings.ToUpper(strings.TrimSpace(commentSide))
		switch side {
		case "", "RIGHT":
			side = "RIGHT"
		case "LEFT":
		default:
			return fmt.Errorf("could not build review comment\n  why: --side must be LEFT or RIGHT\n  try: %s --pr 42 --file %s --line %d --side RIGHT --body \"comment\"", cmd.CommandPath(), strings.TrimSpace(commentFile), commentLine)
		}

		target, err := resolveTarget(cmd, commentTarget, args)
		if err != nil {
			return err
		}
		ref := target.Ref

		pr, _, err := fetchPRForComment(ghClient, ctx, ref)
		if err != nil {
			return err
		}
		rawDiff, err := fetchDiffForComment(ghClient, ctx, ref)
		if err != nil {
			return err
		}
		patch, err := diff.ParseUnified(rawDiff)
		if err != nil {
			return fmt.Errorf("could not validate comment target\n  why: failed to parse the pull request diff: %v\n  try: %s review --repo %s/%s --pr %d", err, cmd.Root().Name(), ref.Owner, ref.Repo, ref.Number)
		}

		workingDir, _ := os.Getwd()
		repoRoot, _ := git.RepoRoot()
		commentTarget, err := preflightCommentTarget(commentPreflightInput{
			CommandPath: cmd.CommandPath(),
			RawPath:     commentFile,
			Line:        commentLine,
			Side:        side,
			Patch:       patch,
			WorkingDir:  workingDir,
			RepoRoot:    repoRoot,
		})
		if err != nil {
			return err
		}

		if commentDryRun {
			printCommentDryRun(ref, commentTarget, bodyInput)
			return nil
		}

		// Cache HeadSHA for future use.
		prCache, _ := cache.Load(ref)
		prCache.PRID = pr.ID
		prCache.HeadSHA = pr.HeadSHA
		prCache.Owner = ref.Owner
		prCache.Repo = ref.Repo
		prCache.Number = ref.Number
		_ = cache.Save(ref, prCache)

		if _, err := postReviewComment(ghClient, ctx, ref, pr.HeadSHA, commentTarget.Path, bodyInput.Body, commentTarget.Side, commentTarget.Line); err != nil {
			return err
		}

		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		fmt.Println(green.Render("✓") + " Comment posted.")

		anchorTag := strings.TrimPrefix(strings.TrimSpace(commentAnchor), "#")
		if anchorTag != "" {
			if err := storeAnchor(ctx, ref, anchorTag, commentTarget.Path, commentTarget.Line, bodyInput.Body); err != nil {
				fmt.Fprintln(os.Stderr, lipgloss.NewStyle().Faint(true).Render(
					fmt.Sprintf("warning: comment posted, but anchor #%s was not saved: %v", anchorTag, err),
				))
				fmt.Fprintf(os.Stderr, "  try: %s comments --repo %s/%s --pr %d\n", cmd.Root().Name(), ref.Owner, ref.Repo, ref.Number)
			}
		}

		return nil
	},
}

func readCommentBody() (commentBodyInput, error) {
	if strings.TrimSpace(commentBody) != "" && strings.TrimSpace(commentBodyFile) != "" {
		return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: --body and --body-file are mutually exclusive\n  try: use only one of --body, --body-file, or stdin")
	}

	if strings.TrimSpace(commentBody) != "" {
		return commentBodyInput{Body: strings.TrimSpace(commentBody), Source: "--body"}, nil
	}

	if strings.TrimSpace(commentBodyFile) != "" {
		var (
			data []byte
			err  error
		)
		if commentBodyFile == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(commentBodyFile)
		}
		if err != nil {
			return commentBodyInput{}, err
		}
		body := strings.TrimSpace(string(data))
		if body == "" {
			return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: %s did not contain any comment text\n  try: add text to %s or pass --body", commentBodyFile, commentBodyFile)
		}
		return commentBodyInput{Body: body, Source: "--body-file=" + commentBodyFile}, nil
	}

	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return commentBodyInput{}, err
		}
		body := strings.TrimSpace(string(data))
		if body == "" {
			return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: stdin was provided but empty\n  try: pipe text into %s or pass --body", "bv comment")
		}
		return commentBodyInput{Body: body, Source: "stdin"}, nil
	}

	return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: no comment body was provided\n  try: use --body, --body-file, or pipe stdin")
}

func storeAnchor(ctx context.Context, ref model.PRRef, tag, path string, line int, body string) error {
	threadNodeID, ok, err := findUnresolvedThreadByAt(ghClient, ctx, ref, path, line, body)
	if err != nil {
		return err
	}
	if !ok {
		// GitHub may normalize comment text; fall back to path+line when body match misses.
		threadNodeID, ok, err = findUnresolvedThreadByAt(ghClient, ctx, ref, path, line, "")
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("the thread was not yet visible in GitHub after posting")
		}
	}

	anchor := model.Anchor{
		Tag:      tag,
		Path:     path,
		Line:     line,
		Body:     body,
		Created:  time.Now(),
		ThreadID: threadNodeID,
	}
	if err := cache.AddAnchor(ref, anchor); err != nil {
		return err
	}

	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")).Render(
		"⚓ anchor #" + tag + " saved",
	))
	return nil
}

func printCommentDryRun(ref model.PRRef, target commentPreflightResult, body commentBodyInput) {
	dim := lipgloss.NewStyle().Faint(true)
	bold := lipgloss.NewStyle().Bold(true)
	fmt.Println(bold.Render("Dry run: comment would be posted"))
	fmt.Printf("  %s %s/%s PR #%d\n", dim.Render("target:"), ref.Owner, ref.Repo, ref.Number)
	fmt.Printf("  %s %s\n", dim.Render("file:"), target.Path)
	fmt.Printf("  %s %d\n", dim.Render("line:"), target.Line)
	fmt.Printf("  %s %s\n", dim.Render("side:"), target.Side)
	fmt.Printf("  %s %s\n", dim.Render("body:"), body.Source)
	fmt.Println()
	fmt.Println(body.Body)
}

func init() {
	addTargetFlags(commentCmd, &commentTarget)
	commentCmd.Flags().StringVar(&commentFile, "file", "", "File path to comment on")
	commentCmd.Flags().IntVar(&commentLine, "line", 0, "Line number to comment on")
	commentCmd.Flags().StringVar(&commentBody, "body", "", "Comment body text (highest precedence)")
	commentCmd.Flags().StringVar(&commentBodyFile, "body-file", "", "Read comment body from file ('-' for stdin, used when --body is absent)")
	commentCmd.Flags().StringVar(&commentAnchor, "anchor", "", "Optional anchor tag to save for the new thread")
	commentCmd.Flags().StringVar(&commentSide, "side", "RIGHT", "Diff side to comment on (RIGHT or LEFT)")
	commentCmd.Flags().BoolVar(&commentDryRun, "dry-run", false, "Validate the target and print the resolved comment without posting")
}
