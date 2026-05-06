package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/may1a/bad-vibes/internal/model"
)

func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home dir: %w", err)
	}
	return filepath.Join(home, ".bv"), nil
}

func sessionPath(ref model.PRRef) (string, error) {
	base, err := dir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(base, ref.Owner, ref.Repo)
	if err := os.MkdirAll(d, 0700); err != nil {
		return "", fmt.Errorf("creating session dir: %w", err)
	}
	return filepath.Join(d, fmt.Sprintf("%d.json", ref.Number)), nil
}

// Load reads the session for a PR. Returns (zero, false, nil) if no session exists.
func Load(ref model.PRRef) (model.ReviewSession, bool, error) {
	path, err := sessionPath(ref)
	if err != nil {
		return model.ReviewSession{}, false, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return model.ReviewSession{}, false, nil
	}
	if err != nil {
		return model.ReviewSession{}, false, err
	}
	var s model.ReviewSession
	if err := json.Unmarshal(data, &s); err != nil {
		return model.ReviewSession{}, false, fmt.Errorf("parsing session: %w", err)
	}
	return s, true, nil
}

// Save writes the session atomically (temp file → rename).
func Save(s model.ReviewSession) error {
	ref := model.PRRef{Owner: s.Owner, Repo: s.Repo, Number: s.Number}
	path, err := sessionPath(ref)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Delete removes the session file for a PR.
func Delete(ref model.PRRef) error {
	path, err := sessionPath(ref)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Active looks for a single active session in ~/.bv/<owner>/<repo>/.
// Returns (zero, false, nil) when none exist.
// Returns an error when multiple sessions exist (caller should use --pr to disambiguate).
func Active(owner, repo string) (model.ReviewSession, bool, error) {
	base, err := dir()
	if err != nil {
		return model.ReviewSession{}, false, err
	}
	d := filepath.Join(base, owner, repo)
	entries, err := os.ReadDir(d)
	if os.IsNotExist(err) {
		return model.ReviewSession{}, false, nil
	}
	if err != nil {
		return model.ReviewSession{}, false, err
	}

	var sessions []model.ReviewSession
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(d, e.Name()))
		if err != nil {
			continue
		}
		var s model.ReviewSession
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}

	if len(sessions) == 0 {
		return model.ReviewSession{}, false, nil
	}
	if len(sessions) > 1 {
		var nums []string
		for _, s := range sessions {
			nums = append(nums, fmt.Sprintf("PR #%d", s.Number))
		}
		return model.ReviewSession{}, false, fmt.Errorf(
			"multiple review sessions active for %s/%s: %s — use --pr to specify",
			owner, repo, strings.Join(nums, ", "),
		)
	}
	return sessions[0], true, nil
}
