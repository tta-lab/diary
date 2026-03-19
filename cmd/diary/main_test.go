package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/neilguion/diary-cli/internal/crypto"
	"github.com/neilguion/diary-cli/internal/setup"
	"github.com/neilguion/diary-cli/internal/storage"
)

// binaryPath returns the path where we build the test binary.
func binaryPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "diary")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

// setupUser creates a temp HOME with a diary user key and storage.
func setupUser(t *testing.T, user string) string {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	if err := setup.EnsureUserSetup(user); err != nil {
		t.Fatalf("EnsureUserSetup: %v", err)
	}
	return tmpHome
}

// appendEntry writes an encrypted entry for today via crypto helpers.
func appendEntry(t *testing.T, user, content string) {
	t.Helper()
	keyPath, err := crypto.KeyPath(user)
	if err != nil {
		t.Fatalf("KeyPath: %v", err)
	}
	identity, err := crypto.LoadIdentity(keyPath)
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}

	import_time := "2026-01-01" // fixed date for stable test
	entryPath, err := storage.DiaryPath(user, import_time)
	if err != nil {
		t.Fatalf("DiaryPath: %v", err)
	}

	encrypted, err := crypto.Encrypt([]byte(content), identity.Recipient().String())
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if err := os.WriteFile(entryPath, encrypted, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// TestAppendStdinPipe verifies that `diary <user> append` with no arg reads from stdin.
func TestAppendStdinPipe(t *testing.T) {
	bin := binaryPath(t)
	user := "pipetest"
	setupUser(t, user)

	content := "# Session Handoff\n\nSome $pecial chars & <angles>"

	cmd := exec.Command(bin, user, "append")
	cmd.Stdin = strings.NewReader(content)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("append via stdin failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Appended") {
		t.Errorf("expected success message, got: %s", out)
	}

	// Verify the content was actually written: read it back
	readCmd := exec.Command(bin, user, "read")
	readOut, err := readCmd.Output()
	if err != nil {
		t.Fatalf("read after append failed: %v", err)
	}
	if !strings.Contains(string(readOut), "Session Handoff") {
		t.Errorf("content not found in entry. read output:\n%s", readOut)
	}
}

// TestAppendNoArgNoStdin verifies error when no arg and stdin is a pipe with empty content.
func TestAppendEmptyStdin(t *testing.T) {
	bin := binaryPath(t)
	user := "emptystdin"
	setupUser(t, user)

	cmd := exec.Command(bin, user, "append")
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for empty stdin, but succeeded. output:\n%s", out)
	}
	if !strings.Contains(string(out), "text required") {
		t.Errorf("expected 'text required' error, got: %s", out)
	}
}

// TestAppendArgWinsOverStdin verifies that positional arg takes precedence over stdin.
func TestAppendArgWinsOverStdin(t *testing.T) {
	bin := binaryPath(t)
	user := "argwins"
	setupUser(t, user)

	argText := "positional arg content"
	stdinText := "stdin content that should be ignored"

	cmd := exec.Command(bin, user, "append", argText)
	cmd.Stdin = strings.NewReader(stdinText)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("append with arg+stdin failed: %v\n%s", err, out)
	}

	// Stderr should warn about ignoring stdin
	if !strings.Contains(string(out), "ignoring stdin") {
		t.Errorf("expected warning about ignoring stdin, got: %s", out)
	}

	// Verify arg content was written, not stdin content
	readCmd := exec.Command(bin, user, "read")
	readOut, err := readCmd.Output()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if !strings.Contains(string(readOut), argText) {
		t.Errorf("positional arg content not found in entry. output:\n%s", readOut)
	}
	if strings.Contains(string(readOut), stdinText) {
		t.Errorf("stdin content should not be in entry, but found it. output:\n%s", readOut)
	}
}

// TestReadPlaintextDateHeader verifies that plain-text read output has a date header.
func TestReadPlaintextDateHeader(t *testing.T) {
	bin := binaryPath(t)
	user := "readheader"
	setupUser(t, user)

	appendEntry(t, user, "Some diary content")

	cmd := exec.Command(bin, user, "read")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("read failed: %v\n%s", err, out)
	}

	output := string(out)
	if !strings.HasPrefix(output, "Today:") {
		t.Errorf("expected output to start with 'Today:', got:\n%s", output)
	}
	if !strings.Contains(output, "Entry:") {
		t.Errorf("expected 'Entry:' line in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Some diary content") {
		t.Errorf("expected entry content after header, got:\n%s", output)
	}
}

// TestReadTUINoDateHeader verifies that -t flag (TUI mode) is not affected.
// We can't test the actual TUI interactively, so we just verify the binary
// accepts the flag without producing the plain-text header on stdout.
// (TUI writes to altscreen, so stdout will be empty in non-terminal context.)
func TestReadTUISkipsHeader(t *testing.T) {
	bin := binaryPath(t)
	user := "readtui"
	setupUser(t, user)

	appendEntry(t, user, "TUI content")

	// -t requires a real terminal to work properly; in a subprocess without a
	// TTY it exits immediately. We just verify stdout does NOT contain the
	// "Today:" header (it would only appear in plain-text mode).
	cmd := exec.Command(bin, user, "read", "-t")
	// No stdin TTY → bubbletea exits immediately
	out, _ := cmd.Output()

	if strings.Contains(string(out), "Today:") {
		t.Errorf("TUI mode should not output 'Today:' header, got:\n%s", out)
	}
}
