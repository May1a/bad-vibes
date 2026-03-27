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
)

var commentCmd = &cobra.Command{
	Use:   "comment [PR]",
	Short: "Leave an inline review comment",
	Long: `Post an inline review comment directly from the CLI.

Required flags:
  --file PATH
  --line N
  --body TEXT   (or pipe stdin / use --body-file)

Examples:
  bv comment --file cmd/root.go --line 42 --body "Needs a guard here"
  bv comment 42 --file cmd/root.go --line 42 --body-file ./comment.md --anchor perf
  printf 'Needs a guard here\n' | bv comment --file cmd/root.go --line 42`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if strings.TrimSpace(commentFile) == "" {
			return fmt.Errorf("--file is required")
		}
		if commentLine < 1 {
			return fmt.Errorf("--line must be >= 1")
		}

		body, err := readCommentBody()
		if err != nil {
			return err
		}

		side := strings.ToUpper(strings.TrimSpace(commentSide))
		switch side {
		case "", "RIGHT":
			side = "RIGHT"
		case "LEFT":
		default:
			return fmt.Errorf("--side must be LEFT or RIGHT")
		}

		ref, err := resolvePR(args)
		if err != nil {
			return err
		}

		pr, _, err := github.FetchPR(ghClient, ctx, ref)
		if err != nil {
			return err
		}

		// Cache HeadSHA for future use.
		prCache, _ := cache.Load(ref)
		prCache.PRID = pr.ID
		prCache.HeadSHA = pr.HeadSHA
		prCache.Owner = ref.Owner
		prCache.Repo = ref.Repo
		prCache.Number = ref.Number
		_ = cache.Save(ref, prCache)

		if _, err := github.PostReviewComment(ghClient, ctx, ref, pr.HeadSHA, commentFile, body, side, commentLine); err != nil {
			return err
		}

		green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
		fmt.Println(green.Render("✓") + " Comment posted.")

		anchorTag := strings.TrimPrefix(strings.TrimSpace(commentAnchor), "#")
		if anchorTag != "" {
			storeAnchor(ctx, ref, anchorTag, commentFile, commentLine, body)
		}

		return nil
	},
}

func readCommentBody() (string, error) {
	if strings.TrimSpace(commentBody) != "" && strings.TrimSpace(commentBodyFile) != "" {
		return "", fmt.Errorf("--body and --body-file are mutually exclusive")
	}

	if strings.TrimSpace(commentBody) != "" {
		return strings.TrimSpace(commentBody), nil
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
			return "", err
		}
		body := strings.TrimSpace(string(data))
		if body == "" {
			return "", fmt.Errorf("comment body is empty")
		}
		return body, nil
	}

	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		body := strings.TrimSpace(string(data))
		if body == "" {
			return "", fmt.Errorf("comment body is empty")
		}
		return body, nil
	}

	return "", fmt.Errorf("comment body required: use --body, --body-file, or pipe stdin")
}

func storeAnchor(ctx context.Context, ref model.PRRef, tag, path string, line int, body string) {
	threadNodeID, ok, err := github.FindUnresolvedThreadAt(ghClient, ctx, ref, path, line, body)
	if err != nil {
		fmt.Println(lipgloss.NewStyle().Faint(true).Render("(anchor not saved: " + err.Error() + ")"))
		return
	}
	if !ok {
		// GitHub may normalize comment text; fall back to path+line when body match misses.
		threadNodeID, ok, err = github.FindUnresolvedThreadAt(ghClient, ctx, ref, path, line, "")
		if err != nil {
			fmt.Println(lipgloss.NewStyle().Faint(true).Render("(anchor not saved: " + err.Error() + ")"))
			return
		}
		if !ok {
			fmt.Println(lipgloss.NewStyle().Faint(true).Render("(anchor not saved: could not resolve posted thread ID)"))
			return
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
		fmt.Println(lipgloss.NewStyle().Faint(true).Render("(anchor not saved: " + err.Error() + ")"))
		return
	}

	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")).Render(
		"⚓ anchor #" + tag + " saved",
	))
}

func init() {
	commentCmd.Flags().StringVar(&commentFile, "file", "", "File path to comment on")
	commentCmd.Flags().IntVar(&commentLine, "line", 0, "Line number to comment on")
	commentCmd.Flags().StringVar(&commentBody, "body", "", "Comment body text")
	commentCmd.Flags().StringVar(&commentBodyFile, "body-file", "", "Read comment body from file ('-' for stdin)")
	commentCmd.Flags().StringVar(&commentAnchor, "anchor", "", "Optional anchor tag to save for the new thread")
	commentCmd.Flags().StringVar(&commentSide, "side", "RIGHT", "Diff side to comment on (RIGHT or LEFT)")
}
