package shell

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// redactedFlags are flag tokens that, if present anywhere in a history
// line, mean the line must never be persisted to disk (it may contain a
// secret). Matching is by token: a token IS one of these names, or STARTS
// WITH "name=" (the --flag=value form).
var redactedFlags = []string{"--api-key", "--auth-pass", "--auth-user"}

// History tracks shell command-line history: an in-memory session view
// (used for e.g. Ctrl-P recall and the `history` builtin, including lines
// that are redacted from disk) and a persisted subset written to disk.
type History struct {
	path      string
	max       int
	persisted []string
	session   []string
}

// LoadHistory reads history from path (0600), capped conceptually at max
// entries. If path does not exist, LoadHistory starts with empty history
// (not an error). The parent directory is NOT created here (Save creates
// it on write).
func LoadHistory(path string, max int) (*History, error) {
	h := &History{path: path, max: max}
	if path == "" {
		return h, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return h, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		h.persisted = append(h.persisted, line)
		h.session = append(h.session, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return h, nil
}

// Add appends line to the in-memory session history (always, subject to
// adjacent-dedupe) and, if Redact(line) permits, to the to-be-persisted
// history as well.
func (h *History) Add(line string) {
	if len(h.session) > 0 && h.session[len(h.session)-1] == line {
		return
	}
	h.session = append(h.session, line)
	if Redact(line) {
		h.persisted = append(h.persisted, line)
	}
}

// Save writes the persisted history (oldest-first, trimmed to h.max most
// recent entries) to h.path with mode 0600, creating the parent directory
// if needed.
func (h *History) Save() error {
	if h.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(h.path), 0o700); err != nil {
		return err
	}
	lines := h.persisted
	if h.max > 0 && len(lines) > h.max {
		lines = lines[len(lines)-h.max:]
	}
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return os.WriteFile(h.path, []byte(b.String()), 0o600)
}

// Lines returns the in-memory (session) history view, including lines
// that were redacted from disk.
func (h *History) Lines() []string {
	return h.session
}

// At returns the 1-based n'th entry of the in-memory session history (the
// same numbering the "history" builtin prints), and whether n was in
// range. Used by "history -e/--edit <num>" to select an entry to open in
// $EDITOR.
func (h *History) At(n int) (string, bool) {
	if n < 1 || n > len(h.session) {
		return "", false
	}
	return h.session[n-1], true
}

// Redact reports whether line is safe to persist to disk: false if it
// contains a token matching (or starting with, for the --flag=value form)
// any of redactedFlags. Callers should err toward not persisting (false)
// whenever in doubt.
func Redact(line string) bool {
	for _, tok := range strings.Fields(line) {
		for _, secretFlag := range redactedFlags {
			if tok == secretFlag || strings.HasPrefix(tok, secretFlag+"=") {
				return false
			}
		}
	}
	return true
}

// DefaultHistoryPath returns the platform-default shell history file path:
// $XDG_STATE_HOME/maelsink/shell_history (or ~/.local/state/maelsink/shell_history)
// on Unix, %LocalAppData%\maelsink\shell_history on Windows.
func DefaultHistoryPath() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LocalAppData")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "maelsink", "shell_history"), nil
	}

	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "maelsink", "shell_history"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "maelsink", "shell_history"), nil
}
