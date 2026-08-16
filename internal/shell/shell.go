package shell

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/config"
	"github.com/0funct0ry/maelsink/internal/shell/lineedit"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// Options configures a shell Run invocation: connection/config info, which
// non-interactive mode (if any) to use, and I/O streams.
type Options struct {
	Cfg      config.Shell
	Client   *cliclient.Client
	SMTPAddr string
	SMTPAuth *cliclient.Auth
	SMTPTLS  cliclient.TLSOptions

	// Registry is the builtin command table. It may be nil or empty (the
	// internal/shell/builtin package is built in a later phase) — Run must
	// not crash on an empty registry; every command simply reports
	// "unknown command".
	Registry *Registry

	// Execs, when non-empty, are evaluated in order via -e/--execute
	// (SPEC.md §7.5.11).
	Execs []string
	// ScriptPath, when set (and Execs is empty), is read and evaluated
	// line by line via --script.
	ScriptPath string

	// HistoryPath overrides the history file location; if empty,
	// Cfg.HistoryFile is used, falling back to DefaultHistoryPath().
	HistoryPath string

	// Interactive, when true (and Execs/ScriptPath are both unset),
	// selects the real lineedit-backed REPL over the plain line-scanner
	// fallback. cmd/shell.go decides this via isatty — internal/shell
	// must not import go-isatty itself, so the decision is injected here.
	Interactive bool

	// NewEditor constructs the lineedit.Editor for an interactive session.
	// Required when Interactive is true; internal/shell.DefaultNewEditor
	// is the standard implementation cmd/shell.go wires up.
	NewEditor func(s *Session) (*lineedit.Editor, error)

	Stdin          io.Reader
	Stdout, Stderr io.Writer
}

// Run executes a shell session per Options and returns the process exit
// code: 0 if the last evaluated command succeeded, 1 if it failed (or if
// no command matched a supported mode and no fallback path applies).
func Run(ctx context.Context, opts Options) (int, error) {
	engine, err := tmpl.New(opts.Cfg.Seed, opts.Cfg.TemplateUnsafeFuncs)
	if err != nil {
		return 1, err
	}

	stdout := opts.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	stdin := opts.Stdin

	s := NewSession(opts.Cfg, opts.Client, opts.SMTPAddr, opts.SMTPAuth, opts.SMTPTLS, engine, stdout, stderr, stdin)
	s.ExitOnError = opts.Cfg.ExitOnError

	reg := opts.Registry
	if reg == nil {
		reg = NewRegistry()
	}
	s.SetRegistry(reg)

	histPath := opts.HistoryPath
	if histPath == "" {
		histPath = opts.Cfg.HistoryFile
	}
	if histPath == "" {
		if p, herr := DefaultHistoryPath(); herr == nil {
			histPath = p
		}
	}
	hist, err := LoadHistory(histPath, opts.Cfg.HistorySize)
	if err != nil {
		s.Close()
		return 1, err
	}
	s.SetHistory(hist)

	// $connected drives the default prompt's "(offline)" indicator
	// (SPEC.md §7.5.10) and is available to any template. Refresh it once
	// up front for EVERY mode (interactive, -e, --script, piped stdin) —
	// not just interactive — so e.g. `shell -e 'prompt'` or a script that
	// branches on {{ .connected }} sees the real state from its very first
	// command, not the NewSession default. runInteractive additionally
	// refreshes it again after each line, since an interactive session can
	// outlive the server's availability; a one-shot batch run has no
	// "later" to refresh at.
	s.RefreshConnected(ctx)

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	var signalExit atomic.Int32
	signalExit.Store(-1)
	done := make(chan struct{})
	signalHandled := make(chan struct{})
	go func() {
		defer close(signalHandled)
		select {
		case <-ctx.Done():
			_ = hist.Save()
			_ = s.Close()
			signalExit.Store(130)
		case <-done:
		}
	}()

	var runErr error
	exitCode := 0

	switch {
	case len(opts.Execs) > 0:
		exitCode, runErr = runLines(ctx, s, reg, hist, opts.Execs)
	case opts.ScriptPath != "":
		exitCode, runErr = runScriptFile(ctx, s, reg, hist, opts.ScriptPath)
	case opts.Interactive && opts.NewEditor != nil:
		s.Interactive = true
		exitCode, runErr = runInteractive(ctx, s, reg, hist, opts.NewEditor)
	default:
		// Non-interactive fallback: read logical lines from Stdin (used for
		// piped/non-interactive invocations where Interactive is false, or
		// as a safety net if no editor constructor was supplied).
		if stdin != nil {
			exitCode, runErr = runReader(ctx, s, reg, hist, stdin)
		}
	}

	close(done)
	<-signalHandled

	if code := signalExit.Load(); code >= 0 {
		return int(code), nil
	}

	_ = hist.Save()
	_ = s.Close()

	return exitCode, runErr
}

