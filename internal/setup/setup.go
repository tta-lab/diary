package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/tta-lab/diary/internal/crypto"
)

// EnsureUserSetup ensures all necessary directories and keys exist for a user
// Creates on first use:
// 1. ~/.config/diary/{user}.age.key (if not exists)
// 2. ~/.diary/ (if not exists)
// 3. ~/.diary/.git (if not exists) - init git repo
// 4. ~/.diary/{user}/ (if not exists)
func EnsureUserSetup(user string) error {
	// 1. Ensure key exists
	keyPath, err := crypto.KeyPath(user)
	if err != nil {
		return fmt.Errorf("failed to get key path: %w", err)
	}

	if !fileExists(keyPath) {
		if err := generateUserKey(user, keyPath); err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}
	}

	// 2. Ensure ~/.diary/ exists
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	diaryRoot := filepath.Join(home, ".diary")
	if err := ensureDir(diaryRoot, 0700); err != nil {
		return fmt.Errorf("failed to create diary root: %w", err)
	}

	// 3. Ensure git repo initialized
	gitDir := filepath.Join(diaryRoot, ".git")
	if !dirExists(gitDir) {
		if err := initGitRepo(diaryRoot); err != nil {
			return fmt.Errorf("failed to init git repo: %w", err)
		}
	}

	// 4. Ensure user directory exists
	userDir := filepath.Join(diaryRoot, user)
	if err := ensureDir(userDir, 0700); err != nil {
		return fmt.Errorf("failed to create user directory: %w", err)
	}

	return nil
}

// generateUserKey generates a new age key for the user
func generateUserKey(user, keyPath string) error {
	// Ensure config directory exists
	configDir := filepath.Dir(keyPath)
	if err := ensureDir(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create key file
	keyFile, err := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()

	// Generate key
	if err := crypto.GenerateKey(keyFile, user); err != nil {
		return fmt.Errorf("failed to generate age key: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Generated age key for user '%s' at %s\n", user, keyPath)
	fmt.Fprintf(os.Stderr, "  IMPORTANT: Back up this key! Without it, encrypted diaries cannot be recovered.\n")

	return nil
}

// initGitRepo initializes a git repository in the diary root
func initGitRepo(diaryRoot string) error {
	// Init git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = diaryRoot
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

	// Set local git config (in case global not set)
	configCmds := [][]string{
		{"git", "config", "user.email", "diary@localhost"},
		{"git", "config", "user.name", "Diary CLI"},
	}
	for _, args := range configCmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = diaryRoot
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git config failed: %w", err)
		}
	}

	// Create .gitignore
	gitignorePath := filepath.Join(diaryRoot, ".gitignore")
	gitignoreContent := `# Never commit age keys
*.age.key
*.key

# Unencrypted files (should never exist, but just in case)
*.md

# OS files
.DS_Store
Thumbs.db
`
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Add and commit .gitignore
	addCmd := exec.Command("git", "add", ".gitignore")
	addCmd.Dir = diaryRoot
	addCmd.Stderr = os.Stderr
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "chore(diary): initialize encrypted diary repository")
	commitCmd.Dir = diaryRoot
	commitCmd.Stderr = os.Stderr
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Initialized git repository at %s\n", diaryRoot)

	return nil
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(path string, perm os.FileMode) error {
	if dirExists(path) {
		return nil
	}

	if err := os.MkdirAll(path, perm); err != nil {
		return err
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
