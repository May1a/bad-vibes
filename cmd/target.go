package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/may/bad-vibes/internal/git"
	"github.com/may/bad-vibes/internal/github"
	"github.com/may/bad-vibes/internal/model"
	"github.com/may/bad-vibes/internal/parse"
	"github.com/spf13/cobra"
)

type targetFlags struct {
	repo string
	pr   string
}

type targetResolutionInput struct {
	CommandPath    string
	RepoFlag       string
	PRFlag         string
	Args           []string
	DetectedRepo   string
	DetectedBranch string
}

type targetResolution struct {
	Ref             model.PRRef
	Branch          string
	NeedsPRAutoPick bool
	UsedAutoDetect  bool
}

func addTargetFlags(cmd *cobra.Command, flags *targetFlags) {
	cmd.Flags().StringVar(&flags.repo, "repo", "", "Repository to target (owner/repo)")
	cmd.Flags().StringVar(&flags.pr, "pr", "", "Pull request to target (number, owner/repo#N, or URL)")
}

func resolveTarget(cmd *cobra.Command, flags targetFlags, args []string) (targetResolution, error) {
	detectedRepo, detectedBranch, err := detectTargetContext(flags, args)
	if err != nil {
		rawPR, _, _ := selectedPRArg(flags.pr, args)
		if strings.TrimSpace(rawPR) == "" {
			repoHint := strings.TrimSpace(flags.repo)
			if repoHint == "" {
				repoHint = "owner/repo"
			}
			return targetResolution{}, fmt.Errorf("%s\n  why: %v\n  try: %s --repo %s --pr 42", "could not resolve target pull request", err, cmd.CommandPath(), repoHint)
		}
		return targetResolution{}, fmt.Errorf("%s\n  why: %v\n  try: %s --repo owner/repo --pr %s", "could not resolve target pull request", err, cmd.CommandPath(), rawPR)
	}

	resolved, err := resolveTargetInput(targetResolutionInput{
		CommandPath:    cmd.CommandPath(),
		RepoFlag:       flags.repo,
		PRFlag:         flags.pr,
		Args:           args,
		DetectedRepo:   detectedRepo,
		DetectedBranch: detectedBranch,
	})
	if err != nil {
		return targetResolution{}, err
	}

	if !resolved.NeedsPRAutoPick {
		return resolved, nil
	}

	pr, err := github.LatestOpenPR(ghClient, cmd.Context(), model.PRRef{
		Owner: resolved.Ref.Owner,
		Repo:  resolved.Ref.Repo,
	}, resolved.Branch)
	if err != nil {
		return targetResolution{}, err
	}

	resolved.Ref.Number = pr.Number
	resolved.UsedAutoDetect = true
	fmt.Fprintf(os.Stderr, "  -> %s/%s @ %s -> PR #%d: %s\n\n", resolved.Ref.Owner, resolved.Ref.Repo, resolved.Branch, pr.Number, pr.Title)
	return resolved, nil
}

func resolveTargetInput(input targetResolutionInput) (targetResolution, error) {
	repoFlag := strings.TrimSpace(input.RepoFlag)
	if repoFlag != "" {
		if _, _, err := splitRepo(repoFlag); err != nil {
			return targetResolution{}, fmt.Errorf("%s\n  why: %v\n  try: %s --repo owner/repo --pr 42", "could not resolve target repository", err, input.CommandPath)
		}
	}

	rawPR, hasPositional, err := selectedPRArg(input.PRFlag, input.Args)
	if err != nil {
		return targetResolution{}, fmt.Errorf("%s\n  why: %v\n  try: %s --repo owner/repo --pr 42", "could not resolve target pull request", err, input.CommandPath)
	}

	defaultRepo := repoFlag
	if defaultRepo == "" {
		defaultRepo = strings.TrimSpace(input.DetectedRepo)
	}

	if rawPR != "" {
		ref, err := parse.ParseRef(rawPR, defaultRepo)
		if err != nil {
			return targetResolution{}, fmt.Errorf("%s\n  why: %v\n  try: %s --repo owner/repo --pr 42", "could not resolve target pull request", err, input.CommandPath)
		}
		if repoFlag != "" && prIncludesRepo(rawPR) {
			owner, repo, _ := splitRepo(repoFlag)
			if ref.Owner != owner || ref.Repo != repo {
				return targetResolution{}, fmt.Errorf("%s\n  why: --repo %q conflicts with %q\n  try: %s --repo %s --pr %d", "could not resolve target pull request", repoFlag, rawPR, input.CommandPath, ref.Owner+"/"+ref.Repo, ref.Number)
			}
		}
		_ = hasPositional
		return targetResolution{Ref: ref}, nil
	}

	if defaultRepo == "" {
		return targetResolution{}, fmt.Errorf("%s\n  why: no repository was provided and Git context was unavailable\n  try: %s --repo owner/repo --pr 42", "could not resolve target pull request", input.CommandPath)
	}
	if strings.TrimSpace(input.DetectedBranch) == "" {
		return targetResolution{}, fmt.Errorf("%s\n  why: no pull request was provided and the current branch could not be detected\n  try: %s --repo %s --pr 42", "could not auto-detect the pull request", input.CommandPath, defaultRepo)
	}

	owner, repo, _ := splitRepo(defaultRepo)
	return targetResolution{
		Ref: model.PRRef{
			Owner: owner,
			Repo:  repo,
		},
		Branch:          input.DetectedBranch,
		NeedsPRAutoPick: true,
	}, nil
}

func detectTargetContext(flags targetFlags, args []string) (string, string, error) {
	rawPR, _, err := selectedPRArg(flags.pr, args)
	if err != nil {
		return "", "", err
	}

	needRepo := strings.TrimSpace(flags.repo) == "" && (rawPR == "" || isBarePRNumber(rawPR))
	needBranch := rawPR == ""

	var repo string
	if needRepo {
		repo, err = git.RemoteRepo()
		if err != nil {
			return "", "", fmt.Errorf("could not read git context: %w", err)
		}
	}

	var branch string
	if needBranch {
		branch, err = git.CurrentBranch()
		if err != nil {
			return "", "", fmt.Errorf("could not read git context: %w", err)
		}
	}

	return repo, branch, nil
}

func selectedPRArg(flag string, args []string) (string, bool, error) {
	if strings.TrimSpace(flag) != "" {
		return strings.TrimSpace(flag), false, nil
	}
	switch len(args) {
	case 0:
		return "", false, nil
	case 1:
		return strings.TrimSpace(args[0]), true, nil
	default:
		return "", false, fmt.Errorf("expected at most 1 pull request argument")
	}
}

func prIncludesRepo(raw string) bool {
	raw = strings.TrimSpace(raw)
	return strings.Contains(raw, "/") || strings.Contains(raw, "#")
}

func isBarePRNumber(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	for _, r := range raw {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func splitRepo(raw string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(raw), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected owner/repo, got %q", raw)
	}
	return parts[0], parts[1], nil
}
