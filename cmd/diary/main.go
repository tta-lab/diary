package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/neilguion/diary-cli/internal/crypto"
	"github.com/neilguion/diary-cli/internal/editor"
	"github.com/neilguion/diary-cli/internal/git"
	"github.com/neilguion/diary-cli/internal/logger"
	"github.com/neilguion/diary-cli/internal/setup"
	"github.com/neilguion/diary-cli/internal/storage"
	"github.com/neilguion/diary-cli/internal/tui"
)

const version = "0.1.0"

func main() {
	// Setup logging
	closeLog, err := logger.Setup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to setup logging: %v\n", err)
	}
	if closeLog != nil {
		defer closeLog()
	}

	// Handle global commands (no user needed)
	if len(os.Args) >= 2 {
		arg := os.Args[1]
		if arg == "version" || arg == "--version" || arg == "-v" {
			fmt.Printf("diary version %s\n", version)
			return
		}
		if arg == "help" || arg == "--help" || arg == "-h" {
			printUsage()
			return
		}
	}

	// Need at least user argument
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	user := os.Args[1]
	var command string
	var args []string

	// If only user provided: default to reading latest entry in interactive viewer
	if len(os.Args) == 2 {
		command = "read"
		args = []string{"-t"} // Interactive TUI viewer
	} else {
		command = os.Args[2]
		args = os.Args[3:]
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
	case "replace":
		handleReplace(user, args)
	case "list":
		handleList(user, args)
	case "search":
		handleSearch(user, args)
	case "import":
		handleImport(user, args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`diary - Encrypted multi-user diary management

Usage:
  diary <user> [command] [arguments]

Commands:
  (none)             Read latest entry in interactive viewer (default)
  read [date] [-t]   Show diary entry (plain text by default, -t for interactive viewer)
  append ["text"]    Append text to today's entry (reads from stdin if no text given)
  replace            Replace today's entry entirely (reads from stdin)
  edit [date]        Open entry in $EDITOR (today if no date, or specific date)
  list               List all available diary entries for user
  search ["term"]    Search across all diary entries (interactive if no term)
  import <dir>       Import markdown files (YYYY-MM-DD.md) from directory

Global:
  version            Show version information
  help               Show this help message

Examples:
  # Interactive viewer (human-friendly)
  diary neil                   # Read latest in interactive viewer

  # Agent usage (plain text output)
  diary yuki read              # Latest entry, plain text
  diary yuki read 2026-02-07   # Specific date, plain text
  diary yuki append "Completed task #107"

  # Replace (fix/compact diary)
  echo "clean content" | diary yuki replace
  cat compacted.md | diary yuki replace

  # Human usage
  diary neil read -t           # Read latest in interactive viewer
  diary neil edit              # Edit today's entry
  diary neil edit 2026-02-07   # Edit specific date

  # Listing and searching
  diary neil list              # Show all dates (greppable)
  diary neil search "task #107"

  # Import bulk entries
  diary neil import ~/Documents/old-diary/  # Import YYYY-MM-DD.md files

Multi-User:
  Each user has separate encryption key and storage:
  - Keys:    ~/.config/diary/<user>.age.key
  - Storage: ~/.diary/<user>/YYYY-MM-DD.md.age

For more information: https://github.com/neilguion/diary-cli`)
}

func handleRead(user string, args []string) {
	// Check for -t flag (TUI/markdown mode)
	useTUI := false
	var filteredArgs []string
	for _, arg := range args {
		if arg == "-t" || arg == "--tui" {
			useTUI = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	// List all entries (needed for both TUI navigation and finding latest)
	entries, err := storage.ListEntries(user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "No diary entries found for user '%s'\n", user)
		os.Exit(1)
	}

	// Sort descending (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i] > entries[j]
	})

	// Determine starting date and index
	startIndex := 0
	if len(filteredArgs) > 0 {
		date := filteredArgs[0]
		found := false
		for i, e := range entries {
			if e == date {
				startIndex = i
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "No diary entry found for %s on %s\n", user, date)
			os.Exit(1)
		}
	}

	keyPath, err := crypto.KeyPath(user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if useTUI {
		// Interactive TUI viewer with entry navigation (J/K)
		model := tui.NewReaderModel(user, keyPath, entries, startIndex)
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: TUI failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Plain mode: decrypt and output text (for agents/scripts)
		date := entries[startIndex]
		entryPath, err := storage.DiaryPath(user, date)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

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

		now := time.Now()
		fmt.Printf("Now:   %s\n", now.Format("2006-01-02 (Mon) 15:04 -07:00"))
		fmt.Printf("Entry: %s\n\n", date)
		fmt.Print(strings.TrimLeft(string(plaintext), "\n"))
	}
}

