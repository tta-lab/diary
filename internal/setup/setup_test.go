package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureUserSetup(t *testing.T) {
	// Create temp directories for test
	tmpHome := t.TempDir()

	// Override home directory for test
	t.Setenv("HOME", tmpHome)

	// Test user
	user := "testuser"

	// Run setup
	if err := EnsureUserSetup(user); err != nil {
		t.Fatalf("EnsureUserSetup failed: %v", err)
	}

	// Verify key exists
	keyPath := filepath.Join(tmpHome, ".config", "diary", user+".age.key")
	if !fileExists(keyPath) {
		t.Errorf("age key not created at %s", keyPath)
	}

	// Verify key permissions
	keyInfo, _ := os.Stat(keyPath)
	if keyInfo.Mode().Perm() != 0600 {
		t.Errorf("key permissions are %o, expected 0600", keyInfo.Mode().Perm())
	}

	// Verify diary root exists
	diaryRoot := filepath.Join(tmpHome, ".diary")
	if !dirExists(diaryRoot) {
		t.Errorf("diary root not created at %s", diaryRoot)
	}

	// Verify git repo initialized
	gitDir := filepath.Join(diaryRoot, ".git")
	if !dirExists(gitDir) {
		t.Errorf("git repo not initialized at %s", gitDir)
	}

	// Verify .gitignore exists
	gitignore := filepath.Join(diaryRoot, ".gitignore")
	if !fileExists(gitignore) {
		t.Errorf(".gitignore not created at %s", gitignore)
	}

	// Verify user directory exists
	userDir := filepath.Join(diaryRoot, user)
	if !dirExists(userDir) {
		t.Errorf("user directory not created at %s", userDir)
	}

	// Run setup again (should be idempotent)
	if err := EnsureUserSetup(user); err != nil {
		t.Fatalf("EnsureUserSetup not idempotent: %v", err)
	}
}

func TestMultipleUsers(t *testing.T) {
	// Create temp directories for test
	tmpHome := t.TempDir()

	// Override home directory for test
	t.Setenv("HOME", tmpHome)

	// Setup multiple users
	users := []string{"alice", "bob", "charlie"}

	for _, user := range users {
		if err := EnsureUserSetup(user); err != nil {
			t.Fatalf("EnsureUserSetup failed for %s: %v", user, err)
		}
	}

	// Verify each user has their own key and directory
	for _, user := range users {
		keyPath := filepath.Join(tmpHome, ".config", "diary", user+".age.key")
		if !fileExists(keyPath) {
			t.Errorf("key not found for user %s", user)
		}

		userDir := filepath.Join(tmpHome, ".diary", user)
		if !dirExists(userDir) {
			t.Errorf("directory not found for user %s", user)
		}
	}

	// Verify only one git repo (shared)
	gitDir := filepath.Join(tmpHome, ".diary", ".git")
	if !dirExists(gitDir) {
		t.Error("shared git repo not found")
	}
}
