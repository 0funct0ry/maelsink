package shell

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEditBuffer_StripsTrailingNewline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a unix shell script as a fake editor")
	}
	dir := t.TempDir()
	fakeEditor := filepath.Join(dir, "fake-editor.sh")
	if err := os.WriteFile(fakeEditor, []byte("#!/bin/sh\necho appended >> \"$1\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result, err := editBuffer(fakeEditor, "list --limit 10")
	if err != nil {
		t.Fatalf("editBuffer: %v", err)
	}
	if !strings.Contains(result, "list --limit 10") || !strings.Contains(result, "appended") {
		t.Errorf("result = %q", result)
	}
	if strings.HasSuffix(result, "\n") {
		t.Errorf("editBuffer's result must not have a trailing newline (it becomes a single-line readline buffer via Ctrl-X Ctrl-E): %q", result)
	}
}
