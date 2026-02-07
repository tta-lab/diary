package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/neilguion/diary-cli/internal/crypto"
	"github.com/neilguion/diary-cli/internal/editor"
	"github.com/neilguion/diary-cli/internal/git"
	"github.com/neilguion/diary-cli/internal/setup"
	"github.com/neilguion/diary-cli/internal/storage"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	user := os.Args[1]
	command := os.Args[2]
	args := os.Args[3:]

	// Handle global commands (no user needed)
	if user == "version" || user == "--version" || user == "-v" {
		fmt.Printf("diary version %s\n", version)
		return
	}
	if user == "help" || user == "--help" || user == "-h" {
		printUsage()
		return
	}

	// Ensure user setup (auto-creates key, directories, git repo on first use)
	if err := setup.EnsureUserSetup(user); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to setup user '%s': %v\n", user, err)
		os.Exit(1)
	}

	switch command {
	case "read":
		handleRead(user, args)
	case "append":
		handleAppend(user, args)
	case "edit":
		handleEdit(user, args)
	case "list":
		handleList(user, args)
	case "search":
		handleSearch(user, args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`diary - Encrypted multi-user diary management

Usage:
  diary <user> <command> [arguments]

Commands:
  read [date]        Show diary entry (latest if no date, or specific date)
  append "text"      Append text to today's entry (auto-creates if needed)
  edit [date]        Open entry in $EDITOR (today if no date, or specific date)
  list               List all available diary entries for user
  search "term"      Search across all diary entries for user

Global:
  version            Show version information
  help               Show this help message

Examples:
  # Agent usage (programmatic append)
  diary yuki append "Completed task #107"
  diary neil append "Meeting notes: Q2 planning"

  # Human usage (interactive editing)
  diary neil edit              # Edit today's entry
  diary neil edit 2026-02-07   # Edit specific date

  # Reading
  diary neil read              # Show latest entry
  diary neil read 2026-02-07   # Show specific date
  diary neil list              # Show all available dates

  # Searching
  diary neil search "task #107"

Multi-User:
  Each user has separate encryption key and storage:
  - Keys:    ~/.config/diary/<user>.age.key
  - Storage: ~/.diary/<user>/YYYY-MM-DD.md.age

For more information: https://github.com/neilguion/diary-cli`)
}