// runLines evaluates each line via Eval, honoring ExitOnError, and returns
// the exit code reflecting the last EXECUTED command's status.
func runLines(ctx context.Context, s *Session, reg *Registry, hist *History, lines []string) (int, error) {
	var lastErr error
	for _, line := range lines {
		hist.Add(line)
		err := Eval(ctx, s, reg, line)
		lastErr = err
		if code, exit := asExitError(err); exit {
			return code, nil
		}
		if err != nil && s.ExitOnError {
			break
		}
	}
	if s.LastStatus != 0 {
		return 1, lastErr
	}
	return 0, nil
}

// runInteractive drives the real lineedit-backed REPL: builds one Editor for
// the session's lifetime, reads lines until io.EOF (Ctrl-D on an empty
// buffer), evaluates each via Eval, and keeps looping on a command error —
// interactive mode never aborts on error, unlike -e/--script's
// ExitOnError-gated behavior (SPEC.md §7.5.11).
func runInteractive(ctx context.Context, s *Session, reg *Registry, hist *History, newEditor func(*Session) (*lineedit.Editor, error)) (int, error) {
	editor, err := newEditor(s)
	if err != nil {
		return 1, err
	}
	defer editor.Close()

	// Run already refreshed $connected once before dispatching here; keep
	// it live for the rest of the (potentially long-lived) interactive
	// session by refreshing again after every evaluated line, so an
	// unreachable server becomes visible on the very next prompt without
	// the user having to run a command that fails first. RefreshConnected
	// internally bounds itself to a 1s timeout, so this is a bounded,
	// once-per-line cost, not a per-keystroke one.
	for {
		pending := s.PendingBuffer
		s.PendingBuffer = ""

		var line string
		var err error
		if pending != "" {
			line, err = editor.ReadLineWithDefault(pending)
		} else {
			line, err = editor.ReadLine()
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return s.LastStatus, nil
			}
			return 1, err
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		hist.Add(line)
		if evalErr := Eval(ctx, s, reg, line); evalErr != nil {
			if code, exit := asExitError(evalErr); exit {
				return code, nil
			}
			fmt.Fprintln(s.Err, evalErr)
		}
		editor.AddHistoryLine(line)
		s.RefreshConnected(ctx)
	}
}

// asExitError reports whether err wraps an *ExitError, returning its Code.
func asExitError(err error) (int, bool) {
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code, true
	}
	return 0, false
}

// runScriptFile reads path line by line, skipping blank lines and lines
// whose first non-whitespace character is '#', evaluating the rest with
// the same exit-on-error semantics as runLines.
func runScriptFile(ctx context.Context, s *Session, reg *Registry, hist *History, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 1, err
	}
	defer f.Close()
	return runReader(ctx, s, reg, hist, f)
}

// runReader reads r line by line (used for --script and for piped stdin
// pending real interactive readline integration), skipping blank/comment
// lines, evaluating the rest with exit-on-error semantics.
func runReader(ctx context.Context, s *Session, reg *Registry, hist *History, r io.Reader) (int, error) {
	scanner := bufio.NewScanner(r)
	var lastErr error
	ran := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		hist.Add(line)
		err := Eval(ctx, s, reg, line)
		lastErr = err
		ran = true
		if code, exit := asExitError(err); exit {
			return code, nil
		}
		if err != nil && s.ExitOnError {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return 1, err
	}
	if !ran {
		return 0, nil
	}
	if s.LastStatus != 0 {
		return 1, lastErr
	}
	return 0, nil
}
