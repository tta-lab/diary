package editor

import (
	"fmt"
	"os"
	"os/exec"
)

// Open opens the content in the user's preferred editor
// Returns the edited content and any error
func Open(initialContent []byte) ([]byte, error) {
	editor := getEditor()

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

	// Open in editor
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

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
