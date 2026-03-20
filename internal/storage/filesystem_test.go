package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiaryPath(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		date     string
		wantErr  bool
		contains []string
	}{
		{
			name:     "valid date and user",
			user:     "alice",
			date:     "2026-02-07",
			wantErr:  false,
			contains: []string{"alice", "2026-02-07.md.age"},
		},
		{
			name:     "another valid date",
			user:     "bob",
			date:     "2025-12-25",
			wantErr:  false,
			contains: []string{"bob", "2025-12-25.md.age"},
		},
		{
			name:    "invalid date format",
			user:    "alice",
			date:    "2026-2-7",
			wantErr: true,
		},
		{
			name:    "invalid date - wrong format",
			user:    "alice",
			date:    "02-07-2026",
			wantErr: true,
		},
		{
			name:    "invalid date - not a date",
			user:    "alice",
			date:    "not-a-date",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := DiaryPath(tt.user, tt.date)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			// Check path contains expected strings
			for _, expected := range tt.contains {
				if !strings.Contains(path, expected) {
					t.Errorf("Expected path to contain %q, got %s", expected, path)
				}
			}

			// Verify .md.age extension
			if !strings.HasSuffix(path, ".md.age") {
				t.Errorf("Expected path to end with .md.age, got %s", path)
			}
		})
	}
}

func TestListEntries(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	user := "testuser"

	// Create diary directory
	diaryDir := filepath.Join(tmpDir, ".diary", user)
	if err := os.MkdirAll(diaryDir, 0700); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create some test entries
	testDates := []string{
		"2026-02-07.md.age",
		"2026-02-06.md.age",
		"2026-01-15.md.age",
		"invalid.txt",   // Should be ignored
		"2026-02-01.md", // Should be ignored (no .age)
	}

	for _, date := range testDates {
		path := filepath.Join(diaryDir, date)
		if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test ListEntries
	entries, err := ListEntries(user)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}

	// Should only return valid YYYY-MM-DD entries
	expected := []string{
		"2026-02-07",
		"2026-02-06",
		"2026-01-15",
	}

	if len(entries) != len(expected) {
		t.Errorf("Expected %d entries, got %d", len(expected), len(entries))
	}

	// Check all expected entries are present
	entriesMap := make(map[string]bool)
	for _, entry := range entries {
		entriesMap[entry] = true
	}

	for _, exp := range expected {
		if !entriesMap[exp] {
			t.Errorf("Expected entry %q not found in results", exp)
		}
	}
}

func TestListEntriesNonexistentUser(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// List entries for user with no directory
	entries, err := ListEntries("nonexistent")
	if err != nil {
		t.Fatalf("Expected no error for nonexistent user, got %v", err)
	}

	// Should return empty slice, not error
	if len(entries) != 0 {
		t.Errorf("Expected empty slice for nonexistent user, got %d entries", len(entries))
	}
}

func TestEnsureDiaryDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	user := "testuser"

	// Ensure directory
	if err := EnsureDiaryDir(user); err != nil {
		t.Fatalf("EnsureDiaryDir failed: %v", err)
	}

	// Check directory exists
	diaryDir := filepath.Join(tmpDir, ".diary", user)
	info, err := os.Stat(diaryDir)
	if err != nil {
		t.Fatalf("Directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Expected directory, got file")
	}

	// Check permissions (should be 0700)
	if info.Mode().Perm() != 0700 {
		t.Errorf("Expected permissions 0700, got %v", info.Mode().Perm())
	}

	// Calling again should not error (idempotent)
	if err := EnsureDiaryDir(user); err != nil {
		t.Errorf("Second call to EnsureDiaryDir should not error: %v", err)
	}
}

func TestReadEntry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testContent := []byte("test diary content")
	testPath := filepath.Join(tmpDir, "test.md.age")

	if err := os.WriteFile(testPath, testContent, 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test ReadEntry
	content, err := ReadEntry(testPath)
	if err != nil {
		t.Fatalf("ReadEntry failed: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Expected content %q, got %q", testContent, content)
	}
}

func TestReadEntryNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentPath := filepath.Join(tmpDir, "nonexistent.md.age")

	// Should return error for nonexistent file
	_, err := ReadEntry(nonexistentPath)
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}
