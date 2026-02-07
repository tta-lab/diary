package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DiaryPath returns the path to a diary entry file
func DiaryPath(user, date string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Validate date format (YYYY-MM-DD)
	if !isValidDate(date) {
		return "", fmt.Errorf("invalid date format: expected YYYY-MM-DD, got %s", date)
	}

	path := filepath.Join(home, ".diary", user, fmt.Sprintf("%s.md.age", date))
	return path, nil
}

// EnsureDiaryDir creates the diary directory if it doesn't exist
func EnsureDiaryDir(user string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	dirPath := filepath.Join(home, ".diary", user)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return fmt.Errorf("failed to create diary directory: %w", err)
	}

	return nil
}

// ListEntries returns a list of diary entry dates for the user
func ListEntries(user string) ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dirPath := filepath.Join(home, ".diary", user)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read diary directory: %w", err)
	}

	var dates []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md.age") {
			continue
		}

		// Extract date from filename (YYYY-MM-DD.md.age)
		date := strings.TrimSuffix(name, ".md.age")
		if isValidDate(date) {
			dates = append(dates, date)
		}
	}

	return dates, nil
}

// ReadEntry reads the encrypted content of a diary entry
func ReadEntry(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read entry: %w", err)
	}
	return content, nil
}

// isValidDate checks if a string is in YYYY-MM-DD format
func isValidDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}
