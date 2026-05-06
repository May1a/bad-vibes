package issues

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/may1a/bad-vibes/internal/model"
)

// Dir returns the issues directory path: <repoRoot>/.bv/issues
func Dir(repoRoot string) string {
	return filepath.Join(repoRoot, ".bv", "issues")
}

// Init creates the issues directory if it does not exist.
func Init(repoRoot string) error {
	d := Dir(repoRoot)
	if err := os.MkdirAll(d, 0755); err != nil {
		return fmt.Errorf("creating issues dir: %w", err)
	}
	return nil
}

// NextID returns the next zero-padded 4-digit issue ID ("0001", "0002", ...).
func NextID(repoRoot string) (string, error) {
	existing, err := List(repoRoot, true)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", len(existing)+1), nil
}

// Load reads a single issue by ID.
func Load(repoRoot, id string) (model.Issue, error) {
	path := filepath.Join(Dir(repoRoot), id+".json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return model.Issue{}, fmt.Errorf("issue %q not found", id)
	}
	if err != nil {
		return model.Issue{}, err
	}
	var issue model.Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return model.Issue{}, fmt.Errorf("parsing issue %s: %w", id, err)
	}
	return issue, nil
}

// Save writes an issue atomically.
func Save(repoRoot string, issue model.Issue) error {
	if err := Init(repoRoot); err != nil {
		return err
	}
	path := filepath.Join(Dir(repoRoot), issue.ID+".json")
	data, err := json.MarshalIndent(issue, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// List returns all issues sorted by ID. Pass includeAll=true to include closed issues.
func List(repoRoot string, includeAll bool) ([]model.Issue, error) {
	d := Dir(repoRoot)
	entries, err := os.ReadDir(d)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var result []model.Issue
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(d, e.Name()))
		if err != nil {
			continue
		}
		var issue model.Issue
		if err := json.Unmarshal(data, &issue); err != nil {
			continue
		}
		if !includeAll && issue.Status == model.IssueStatusClosed {
			continue
		}
		result = append(result, issue)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}
