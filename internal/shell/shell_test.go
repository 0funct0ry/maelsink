package shell

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0funct0ry/maelsink/internal/config"
)

func testRegistry() *Registry {
	ok := &stubBuiltin{name: "ok", run: func(ctx context.Context, s *Session, args []string) error {
		return nil
	}}
	fail := &stubBuiltin{name: "fail", run: func(ctx context.Context, s *Session, args []string) error {
		return errors.New("boom")
	}}
	echo := &stubBuiltin{name: "echo", run: func(ctx context.Context, s *Session, args []string) error {
		s.Out.Write([]byte(strings.Join(args, " ") + "\n"))
		return nil
	}}
	return NewRegistry(ok, fail, echo)
}

func baseOpts(t *testing.T) Options {
	t.Helper()
	return Options{
		Cfg:         config.Shell{TemplateEnabled: false},
		Registry:    testRegistry(),
		HistoryPath: filepath.Join(t.TempDir(), "hist"),
		Stdout:      new(bytes.Buffer),
		Stderr:      new(bytes.Buffer),
	}
}

func TestRunExecsSuccess(t *testing.T) {
	opts := baseOpts(t)
	opts.Execs = []string{"ok", "echo hi"}
	code, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if got := opts.Stdout.(*bytes.Buffer).String(); !strings.Contains(got, "hi") {
		t.Errorf("stdout = %q, want to contain hi", got)
	}
}

func TestRunExecsFailureExitCode(t *testing.T) {
	opts := baseOpts(t)
	opts.Execs = []string{"ok", "fail"}
	code, _ := Run(context.Background(), opts)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestRunExecsExitOnErrorStopsEarly(t *testing.T) {
	opts := baseOpts(t)
	opts.Cfg.ExitOnError = true
	opts.Execs = []string{"fail", "echo should-not-run"}
	code, _ := Run(context.Background(), opts)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if got := opts.Stdout.(*bytes.Buffer).String(); strings.Contains(got, "should-not-run") {
		t.Errorf("expected exit-on-error to stop before echo ran, got stdout %q", got)
	}
}

func TestRunExecsContinuesWithoutExitOnError(t *testing.T) {
	opts := baseOpts(t)
	opts.Cfg.ExitOnError = false
	opts.Execs = []string{"fail", "echo did-run"}
	code, _ := Run(context.Background(), opts)
	if code != 0 {
		t.Errorf("exit code = %d, want 0 (last command succeeded)", code)
	}
	if got := opts.Stdout.(*bytes.Buffer).String(); !strings.Contains(got, "did-run") {
		t.Errorf("expected echo to have run, got stdout %q", got)
	}
}

func TestRunScriptFileSkipsCommentsAndBlankLines(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "script.msh")
	content := "# a comment\n\n  \nok\necho from-script\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	opts := baseOpts(t)
	opts.ScriptPath = scriptPath
	code, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if got := opts.Stdout.(*bytes.Buffer).String(); !strings.Contains(got, "from-script") {
		t.Errorf("stdout = %q, want to contain from-script", got)
	}
}

func TestRunPipedStdin(t *testing.T) {
	opts := baseOpts(t)
	opts.Stdin = strings.NewReader("echo piped-input\nok\n")
	code, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if got := opts.Stdout.(*bytes.Buffer).String(); !strings.Contains(got, "piped-input") {
		t.Errorf("stdout = %q, want to contain piped-input", got)
	}
}

func TestRunEmptyRegistryUnknownCommand(t *testing.T) {
	opts := baseOpts(t)
	opts.Registry = NewRegistry()
	opts.Execs = []string{"whatever"}
	code, _ := Run(context.Background(), opts)
	if code != 1 {
		t.Errorf("exit code = %d, want 1 for unknown command", code)
	}
}
