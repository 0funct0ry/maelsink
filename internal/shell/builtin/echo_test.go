package builtin

import (
	"context"
	"testing"

	"github.com/0funct0ry/maelsink/internal/shell"
)

func TestEcho_PrintsArgsJoinedWithNewline(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Echo{}).Run(context.Background(), s, []string{"hello", "there"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got, want := out.String(), "hello there\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestEcho_NoNewline(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Echo{}).Run(context.Background(), s, []string{"-n", "hello"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got, want := out.String(), "hello"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestEcho_NoArgsPrintsBlankLine(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Echo{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got, want := out.String(), "\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

// TestEcho_TemplateExpansionHappensInEvalNotEcho proves the doc comment's
// claim: Echo.Run never touches s.Tmpl itself, but running "echo {{ ... }}"
// through the full shell.Eval pipeline (as the interactive/-e/--script/
// piped-stdin loops all do) still expands variables and template functions
// before Echo ever sees its args — because ExpandTemplate runs on the whole
// line ahead of tokenization/dispatch (SPEC.md §7.5.3).
func TestEcho_TemplateExpansionHappensInEvalNotEcho(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.SetVar("foo", "bar")
	reg := shell.NewRegistry(Echo{})

	if err := shell.Eval(context.Background(), s, reg, `echo {{ .foo }} {{ add 1 2 }}`); err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if got, want := out.String(), "bar 3\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestEcho_Help(t *testing.T) {
	if (Echo{}).Name() != "echo" {
		t.Fatalf("Name() = %q", (Echo{}).Name())
	}
	if (Echo{}).Short() == "" {
		t.Error("Short() should not be empty")
	}
}
