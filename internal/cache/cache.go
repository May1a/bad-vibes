package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/may1a/bad-vibes/internal/model"
)

func cachePath(ref model.PRRef) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolving cache dir: %w", err)
	}
	dir := filepath.Join(base, "bad-vibes", ref.Owner, ref.Repo)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating cache dir: %w", err)
	}
	return filepath.Join(dir, fmt.Sprintf("%d.json", ref.Number)), nil
}

// Load reads the cache for a PR. Returns a zero-value PRCache (no error) if absent.
func Load(ref model.PRRef) (model.PRCache, error) {
	path, err := cachePath(ref)
	if err != nil {
		return model.PRCache{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return model.PRCache{Owner: ref.Owner, Repo: ref.Repo, Number: ref.Number}, nil
	}
	if err != nil {
		return model.PRCache{}, err
	}
	var c model.PRCache
	if err := json.Unmarshal(data, &c); err != nil {
		return model.PRCache{}, fmt.Errorf("parsing cache: %w", err)
	}
	return c, nil
}

// Save writes the cache atomically (temp file → rename).
func Save(ref model.PRRef, c model.PRCache) error {
	path, err := cachePath(ref)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
