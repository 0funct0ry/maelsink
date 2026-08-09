package shell

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestRunEditor_SplitsMultiWordEditorCommand proves $EDITOR/$VISUAL/
// shell.editor values that carry extra arguments — a common real-world
// convention (e.g. "code --wait", "vim -f", or a one-liner shell script
// invocation) — actually work, rather than being handed to exec.Command
// as one literal (and nonexistent) program name containing spaces.
func TestRunEditor_SplitsMultiWordEditorCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh -c as the editor")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "ran")
	editorCmd := "/bin/sh -c 'echo ran > " + marker + "; echo appended >> \"$1\"' --"

	result, err := RunEditor(context.Background(), editorCmd, "original")
	if err != nil {
		t.Fatalf("RunEditor: %v", err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Fatalf("editor command did not run (marker file missing): %v", statErr)
	}
	if !strings.Contains(result, "original") || !strings.Contains(result, "appended") {
		t.Errorf("result = %q", result)
	}
}

func TestRunEditorOnFile_SplitsMultiWordEditorCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh -c as the editor")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("original\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	editorCmd := "/bin/sh -c 'echo appended >> \"$1\"' --"

	if err := RunEditorOnFile(context.Background(), editorCmd, target); err != nil {
		t.Fatalf("RunEditorOnFile: %v", err)
	}
	saved, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(saved), "original") || !strings.Contains(string(saved), "appended") {
		t.Errorf("file contents = %q", saved)
	}
}

func TestEditBuffer_SplitsMultiWordEditorCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh -c as the editor")
	}
	editorCmd := "/bin/sh -c 'echo appended >> \"$1\"' --"

	result, err := editBuffer(editorCmd, "list --limit 10")
	if err != nil {
		t.Fatalf("editBuffer: %v", err)
	}
	if !strings.Contains(result, "list --limit 10") || !strings.Contains(result, "appended") {
		t.Errorf("result = %q", result)
	}
}
