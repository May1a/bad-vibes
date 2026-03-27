package auth

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const tokenCacheTTL = time.Hour

// Token resolves a GitHub auth token.
// Checks GITHUB_TOKEN env var first, then a 1-hour disk cache, then `gh auth token`.
func Token() (string, error) {
	// Explicit env var wins immediately — no subprocess, no disk.
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t, nil
	}

	// Check on-disk cache (avoids slow gh subprocess on repeat invocations).
	cachePath, _ := tokenCachePath()
	if cachePath != "" {
		if t, ok := readCachedToken(cachePath); ok {
			return t, nil
		}
	}

	// Run gh auth token and cache the result.
	if _, err := exec.LookPath("gh"); err == nil {
		out, err := exec.Command("gh", "auth", "token").Output()
		if err == nil {
			if t := strings.TrimSpace(string(out)); t != "" {
				if cachePath != "" {
					_ = writeCachedToken(cachePath, t)
				}
				return t, nil
			}
		}
	}

	return "", fmt.Errorf(
		"no GitHub token found\n" +
			"  • run `gh auth login` to authenticate via the GitHub CLI, or\n" +
			"  • set the GITHUB_TOKEN environment variable",
	)
}

func tokenCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bad-vibes", "token"), nil
}

func readCachedToken(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > tokenCacheTTL {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	t := strings.TrimSpace(string(data))
	return t, t != ""
}

func writeCachedToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token+"\n"), 0600)
}
