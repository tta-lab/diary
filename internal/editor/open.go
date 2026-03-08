package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Open opens the content in the user's preferred editor
// Returns the edited content and any error
func Open(initialContent []byte) ([]byte, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "diary-*.md")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial content
	if _, err := tmpFile.Write(initialContent); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	bin, args := buildEditorArgs(tmpFile.Name())
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Some editors (like helix) need access to /dev/tty
	// Open the controlling terminal
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		defer tty.Close()
		cmd.Stdin = tty
		cmd.Stdout = tty
		cmd.Stderr = tty
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("editor exited with error: %w", err)
	}

	// Read edited content
	editedContent, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read edited content: %w", err)
	}

	return editedContent, nil
}

// buildEditorArgs parses the editor string and appends path, returning (binary, args).
func buildEditorArgs(path string) (string, []string) {
	parts := strings.Fields(getEditor())
	if len(parts) == 0 {
		return "vim", []string{path}
	}
	return parts[0], append(parts[1:], path)
}

// EditorCmd returns an exec.Cmd for opening path in the user's preferred editor.
// The caller is responsible for running it (e.g. via tea.ExecProcess).
func EditorCmd(path string) *exec.Cmd {
	bin, args := buildEditorArgs(path)
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func getEditor() string {
	// Try EDITOR environment variable
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// Try VISUAL environment variable
	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual
	}

	// Fallback to vim
	return "vim"
}
