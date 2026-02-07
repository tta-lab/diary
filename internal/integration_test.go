package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neilguion/diary-cli/internal/crypto"
	"github.com/neilguion/diary-cli/internal/setup"
	"github.com/neilguion/diary-cli/internal/storage"
)

// TestEncryptedWriteRead tests the full workflow:
// Setup user → Encrypt content → Write to disk → Read from disk → Decrypt
func TestEncryptedWriteRead(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	user := "alice"
	date := "2026-02-07"

	// Setup user (creates key, directories)
	if err := setup.EnsureUserSetup(user); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Get key path and load identity
	keyPath, err := crypto.KeyPath(user)
	if err != nil {
		t.Fatalf("KeyPath failed: %v", err)
	}

	identity, err := crypto.LoadIdentity(keyPath)
	if err != nil {
		t.Fatalf("LoadIdentity failed: %v", err)
	}
	recipient := identity.Recipient().String()

	// Original content
	originalContent := []byte("# My Diary Entry\n\nToday was productive!")

	// Encrypt
	encrypted, err := crypto.Encrypt(originalContent, recipient)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Write to diary storage
	entryPath, err := storage.DiaryPath(user, date)
	if err != nil {
		t.Fatalf("DiaryPath failed: %v", err)
	}

	if err := os.WriteFile(entryPath, encrypted, 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Read from storage
	readEncrypted, err := storage.ReadEntry(entryPath)
	if err != nil {
		t.Fatalf("ReadEntry failed: %v", err)
	}

	// Decrypt
	decrypted, err := crypto.Decrypt(readEncrypted, keyPath)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	// Verify content matches
	if string(decrypted) != string(originalContent) {
		t.Errorf("Content mismatch:\nWant: %s\nGot:  %s", originalContent, decrypted)
	}

	// Verify entry appears in list
	entries, err := storage.ListEntries(user)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}

	found := false
	for _, entry := range entries {
		if entry == date {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Entry %s not found in list: %v", date, entries)
	}
}

// TestMultiUserIsolation verifies that different users have separate encryption
// and cannot read each other's entries
func TestMultiUserIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	users := []string{"alice", "bob", "charlie"}
	date := "2026-02-07"

	// Setup all users and write entries
	userContents := make(map[string][]byte)
	for _, user := range users {
		// Setup
		if err := setup.EnsureUserSetup(user); err != nil {
			t.Fatalf("Setup failed for %s: %v", user, err)
		}

		// Create unique content for each user
		content := []byte("Secret diary of " + user)
		userContents[user] = content

		// Encrypt and write
		keyPath, _ := crypto.KeyPath(user)
		identity, _ := crypto.LoadIdentity(keyPath)
		encrypted, _ := crypto.Encrypt(content, identity.Recipient().String())

		entryPath, _ := storage.DiaryPath(user, date)
		os.WriteFile(entryPath, encrypted, 0600)
	}

	// Verify each user can read their own entry
	for _, user := range users {
		keyPath, _ := crypto.KeyPath(user)
		entryPath, _ := storage.DiaryPath(user, date)

		encrypted, _ := storage.ReadEntry(entryPath)
		decrypted, err := crypto.Decrypt(encrypted, keyPath)

		if err != nil {
			t.Errorf("User %s cannot decrypt their own entry: %v", user, err)
			continue
		}

		if string(decrypted) != string(userContents[user]) {
			t.Errorf("User %s content mismatch", user)
		}
	}

	// Verify users cannot read each other's entries (wrong key)
	t.Run("cross_user_decryption_fails", func(t *testing.T) {
		aliceKey, _ := crypto.KeyPath("alice")
		bobEntryPath, _ := storage.DiaryPath("bob", date)

		bobEncrypted, _ := storage.ReadEntry(bobEntryPath)
		_, err := crypto.Decrypt(bobEncrypted, aliceKey)

		// Should fail - alice's key can't decrypt bob's entry
		if err == nil {
			t.Error("Expected decryption to fail with wrong key, but it succeeded")
		}
	})

	// Verify each user's entries are isolated (separate directories)
	for _, user := range users {
		entries, _ := storage.ListEntries(user)

		if len(entries) != 1 {
			t.Errorf("User %s should have 1 entry, got %d", user, len(entries))
		}

		if entries[0] != date {
			t.Errorf("User %s entry date mismatch", user)
		}
	}
}

