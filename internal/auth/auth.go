package auth

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Token resolves a GitHub auth token.
// It tries `gh auth token` first, then falls back to the GITHUB_TOKEN env var.
func Token() (string, error) {
	// Try gh CLI
	if _, err := exec.LookPath("gh"); err == nil {
		out, err := exec.Command("gh", "auth", "token").Output()
		if err == nil {
			if t := strings.TrimSpace(string(out)); t != "" {
				return t, nil
			}
		}
	}

	// Fall back to env var
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t, nil
	}

	return "", fmt.Errorf(
		"no GitHub token found\n" +
			"  • run `gh auth login` to authenticate via the GitHub CLI, or\n" +
			"  • set the GITHUB_TOKEN environment variable",
	)
}
