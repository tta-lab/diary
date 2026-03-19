package editor

import (
	"reflect"
	"testing"
)

func TestGetEditor(t *testing.T) {
	tests := []struct {
		name      string
		setEditor string
		setVisual string
		want      string
	}{
		{
			name:      "EDITOR set",
			setEditor: "nano",
			setVisual: "",
			want:      "nano",
		},
		{
			name:      "VISUAL set, no EDITOR",
			setEditor: "",
			setVisual: "emacs",
			want:      "emacs",
		},
		{
			name:      "EDITOR takes precedence",
			setEditor: "vim",
			setVisual: "emacs",
			want:      "vim",
		},
		{
			name:      "neither set - default vim",
			setEditor: "",
			setVisual: "",
			want:      "vim",
		},
		{
			name:      "editor with args",
			setEditor: "code --wait",
			setVisual: "",
			want:      "code --wait",
		},
		{
			name:      "helix editor",
			setEditor: "hx",
			setVisual: "",
			want:      "hx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment for this test
			t.Setenv("EDITOR", tt.setEditor)
			t.Setenv("VISUAL", tt.setVisual)

			got := getEditor()

			if got != tt.want {
				t.Errorf("getEditor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetEditorEnvPrecedence(t *testing.T) {
	// Test that EDITOR has precedence over VISUAL
	t.Setenv("EDITOR", "primary-editor")
	t.Setenv("VISUAL", "secondary-editor")

	editor := getEditor()

	if editor != "primary-editor" {
		t.Errorf("EDITOR should take precedence over VISUAL, got %s", editor)
	}
}

func TestGetEditorVisualFallback(t *testing.T) {
	// Test that VISUAL is used when EDITOR is not set
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "visual-editor")

	editor := getEditor()

	if editor != "visual-editor" {
		t.Errorf("Should fallback to VISUAL, got %s", editor)
	}
}

func TestGetEditorDefaultVim(t *testing.T) {
	// Test that vim is the default when neither is set
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")

	editor := getEditor()

	if editor != "vim" {
		t.Errorf("Should default to vim, got %s", editor)
	}
}

func TestEditorCmdArgOrder(t *testing.T) {
	// EditorCmd with "code --wait" should produce args: [code --wait <path>]
	t.Setenv("EDITOR", "code --wait")
	t.Setenv("VISUAL", "")

	cmd := EditorCmd("/tmp/test.md")

	want := []string{"code", "--wait", "/tmp/test.md"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("EditorCmd args = %v, want %v", cmd.Args, want)
	}
}
