package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestToken_FromEnv(t *testing.T) {
	original := os.Getenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", original)

	os.Setenv("GITHUB_TOKEN", "test-token-from-env")

	token, err := Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test-token-from-env" {
		t.Fatalf("expected token from env, got %q", token)
	}
}

func TestReadCachedToken_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")

	err := os.WriteFile(path, []byte("cached-token\n"), 0600)
	if err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}

	token, ok := readCachedToken(path)
	if !ok {
		t.Fatal("expected token to be read")
	}
	if token != "cached-token" {
		t.Fatalf("expected cached-token, got %q", token)
	}
}

func TestReadCachedToken_NonExistent(t *testing.T) {
	token, ok := readCachedToken("/nonexistent/path/token")
	if ok {
		t.Fatal("expected ok=false for non-existent file")
	}
	if token != "" {
		t.Fatalf("expected empty token, got %q", token)
	}
}

func TestReadCachedToken_Expired(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")

	err := os.WriteFile(path, []byte("expired-token\n"), 0600)
	if err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}

	// Set modification time to 2 hours ago
	expired := time.Now().Add(-2 * time.Hour)
	os.Chtimes(path, expired, expired)

	_, ok := readCachedToken(path)
	if ok {
		t.Fatal("expected ok=false for expired cache")
	}
}

func TestWriteCachedToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")

	err := writeCachedToken(path, "new-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read cache: %v", err)
	}

	if string(data) != "new-token\n" {
		t.Fatalf("expected new-token, got %q", string(data))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat cache: %v", err)
	}

	// Check permissions (should be 0600)
	if info.Mode().Perm()&0777 != 0600 {
		t.Errorf("expected mode 0600, got %o", info.Mode().Perm()&0777)
	}
}

func TestTokenCachePath_UserCacheDir(t *testing.T) {
	// This test just verifies the function doesn't panic
	path, err := tokenCachePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty cache path")
	}
}
