package builtin

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/0funct0ry/maelsink/internal/shell"
)

func TestSetUnsetVars(t *testing.T) {
	s, out, _ := newTestSession(t, nil)

	if err := (Set{}).Run(context.Background(), s, []string{"foo=bar"}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if v, _ := s.GetVar("foo"); v != "bar" {
		t.Errorf("foo = %q", v)
	}

	out.Reset()
	if err := (Vars{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("vars: %v", err)
	}
	if !strings.Contains(out.String(), "foo") {
		t.Errorf("vars output = %q", out.String())
	}

	if err := (Unset{}).Run(context.Background(), s, []string{"foo"}); err != nil {
		t.Fatalf("unset: %v", err)
	}
	if _, ok := s.GetVar("foo"); ok {
		t.Error("foo should be unset")
	}
}

func TestSet_BareReadNoEditor(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	s.SetVar("k", "v")
	var out strings.Builder
	s.Out = &out
	if err := (Set{}).Run(context.Background(), s, []string{"k"}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if strings.TrimSpace(out.String()) != "v" {
		t.Errorf("out = %q", out.String())
	}
}

func TestAliasUnalias(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Alias{}).Run(context.Background(), s, []string{"l=list --limit 5"}); err != nil {
		t.Fatalf("alias: %v", err)
	}
	if s.Aliases["l"] != "list --limit 5" {
		t.Errorf("aliases = %v", s.Aliases)
	}
	out.Reset()
	if err := (Alias{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("alias list: %v", err)
	}
	if !strings.Contains(out.String(), "l=list --limit 5") {
		t.Errorf("out = %q", out.String())
	}
	if err := (Unalias{}).Run(context.Background(), s, []string{"l"}); err != nil {
		t.Fatalf("unalias: %v", err)
	}
	if _, ok := s.Aliases["l"]; ok {
		t.Error("l should be removed")
	}
}

func TestAbbrUnabbr(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	if err := (Abbr{}).Run(context.Background(), s, []string{"gco=git checkout"}); err != nil {
		t.Fatalf("abbr: %v", err)
	}
	if s.Abbrs["gco"] != "git checkout" {
		t.Errorf("abbrs = %v", s.Abbrs)
	}
	if err := (Unabbr{}).Run(context.Background(), s, []string{"--all"}); err != nil {
		t.Fatalf("unabbr: %v", err)
	}
	if len(s.Abbrs) != 0 {
		t.Errorf("abbrs should be empty, got %v", s.Abbrs)
	}
}

func TestTemplate_Basic(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Template{}).Run(context.Background(), s, []string{"{{ 1 | add 2 }}"}); err != nil {
		t.Fatalf("template: %v", err)
	}
	if strings.TrimSpace(out.String()) != "3" {
		t.Errorf("out = %q", out.String())
	}
}

func TestTemplate_Funcs(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Template{}).Run(context.Background(), s, []string{"--funcs"}); err != nil {
		t.Fatalf("template --funcs: %v", err)
	}
	if !strings.Contains(out.String(), "uuid") {
		t.Errorf("out should list uuid func, got %q", out.String())
	}
}

func TestExit_ReturnsExitError(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	err := (Exit{}).Run(context.Background(), s, []string{"7"})
	var exitErr *shell.ExitError
	if err == nil {
		t.Fatal("expected an ExitError")
	}
	if ee, ok := err.(*shell.ExitError); ok {
		exitErr = ee
	}
	if exitErr == nil || exitErr.Code != 7 {
		t.Errorf("err = %v", err)
	}
}

func TestSource_RunsLines(t *testing.T) {
	dir := t.TempDir()
	scriptPath := dir + "/script.sh"
	if err := os.WriteFile(scriptPath, []byte("set a=1\nset b=2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s, _, _ := newTestSession(t, nil)
	s.SetRegistry(shell.NewRegistry(All()...))

	if err := (Source{}).Run(context.Background(), s, []string{scriptPath, "--quiet"}); err != nil {
		t.Fatalf("source: %v", err)
	}
	if v, _ := s.GetVar("a"); v != "1" {
		t.Errorf("a = %q", v)
	}
	if v, _ := s.GetVar("b"); v != "2" {
		t.Errorf("b = %q", v)
	}
}