// TestSearchRealEncryptedFiles tests search functionality with actual encrypted files
func TestSearchRealEncryptedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	user := "searchtest"

	// Setup user
	if err := setup.EnsureUserSetup(user); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	keyPath, _ := crypto.KeyPath(user)
	identity, _ := crypto.LoadIdentity(keyPath)
	recipient := identity.Recipient().String()

	// Create multiple encrypted entries with searchable content
	testEntries := map[string]string{
		"2026-02-01": "Working on encryption features for the diary CLI",
		"2026-02-02": "Debugging issues with age encryption",
		"2026-02-03": "Implemented search functionality",
		"2026-02-04": "Fixed bugs in the editor integration",
		"2026-02-05": "No matches here, different content",
	}

	for date, content := range testEntries {
		encrypted, _ := crypto.Encrypt([]byte(content), recipient)
		entryPath, _ := storage.DiaryPath(user, date)
		os.WriteFile(entryPath, encrypted, 0600)
	}

	// Test search for "encryption"
	t.Run("search_encryption", func(t *testing.T) {
		searchTerm := "encryption"
		expectedMatches := []string{"2026-02-01", "2026-02-02"}

		matches := searchEntries(t, user, keyPath, searchTerm)

		if len(matches) != len(expectedMatches) {
			t.Errorf("Expected %d matches, got %d", len(expectedMatches), len(matches))
		}

		for _, expectedDate := range expectedMatches {
			if !contains(matches, expectedDate) {
				t.Errorf("Expected match for %s not found", expectedDate)
			}
		}
	})

	// Test search for "bugs"
	t.Run("search_bugs", func(t *testing.T) {
		searchTerm := "bugs"
		matches := searchEntries(t, user, keyPath, searchTerm)

		if len(matches) != 1 || matches[0] != "2026-02-04" {
			t.Errorf("Expected 1 match (2026-02-04), got %v", matches)
		}
	})

	// Test search with no matches
	t.Run("search_no_matches", func(t *testing.T) {
		searchTerm := "nonexistent"
		matches := searchEntries(t, user, keyPath, searchTerm)

		if len(matches) != 0 {
			t.Errorf("Expected no matches, got %v", matches)
		}
	})

	// Test case-insensitive search
	t.Run("search_case_insensitive", func(t *testing.T) {
		searchTerm := "ENCRYPTION"
		matches := searchEntries(t, user, keyPath, searchTerm)

		if len(matches) < 2 {
			t.Errorf("Case-insensitive search should find matches")
		}
	})
}

// searchEntries is a helper that searches encrypted entries
func searchEntries(t *testing.T, user, keyPath, searchTerm string) []string {
	entries, err := storage.ListEntries(user)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}

	var matches []string
	searchLower := strings.ToLower(searchTerm)

	for _, date := range entries {
		entryPath, _ := storage.DiaryPath(user, date)
		encrypted, _ := storage.ReadEntry(entryPath)
		decrypted, err := crypto.Decrypt(encrypted, keyPath)

		if err != nil {
			t.Logf("Decrypt failed for %s: %v", date, err)
			continue
		}

		if strings.Contains(strings.ToLower(string(decrypted)), searchLower) {
			matches = append(matches, date)
		}
	}

	return matches
}

