package tui

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestReaderEditKeybind(t *testing.T) {
	// pressing 'e' should return a Cmd that resolves to readerEditorReadyMsg
	entries := []string{"2026-03-09"}
	m := NewReaderModel("neil", "/fake/key", entries, 0)
	m.ready = true
	m.markdown = "# Test\nHello"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Fatal("expected a Cmd from 'e' keypress, got nil")
	}

	msg := cmd()
	ready, ok := msg.(readerEditorReadyMsg)
	if !ok {
		t.Fatalf("expected readerEditorReadyMsg, got %T", msg)
	}
	os.Remove(ready.tmpPath)
}

func TestReaderEditNoop(t *testing.T) {
	// unchanged content should produce nil Cmd (no save triggered)
	entries := []string{"2026-03-09"}
	m := NewReaderModel("neil", "/fake/key", entries, 0)
	m.ready = true
	m.markdown = "# Test\nHello"

	tmpFile, err := os.CreateTemp("", "diary-test-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString("# Test\nHello"); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	_, cmd := m.Update(readerEditDoneMsg{tmpPath: tmpFile.Name()})
	if cmd != nil {
		t.Fatal("expected nil Cmd for unchanged content, got non-nil")
	}
}