func handleAppend(user string, args []string) {
	var text string
	if len(args) >= 1 {
		text = args[0]
		// If stdin is also a pipe, warn that it's being ignored
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Fprintln(os.Stderr, "Warning: ignoring stdin — using positional argument")
		}
	} else if !term.IsTerminal(int(os.Stdin.Fd())) {
		// No positional arg: read from piped stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to read stdin: %v\n", err)
			os.Exit(1)
		}
		text = strings.TrimRight(string(data), "\n\r")
	}
	if text == "" {
		fmt.Fprintln(os.Stderr, "Error: text required for append command")
		fmt.Fprintln(os.Stderr, "Usage: diary <user> append \"text\"")
		os.Exit(1)
	}

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

		// Append with newline, avoiding double newline if content already ends with one
		existing := string(plaintext)
		if strings.HasSuffix(existing, "\n") {
			content = existing + text
		} else {
			content = existing + "\n" + text
		}
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

func handleReplace(user string, args []string) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintln(os.Stderr, "Error: replace reads from stdin only")
		fmt.Fprintln(os.Stderr, "Usage: echo 'content' | diary <user> replace")
		os.Exit(1)
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read stdin: %v\n", err)
		os.Exit(1)
	}

	content := strings.TrimRight(string(data), "\n\r")
	if content == "" {
		fmt.Fprintln(os.Stderr, "Error: replacement content is empty")
		os.Exit(1)
	}

	today := time.Now().Format("2006-01-02")

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

	identity, err := crypto.LoadIdentity(keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load key: %v\n", err)
		os.Exit(1)
	}
	recipient := identity.Recipient().String()

	encrypted, err := crypto.Encrypt([]byte(content), recipient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to encrypt: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(entryPath, encrypted, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write entry: %v\n", err)
		os.Exit(1)
	}

	if git.IsGitRepo(user) {
		if err := git.AutoCommit(user, today); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: git commit failed: %v\n", err)
		}
	}

	fmt.Fprintf(os.Stderr, "✓ Replaced diary for %s (%s)\n", user, today)
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
	// Search term is optional - if not provided, start in input mode
	term := ""
	if len(args) >= 1 {
		term = args[0]
	}

	// Get key path
	keyPath, err := crypto.KeyPath(user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Launch interactive TUI
	model := tui.NewSearchModel(user, term, keyPath)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err = p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: TUI failed: %v\n", err)
		os.Exit(1)
	}
}

func handleImport(user string, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: directory path required")
		fmt.Fprintln(os.Stderr, "Usage: diary <user> import <directory>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Imports markdown files named YYYY-MM-DD.md from the specified directory")
		os.Exit(1)
	}

	importDir := args[0]

	// Check if directory exists
	if !fileExists(importDir) {
		fmt.Fprintf(os.Stderr, "Error: directory not found: %s\n", importDir)
		os.Exit(1)
	}

	// Get key path
	keyPath, err := crypto.KeyPath(user)
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

	// Read directory
	entries, err := os.ReadDir(importDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read directory: %v\n", err)
		os.Exit(1)
	}

	// Filter for YYYY-MM-DD.md files
	var imported int
	var skipped int
	var failed int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		// Extract date from filename
		date := strings.TrimSuffix(name, ".md")

		// Validate date format
		if _, err := time.Parse("2006-01-02", date); err != nil {
			fmt.Fprintf(os.Stderr, "⊘ Skipping %s (invalid date format)\n", name)
			skipped++
			continue
		}

		// Get destination path
		destPath, err := storage.DiaryPath(user, date)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Failed %s: %v\n", name, err)
			failed++
			continue
		}

		// Check if already exists
		if fileExists(destPath) {
			fmt.Fprintf(os.Stderr, "⊘ Skipping %s (already exists)\n", name)
			skipped++
			continue
		}

		// Read source file
		sourcePath := filepath.Join(importDir, name)
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Failed %s: %v\n", name, err)
			failed++
			continue
		}

		// Encrypt
		encrypted, err := crypto.Encrypt(content, recipient)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Failed %s: encryption error: %v\n", name, err)
			failed++
			continue
		}

		// Write encrypted file
		if err := os.WriteFile(destPath, encrypted, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "✗ Failed %s: %v\n", name, err)
			failed++
			continue
		}

		fmt.Fprintf(os.Stderr, "✓ Imported %s\n", date)
		imported++
	}

	// Auto-commit if git repo exists
	if imported > 0 && git.IsGitRepo(user) {
		// Commit all imported files at once
		home, _ := os.UserHomeDir()
		diaryDir := filepath.Join(home, ".diary")

		// Add all files in user directory
		addCmd := exec.Command("git", "add", user+"/")
		addCmd.Dir = diaryDir
		if err := addCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: git add failed: %v\n", err)
		} else {
			// Commit with bulk message
			commitMsg := fmt.Sprintf("feat(diary): import %d entries", imported)
			commitCmd := exec.Command("git", "commit", "-m", commitMsg)
			commitCmd.Dir = diaryDir
			if err := commitCmd.Run(); err != nil {
				// Ignore "nothing to commit" errors
				if !strings.Contains(err.Error(), "nothing to commit") {
					fmt.Fprintf(os.Stderr, "Warning: git commit failed: %v\n", err)
				}
			}
		}
	}

	// Summary
	fmt.Fprintf(os.Stderr, "\n📊 Import Summary:\n")
	fmt.Fprintf(os.Stderr, "   ✓ Imported: %d\n", imported)
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "   ⊘ Skipped:  %d\n", skipped)
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "   ✗ Failed:   %d\n", failed)
	}
}