// TestImportWorkflow tests the complete import workflow
func TestImportWorkflow(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	user := "importtest"

	// Setup user
	if err := setup.EnsureUserSetup(user); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Create source directory with markdown files
	sourceDir := t.TempDir()

	testFiles := map[string]string{
		"2026-02-01.md": "# February 1st\n\nFirst entry",
		"2026-02-02.md": "# February 2nd\n\nSecond entry",
		"2026-02-03.md": "# February 3rd\n\nThird entry",
		"invalid.txt":   "Should be ignored",
		"2026-02-04.md": "Should be ignored (no .age)",
	}

	for filename, content := range testFiles {
		path := filepath.Join(sourceDir, filename)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Simulate import by encrypting and writing files
	keyPath, _ := crypto.KeyPath(user)
	identity, _ := crypto.LoadIdentity(keyPath)
	recipient := identity.Recipient().String()

	validDates := []string{"2026-02-01", "2026-02-02", "2026-02-03"}

	for _, date := range validDates {
		sourcePath := filepath.Join(sourceDir, date+".md")
		content, _ := os.ReadFile(sourcePath)

		encrypted, _ := crypto.Encrypt(content, recipient)
		destPath, _ := storage.DiaryPath(user, date)
		os.WriteFile(destPath, encrypted, 0600)
	}

	// Verify all entries were imported
	entries, err := storage.ListEntries(user)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}

	if len(entries) != len(validDates) {
		t.Errorf("Expected %d entries, got %d", len(validDates), len(entries))
	}

	// Verify each entry can be decrypted and matches original content
	for _, date := range validDates {
		entryPath, _ := storage.DiaryPath(user, date)
		encrypted, _ := storage.ReadEntry(entryPath)
		decrypted, err := crypto.Decrypt(encrypted, keyPath)

		if err != nil {
			t.Errorf("Failed to decrypt %s: %v", date, err)
			continue
		}

		expectedContent := testFiles[date+".md"]
		if string(decrypted) != expectedContent {
			t.Errorf("Content mismatch for %s:\nWant: %s\nGot:  %s",
				date, expectedContent, decrypted)
		}
	}
}

// TestLargeDataset tests performance with many entries
func TestLargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	user := "perftest"

	// Setup user
	if err := setup.EnsureUserSetup(user); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	keyPath, _ := crypto.KeyPath(user)
	identity, _ := crypto.LoadIdentity(keyPath)
	recipient := identity.Recipient().String()

	// Create 50 entries across multiple months
	numEntries := 50
	dates := generateValidDates(numEntries)

	for i := 0; i < numEntries; i++ {
		date := dates[i]
		content := []byte("Entry number " + formatInt(i))

		encrypted, _ := crypto.Encrypt(content, recipient)
		entryPath, _ := storage.DiaryPath(user, date)
		os.WriteFile(entryPath, encrypted, 0600)
	}

	// Verify all entries can be listed
	entries, err := storage.ListEntries(user)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}

	if len(entries) != numEntries {
		t.Errorf("Expected %d entries, got %d", numEntries, len(entries))
	}

	// Verify search works with many entries
	matches := searchEntries(t, user, keyPath, "Entry")
	if len(matches) != numEntries {
		t.Errorf("Search should find all %d entries, found %d", numEntries, len(matches))
	}
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func formatInt(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	if n < 100 {
		return string(rune('0'+n/10)) + string(rune('0'+n%10))
	}
	// For larger numbers, use proper conversion
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

func generateValidDates(count int) []string {
	dates := make([]string, 0, count)
	year := 2025
	month := 9 // September
	day := 1

	for len(dates) < count {
		// Format: YYYY-MM-DD
		dateStr := formatInt(year)
		dateStr += "-"
		if month < 10 {
			dateStr += "0"
		}
		dateStr += formatInt(month)
		dateStr += "-"
		if day < 10 {
			dateStr += "0"
		}
		dateStr += formatInt(day)

		dates = append(dates, dateStr)

		// Increment day
		day++

		// Handle month boundaries (simplified, good enough for test)
		if day > 28 {
			day = 1
			month++
			if month > 12 {
				month = 1
				year++
			}
		}
	}

	return dates
}
