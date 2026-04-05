package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/may1a/bv/internal/cache"
	"github.com/may1a/bv/internal/diff"
	"github.com/may1a/bv/internal/git"
	"github.com/may1a/bv/internal/github"
	"github.com/may1a/bv/internal/model"
	"github.com/spf13/cobra"
)

var (
	commentBody     string
	commentBodyFile string
	commentAnchor   string
	commentSide     string
	commentDryRun   bool
	commentTarget   targetFlags
)

var (
	fetchPRForComment        = github.FetchPRMetadata
	fetchDiffForComment      = github.FetchDiff
	postReviewComment        = github.PostReviewComment
	findUnresolvedThreadByAt = github.FindUnresolvedThreadAt
	addAnchorToCache         = cache.AddAnchor
	sleepForAnchorRetry      = time.Sleep
)

const (
	anchorLookupAttempts = 4
	anchorLookupDelay    = 250 * time.Millisecond
)

var commentCmd = &cobra.Command{
	Use:   "comment <file>:<line> [body]",
	Short: "Leave an inline review comment",
	Long: `Post an inline review comment directly from the CLI.

Required input:
  <file>:<line>
  body from the optional 2nd argument, --body, --body-file, or stdin

Examples:
  bv comment cmd/root.go:42 "Needs a guard here"
  bv comment cmd/root.go:42 --body-file ./comment.md --anchor perf
  printf 'Needs a guard here\n' | bv comment cmd/root.go:42
  bv comment cmd/root.go:42 "Old code path" --side LEFT
  bv comment cmd/root.go:42 "Needs a guard here" --dry-run`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		location, err := parseCommentLocation(cmd.CommandPath(), args[0])
		if err != nil {
			return err
		}

		bodyInput, err := readCommentBody(args[1:])
		if err != nil {
			return err
		}

		side := strings.ToUpper(strings.TrimSpace(commentSide))
		switch side {
		case "", "RIGHT":
			side = "RIGHT"
		case "LEFT":
		default:
			return fmt.Errorf("could not build review comment\n  why: --side must be LEFT or RIGHT\n  try: %s %s --side RIGHT \"comment\"", cmd.CommandPath(), formatCommentLocation(location.Path, location.Line))
		}

		target, err := resolveTarget(cmd, commentTarget)
		if err != nil {
			return err
		}
		ref := target.Ref

		pr, err := fetchPRForComment(ghClient, ctx, ref)
		if err != nil {
			return err
		}
		rawDiff, err := fetchDiffForComment(ghClient, ctx, ref)
		if err != nil {
			return err
		}
		patch, err := diff.ParseUnified(rawDiff)
		if err != nil {
			return fmt.Errorf("could not validate comment target\n  why: failed to parse the pull request diff: %v\n  try: %s diff --repo %s/%s --pr %d", err, cmd.Root().Name(), ref.Owner, ref.Repo, ref.Number)
		}

		workingDir, _ := os.Getwd()
		repoRoot, _ := git.RepoRoot()
		commentTarget, err := preflightCommentTarget(commentPreflightInput{
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

type commentLocation struct {
	Path string
	Line int
}

func parseCommentLocation(commandPath, raw string) (commentLocation, error) {
	raw = strings.TrimSpace(raw)
	idx := strings.LastIndex(raw, ":")
	if idx <= 0 || idx == len(raw)-1 {
		return commentLocation{}, fmt.Errorf("could not build review comment\n  why: expected <file>:<line>, got %q\n  try: %s path/from/diff:42 \"comment\"", raw, commandPath)
	}

	path := strings.TrimSpace(raw[:idx])
	lineText := strings.TrimSpace(raw[idx+1:])
	line, err := strconv.Atoi(lineText)
	if err != nil {
		return commentLocation{}, fmt.Errorf("could not build review comment\n  why: %q must end with a numeric line number\n  try: %s path/from/diff:42 \"comment\"", raw, commandPath)
	}
	if path == "" || line < 1 {
		return commentLocation{}, fmt.Errorf("could not build review comment\n  why: expected <file>:<line>, got %q\n  try: %s path/from/diff:42 \"comment\"", raw, commandPath)
	}
	return commentLocation{Path: path, Line: line}, nil
}

func formatCommentLocation(path string, line int) string {
	return fmt.Sprintf("%s:%d", strings.TrimSpace(path), line)
}

func readCommentBody(args []string) (commentBodyInput, error) {
	positionalBody := ""
	if len(args) > 0 {
		positionalBody = strings.TrimSpace(args[0])
	}
	if positionalBody != "" && strings.TrimSpace(commentBody) != "" {
		return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: the positional body and --body are mutually exclusive\n  try: pass the comment once, either as the 2nd argument or via --body")
	}
	if positionalBody != "" && strings.TrimSpace(commentBodyFile) != "" {
		return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: the positional body and --body-file are mutually exclusive\n  try: use either the 2nd argument or --body-file")
	}
	if strings.TrimSpace(commentBody) != "" && strings.TrimSpace(commentBodyFile) != "" {
		return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: --body and --body-file are mutually exclusive\n  try: use only one of the 2nd argument, --body, --body-file, or stdin")
	}

	if positionalBody != "" {
		return commentBodyInput{Body: positionalBody, Source: "argument"}, nil
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
			return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: stdin was provided but empty\n  try: pipe text into %s or pass a 2nd argument", "bv comment path/from/diff:42")
		}
		return commentBodyInput{Body: body, Source: "stdin"}, nil
	}

	return commentBodyInput{}, fmt.Errorf("could not build review comment\n  why: no comment body was provided\n  try: use the 2nd argument, --body, --body-file, or pipe stdin")
}

func storeAnchor(ctx context.Context, ref model.PRRef, tag, path string, line int, body string) error {
	threadNodeID, ok, err := waitForPostedThread(ctx, ref, path, line, body)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("the exact thread was not yet visible in GitHub after posting")
	}

	anchor := model.Anchor{
		Tag:      tag,
		Path:     path,
		Line:     line,
		Body:     body,
		Created:  time.Now(),
		ThreadID: threadNodeID,
	}
	if err := addAnchorToCache(ref, anchor); err != nil {
		return err
	}

	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")).Render(
		"⚓ anchor #" + tag + " saved",
	))
	return nil
}

func waitForPostedThread(ctx context.Context, ref model.PRRef, path string, line int, body string) (string, bool, error) {
	for attempt := range anchorLookupAttempts {
		threadNodeID, ok, err := findUnresolvedThreadByAt(ghClient, ctx, ref, path, line, body)
		if err != nil {
			return "", false, err
		}
		if ok {
			return threadNodeID, true, nil
		}
		if attempt == anchorLookupAttempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return "", false, ctx.Err()
		default:
		}
		sleepForAnchorRetry(anchorLookupDelay)
	}
	return "", false, nil
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
	commentCmd.Flags().StringVar(&commentBody, "body", "", "Comment body text (used when the 2nd argument is omitted)")
	commentCmd.Flags().StringVar(&commentBodyFile, "body-file", "", "Read comment body from file ('-' for stdin, used when no body argument is provided)")
	commentCmd.Flags().StringVar(&commentAnchor, "anchor", "", "Optional anchor tag to save for the new thread")
	commentCmd.Flags().StringVar(&commentSide, "side", "RIGHT", "Diff side to comment on (RIGHT or LEFT)")
	commentCmd.Flags().BoolVar(&commentDryRun, "dry-run", false, "Validate the target and print the resolved comment without posting")
}