func handleRead(user string, args []string) {
	var date string
	if len(args) > 0 {
		date = args[0]
	} else {
		// Find latest entry
		entries, err := storage.ListEntries(user)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(entries) == 0 {
			fmt.Fprintf(os.Stderr, "No diary entries found for user '%s'\n", user)
			os.Exit(1)
		}

		// Sort descending and take first (latest)
		sort.Slice(entries, func(i, j int) bool {
			return entries[i] > entries[j]
		})
		date = entries[0]
	}

	// Get paths
	keyPath, err := crypto.KeyPath(user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	entryPath, err := storage.DiaryPath(user, date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check if entry exists
	if !fileExists(entryPath) {
		fmt.Fprintf(os.Stderr, "No diary entry found for %s on %s\n", user, date)
		os.Exit(1)
	}

	// Read and decrypt
	encrypted, err := os.ReadFile(entryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read entry: %v\n", err)
		os.Exit(1)
	}

	plaintext, err := crypto.Decrypt(encrypted, keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to decrypt entry: %v\n", err)
		os.Exit(1)
	}

	// Display with glow (fallback to plain if not available)
	fmt.Fprintf(os.Stderr, "📖 Diary entry for %s (%s):\n\n", user, date)
	if err := displayWithGlow(plaintext); err != nil {
		// Fallback: plain text
		fmt.Println(string(plaintext))
	}
}

func displayWithGlow(markdown []byte) error {
	// Check if glow is available
	if _, err := exec.LookPath("glow"); err != nil {
		return err
	}

	cmd := exec.Command("glow", "-")
	cmd.Stdin = bytes.NewReader(markdown)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func handleAppend(user string, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: text required for append command")
		fmt.Fprintln(os.Stderr, "Usage: diary <user> append \"text\"")
		os.Exit(1)
	}
	text := args[0]

	// Get today's date
	today := time.Now().Format("2006-01-02")

	// Get paths
	keyPath, err := crypto.KeyPath(user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	entryPath, err := storage.DiaryPath(user, today)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Load identity for encryption
	identity, err := crypto.LoadIdentity(keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load key: %v\n", err)
		os.Exit(1)
	}
	recipient := identity.Recipient().String()

	// Check if entry exists
	var content string
	if fileExists(entryPath) {
		// Decrypt existing entry
		encrypted, err := os.ReadFile(entryPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to read entry: %v\n", err)
			os.Exit(1)
		}

		plaintext, err := crypto.Decrypt(encrypted, keyPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to decrypt entry: %v\n", err)
			os.Exit(1)
		}

		// Append with newline
		content = string(plaintext) + "\n" + text
	} else {
		// New entry
		content = text
	}

	// Encrypt
	encrypted, err := crypto.Encrypt([]byte(content), recipient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to encrypt: %v\n", err)
		os.Exit(1)
	}

	// Save
	if err := os.WriteFile(entryPath, encrypted, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write entry: %v\n", err)
		os.Exit(1)
	}

	// Auto-commit if git repo exists
	if git.IsGitRepo(user) {
		if err := git.AutoCommit(user, today); err != nil {
			// Don't fail on git errors, just warn
			fmt.Fprintf(os.Stderr, "Warning: git commit failed: %v\n", err)
		}
	}

	fmt.Fprintf(os.Stderr, "✓ Appended to diary for %s (%s)\n", user, today)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func handleEdit(user string, args []string) {
	var date string
	if len(args) > 0 {
		date = args[0]
	} else {
		// Default to today
		date = time.Now().Format("2006-01-02")
	}

	// Get paths
	keyPath, err := crypto.KeyPath(user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	entryPath, err := storage.DiaryPath(user, date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Load identity for encryption
	identity, err := crypto.LoadIdentity(keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load key: %v\n", err)
		os.Exit(1)
	}
	recipient := identity.Recipient().String()

	// Get existing content (if exists)
	var initialContent []byte
	if fileExists(entryPath) {
		encrypted, err := os.ReadFile(entryPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to read entry: %v\n", err)
			os.Exit(1)
		}

		plaintext, err := crypto.Decrypt(encrypted, keyPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to decrypt entry: %v\n", err)
			os.Exit(1)
		}
		initialContent = plaintext
	}

	// Open in editor
	editedContent, err := editor.Open(initialContent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: editor failed: %v\n", err)
		os.Exit(1)
	}

	// Check if content changed
	if bytes.Equal(editedContent, initialContent) {
		fmt.Fprintf(os.Stderr, "No changes made\n")
		return
	}

	// Encrypt and save
	encrypted, err := crypto.Encrypt(editedContent, recipient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to encrypt: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(entryPath, encrypted, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write entry: %v\n", err)
		os.Exit(1)
	}

	// Auto-commit
	if git.IsGitRepo(user) {
		if err := git.AutoCommit(user, date); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: git commit failed: %v\n", err)
		}
	}

	fmt.Fprintf(os.Stderr, "✓ Saved diary entry for %s (%s)\n", user, date)
}

func handleList(user string, args []string) {
	entries, err := storage.ListEntries(user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		// No entries - silent exit for greppability
		os.Exit(0)
	}

	// Sort descending (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i] > entries[j]
	})

	// Clean output: one date per line, no fluff
	for _, date := range entries {
		fmt.Println(date)
	}
}

func handleSearch(user string, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: search term required")
		fmt.Fprintln(os.Stderr, "Usage: diary <user> search \"term\"")
		os.Exit(1)
	}
	term := args[0]
	// TODO: Implement search
	fmt.Printf("TODO: Search for '%s' in %s's entries\n", term, user)
}
