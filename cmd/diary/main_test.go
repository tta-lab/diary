package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/neilguion/diary-cli/internal/crypto"
	"github.com/neilguion/diary-cli/internal/setup"
	"github.com/neilguion/diary-cli/internal/storage"
)

// testBinary holds the path to the built binary, shared across all tests.
var (
	testBinary     string
	testBinaryOnce sync.Once
)

// TestMain builds the binary once before running all tests.
func TestMain(m *testing.M) {
	testBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "diary-test-bin-*")
		if err != nil {
			panic("failed to create temp dir for binary: " + err.Error())
		}
		bin := filepath.Join(dir, "diary")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
		if err != nil {
			panic("build failed: " + string(out))
		}
		testBinary = bin
	})
	os.Exit(m.Run())
}

// setupUser creates a temp HOME with a diary user key and storage.
func setupUser(t *testing.T, user string) {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	if err := setup.EnsureUserSetup(user); err != nil {
		t.Fatalf("EnsureUserSetup: %v", err)
	}
}

// writeEntry directly writes an encrypted entry for a specific date.
func writeEntry(t *testing.T, user, date, content string) {
	t.Helper()
	keyPath, err := crypto.KeyPath(user)
	if err != nil {
		t.Fatalf("KeyPath: %v", err)
	}
	identity, err := crypto.LoadIdentity(keyPath)
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
	entryPath, err := storage.DiaryPath(user, date)
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
	user := "pipetest"
	setupUser(t, user)

	content := "# Session Handoff\n\nSome $pecial chars & <angles>"

	cmd := exec.Command(testBinary, user, "append")
	cmd.Stdin = strings.NewReader(content)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("append via stdin failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Appended") {
		t.Errorf("expected success message, got: %s", out)
	}

	// Verify the content was actually written: read it back
	readCmd := exec.Command(testBinary, user, "read")
	readOut, err := readCmd.Output()
	if err != nil {
		t.Fatalf("read after append failed: %v", err)
	}
	if !strings.Contains(string(readOut), "Session Handoff") {
		t.Errorf("content not found in entry. read output:\n%s", readOut)
	}
}

// TestAppendEmptyStdin verifies error when stdin is a pipe with empty content.
func TestAppendEmptyStdin(t *testing.T) {
	user := "emptystdin"
	setupUser(t, user)

	cmd := exec.Command(testBinary, user, "append")
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
	user := "argwins"
	setupUser(t, user)

	argText := "positional arg content"
	stdinText := "stdin content that should be ignored"

	cmd := exec.Command(testBinary, user, "append", argText)
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
	readCmd := exec.Command(testBinary, user, "read")
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

// TestAppendSequential verifies that two consecutive appends are joined by exactly one newline.
func TestAppendSequential(t *testing.T) {
	user := "sequential"
	setupUser(t, user)

	first := "First entry line"
	second := "Second entry line"

	cmd1 := exec.Command(testBinary, user, "append", first)
	if out, err := cmd1.CombinedOutput(); err != nil {
		t.Fatalf("first append failed: %v\n%s", err, out)
	}

	cmd2 := exec.Command(testBinary, user, "append", second)
	if out, err := cmd2.CombinedOutput(); err != nil {
		t.Fatalf("second append failed: %v\n%s", err, out)
	}

	readCmd := exec.Command(testBinary, user, "read")
	readOut, err := readCmd.Output()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	output := string(readOut)
	if !strings.Contains(output, first) {
		t.Errorf("first entry not found in output:\n%s", output)
	}
	if !strings.Contains(output, second) {
		t.Errorf("second entry not found in output:\n%s", output)
	}

	// Verify they're joined by exactly one newline (not two)
	idx := strings.Index(output, first)
	if idx == -1 {
		t.Fatalf("first entry not found")
	}
	between := output[idx+len(first):]
	if !strings.HasPrefix(between, "\n"+second) {
		t.Errorf("expected exactly one newline between entries, got: %q", between[:min(len(between), 30)])
	}
}

// TestReadPlaintextDateHeader verifies that plain-text read output has a date header.
func TestReadPlaintextDateHeader(t *testing.T) {
	user := "readheader"
	setupUser(t, user)

	today := time.Now().Format("2006-01-02")
	writeEntry(t, user, today, "Some diary content")

	cmd := exec.Command(testBinary, user, "read")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("read failed: %v\n%s", err, out)
	}

	output := string(out)
	if !strings.HasPrefix(output, "Now:") {
		t.Errorf("expected output to start with 'Now:', got:\n%s", output)
	}
	if !strings.Contains(output, "Entry:") {
		t.Errorf("expected 'Entry:' line in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Some diary content") {
		t.Errorf("expected entry content after header, got:\n%s", output)
	}
}

// TestReadTUISkipsHeader verifies that -t flag (TUI mode) does not output the plain-text header.
func TestReadTUISkipsHeader(t *testing.T) {
	user := "readtui"
	setupUser(t, user)

	today := time.Now().Format("2006-01-02")
	writeEntry(t, user, today, "TUI content")

	// -t requires a real terminal; in a subprocess without a TTY bubbletea exits immediately.
	// We only verify stdout does NOT contain the "Now:" header.
	cmd := exec.Command(testBinary, user, "read", "-t")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			t.Logf("TUI stderr: %s", exitErr.Stderr)
		}
	}

	if strings.Contains(string(out), "Now:") {
		t.Errorf("TUI mode should not output 'Now:' header, got:\n%s", out)
	}
}

func TestReplaceBasic(t *testing.T) {
	user := "replacebasic"
	setupUser(t, user)

	today := time.Now().Format("2006-01-02")
	writeEntry(t, user, today, "old content that should be gone")

	newContent := "# Compacted Diary\nClean slate."
	cmd := exec.Command(testBinary, user, "replace")
	cmd.Stdin = strings.NewReader(newContent)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("replace failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Replaced") {
		t.Errorf("expected success message, got: %s", out)
	}

	readCmd := exec.Command(testBinary, user, "read")
	readOut, err := readCmd.Output()
	if err != nil {
		t.Fatalf("read after replace failed: %v", err)
	}
	output := string(readOut)
	if strings.Contains(output, "old content that should be gone") {
		t.Errorf("old content should be gone after replace, got:\n%s", output)
	}
	if !strings.Contains(output, "Compacted Diary") {
		t.Errorf("new content not found after replace, got:\n%s", output)
	}
}

func TestReplaceEmptyStdin(t *testing.T) {
	user := "replaceempty"
	setupUser(t, user)

	cmd := exec.Command(testBinary, user, "replace")
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for empty stdin, but succeeded: %s", out)
	}
	if !strings.Contains(string(out), "replacement content is empty") {
		t.Errorf("expected 'replacement content is empty' error, got: %s", out)
	}
}

func TestReplaceCreatesNewEntry(t *testing.T) {
	user := "replacenew"
	setupUser(t, user)

	content := "brand new diary content"
	cmd := exec.Command(testBinary, user, "replace")
	cmd.Stdin = strings.NewReader(content)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("replace failed: %v\n%s", err, out)
	}

	readCmd := exec.Command(testBinary, user, "read")
	readOut, err := readCmd.Output()
	if err != nil {
		t.Fatalf("read after replace failed: %v", err)
	}
	if !strings.Contains(string(readOut), "brand new diary content") {
		t.Errorf("content not found after replace, got:\n%s", readOut)
	}
}
