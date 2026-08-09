package shell

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// ResolveEditor implements the shell.editor -> $VISUAL -> $EDITOR -> "vi"
// ("notepad" on Windows) fallback chain (SPEC.md §7.5.4/§7.5.9). It is the
// single implementation shared by internal/shell/builtin's "edit" command
// and the Ctrl-X Ctrl-E lineedit integration.
func ResolveEditor(cfgEditor string) string {
	if cfgEditor != "" {
		return cfgEditor
	}
	if v := os.Getenv("VISUAL"); v != "" {
		return v
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}

// editorCommand resolves editorOverride to a command line and builds an
// *exec.Cmd for it with extraArgs (typically the target file path)
// appended. $EDITOR/$VISUAL/shell.editor conventionally carry more than a
// bare program name — "code --wait", "vim -f", "emacsclient -nw" are all
// common — so the resolved string is split into argv via Tokenize's
// POSIX-like quoting rules (the same splitter the evaluator itself uses),
// not passed to exec.Command as a single literal program name (which
// would look for a binary literally named e.g. "code --wait" and fail).
func editorCommand(ctx context.Context, editorOverride string, extraArgs ...string) (*exec.Cmd, string, error) {
	editor := ResolveEditor(editorOverride)
	argv, err := Tokenize(editor)
	if err != nil {
		return nil, "", fmt.Errorf("edit: parsing editor command %q: %w", editor, err)
	}
	if len(argv) == 0 {
		return nil, "", fmt.Errorf("edit: editor command %q is empty", editor)
	}
	args := append(append([]string{}, argv[1:]...), extraArgs...)
	cmd := exec.CommandContext(ctx, argv[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, editor, nil
}

// RunEditor writes content to a temp file, execs the resolved editor
// (editorOverride -> $VISUAL -> $EDITOR -> vi/notepad) on it, waits for it
// to exit, and returns the reloaded file contents. It never blocks the
// terminal state on its own — callers responsible for a live readline
// session (e.g. the Ctrl-X Ctrl-E integration) must restore cooked terminal
// mode before invoking this and re-enter raw mode afterward as needed.
func RunEditor(ctx context.Context, editorOverride, content string) (string, error) {
	tmp, err := os.CreateTemp("", "maelsink-edit-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}

	cmd, editor, err := editorCommand(ctx, editorOverride, tmpPath)
	if err != nil {
		return "", err
	}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("edit: running %s: %w", editor, err)
	}

	result, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// RunEditorOnFile execs the resolved editor directly on path (no temp
// copy) and waits for it to exit. Unlike RunEditor, whatever the editor
// saves lands directly in path — there is no result to load into a line
// buffer or print, since the file itself IS the result. Used by the "edit"
// builtin's -f/--file mode: editing an actual file in place is a different
// operation from editing scratch text destined for the prompt buffer
// (RunEditor's job), even though both share the same editor-resolution
// chain.
func RunEditorOnFile(ctx context.Context, editorOverride, path string) error {
	cmd, editor, err := editorCommand(ctx, editorOverride, path)
	if err != nil {
		return err
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("edit: running %s: %w", editor, err)
	}
	return nil
}
