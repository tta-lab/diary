package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsGitRepo checks if the diary directory is a git repository
func IsGitRepo(user string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	diaryDir := filepath.Join(home, ".diary")
	gitDir := filepath.Join(diaryDir, ".git")

	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

// AutoCommit commits the diary entry with a standard message
func AutoCommit(user, date string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	diaryDir := filepath.Join(home, ".diary")
	entryPath := filepath.Join(user, fmt.Sprintf("%s.md.age", date))

	// Add the file
	if err := runGitCommand(diaryDir, "add", entryPath); err != nil {
		return fmt.Errorf("failed to add file: %w", err)
	}

	// Commit with standard message
	commitMsg := fmt.Sprintf("feat(diary): update entry for %s", date)
	if err := runGitCommand(diaryDir, "commit", "-m", commitMsg); err != nil {
		// Ignore "nothing to commit" errors
		if !strings.Contains(err.Error(), "nothing to commit") {
			return fmt.Errorf("failed to commit: %w", err)
		}
	}

	return nil
}

func runGitCommand(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}

	return nil
}
