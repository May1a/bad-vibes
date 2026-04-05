package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/may1a/bv/internal/model"
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

// AddAnchor appends an anchor to the PR cache, replacing any existing anchor with the same tag.
func AddAnchor(ref model.PRRef, a model.Anchor) error {
	c, err := Load(ref)
	if err != nil {
		return err
	}
	// Replace if tag already exists
	for i, existing := range c.Anchors {
		if existing.Tag == a.Tag {
			c.Anchors[i] = a
			return Save(ref, c)
		}
	}
	c.Anchors = append(c.Anchors, a)
	return Save(ref, c)
}

// ListAnchors returns all anchors for a PR.
func ListAnchors(ref model.PRRef) ([]model.Anchor, error) {
	c, err := Load(ref)
	if err != nil {
		return nil, err
	}
	return c.Anchors, nil
}
