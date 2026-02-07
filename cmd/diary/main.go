package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neilguion/diary-cli/internal/crypto"
	"github.com/neilguion/diary-cli/internal/editor"
	"github.com/neilguion/diary-cli/internal/git"
	"github.com/neilguion/diary-cli/internal/setup"
	"github.com/neilguion/diary-cli/internal/storage"
	"github.com/neilguion/diary-cli/internal/tui"
)

const version = "0.1.0"

func main() {
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

	// If only user provided: default to read with glow (quick view)
	if len(os.Args) == 2 {
		command = "read"
		args = []string{"-t"} // Force TUI/glow mode
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
  (none)             Quick view: read latest entry with glow (default)
  read [date] [-t]   Show diary entry (plain text by default, -t for glow)
  append "text"      Append text to today's entry (auto-creates if needed)
  edit [date]        Open entry in $EDITOR (today if no date, or specific date)
  list               List all available diary entries for user
  search ["term"]    Search across all diary entries (interactive if no term)
  import <dir>       Import markdown files (YYYY-MM-DD.md) from directory

Global:
  version            Show version information
  help               Show this help message

Examples:
  # Quick view (human-friendly)
  diary neil                   # Read latest with glow

  # Agent usage (plain text output)
  diary yuki read              # Latest entry, plain text
  diary yuki read 2026-02-07   # Specific date, plain text
  diary yuki append "Completed task #107"

  # Human usage
  diary neil read -t           # Force glow rendering
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
	// Check for -t flag (TUI/glow mode)
	useTUI := false
	var filteredArgs []string
	for _, arg := range args {
		if arg == "-t" || arg == "--tui" {
			useTUI = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	var date string
	if len(filteredArgs) > 0 {
		date = filteredArgs[0]
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

	// Display based on mode
	if useTUI {
		// TUI mode: use glow (no header needed, glow handles display)
		if err := displayWithGlow(plaintext); err != nil {
			// Fallback: plain text with header
			fmt.Fprintf(os.Stderr, "📖 Diary entry for %s (%s):\n\n", user, date)
			fmt.Println(string(plaintext))
		}
	} else {
		// Plain mode: just output text (for agents/scripts)
		fmt.Print(string(plaintext))
	}
}

func displayWithGlow(markdown []byte) error {
	// Check if glow is available
	if _, err := exec.LookPath("glow"); err != nil {
		return err
	}

	// For TUI mode, glow needs a file (not stdin)
	tmpFile, err := os.CreateTemp("", "diary-*.md")
	if err != nil {
		// Fallback to stdin mode
		return displayWithGlowStdin(markdown)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(markdown); err != nil {
		tmpFile.Close()
		return displayWithGlowStdin(markdown)
	}
	tmpFile.Close()

	// Open in pager mode (glow -p for interactive scrolling)
	cmd := exec.Command("glow", "-p", tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func displayWithGlowStdin(markdown []byte) error {
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

	// Loop: TUI → view entry → return to TUI
	for {
		// Launch interactive TUI
		model := tui.NewSearchModel(user, term, keyPath)
		p := tea.NewProgram(model, tea.WithAltScreen())

		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: TUI failed: %v\n", err)
			os.Exit(1)
		}

		// Check if user wants to open a specific entry
		if m, ok := finalModel.(tui.Model); ok {
			if openDate := m.GetOpenDate(); openDate != "" {
				// Open in glow
				handleRead(user, []string{openDate, "-t"})

				// After viewing, return to search with same term
				term = m.GetSearchTerm()
				continue
			}
		}

		// User quit, exit loop
		break
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
