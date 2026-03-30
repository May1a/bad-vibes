package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var reRemote = regexp.MustCompile(`github\.com[/:]([^/]+)/([^/]+?)(?:\.git)?$`)

// RemoteRepo returns "owner/repo" parsed from the origin remote URL.
func RemoteRepo() (string, error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", fmt.Errorf("could not read git remote: not inside a git repo, or no 'origin' remote set")
	}
	url := strings.TrimSpace(string(out))
	m := reRemote.FindStringSubmatch(url)
	if m == nil {
		return "", fmt.Errorf("origin remote %q is not a GitHub URL", url)
	}
	return m[1] + "/" + m[2], nil
}

// CurrentBranch returns the name of the currently checked-out branch.
func CurrentBranch() (string, error) {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return "", fmt.Errorf("could not determine current branch")
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", fmt.Errorf("not on a branch (detached HEAD?)")
	}
	return branch, nil
}

// RepoRoot returns the absolute path to the current git repository root.
func RepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("could not determine git repository root")
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", fmt.Errorf("could not determine git repository root")
	}
	return filepath.Clean(root), nil
}
