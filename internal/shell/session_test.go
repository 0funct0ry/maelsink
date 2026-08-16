package shell

import (
	"io"
	"strings"
	"testing"
	"text/template"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/config"
)

// TestNewSession_ConnectedDefaultsToFalsyEmptyNotLiteralFalse guards
// against a real bug: Go template truthiness treats ANY non-empty string
// — including the literal string "false" — as true. If $connected were
// initialized to "false" instead of "", the SPEC.md §7.5.10 default
// prompt's {{ if not .connected }} would silently always evaluate to
// false and never show "(offline)".
func TestNewSession_ConnectedDefaultsToFalsyEmptyNotLiteralFalse(t *testing.T) {
	s := NewSession(config.Shell{}, nil, "", nil, cliclient.TLSOptions{}, nil, io.Discard, io.Discard, nil)
	got, ok := s.GetVar("connected")
	if !ok {
		t.Fatal("expected $connected to be initialized, found nothing")
	}
	if got != "" {
		t.Errorf("GetVar(connected) = %q, want \"\" (falsy) — NOT the string \"false\", which text/template treats as truthy", got)
	}

	tpl := template.Must(template.New("t").Parse(`{{ if not .connected }}offline{{ end }}`))
	var buf strings.Builder
	if err := tpl.Execute(&buf, s.TemplateData()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if buf.String() != "offline" {
		t.Errorf("{{ if not .connected }} rendered %q, want \"offline\"", buf.String())
	}
}

func TestSetGetVar(t *testing.T) {
	s := &Session{}
	if _, ok := s.GetVar("missing"); ok {
		t.Fatal("expected missing var to not be found")
	}
	s.SetVar("foo", "bar")
	got, ok := s.GetVar("foo")
	if !ok || got != "bar" {
		t.Errorf("GetVar = %q, %v; want bar, true", got, ok)
	}
}

func TestTemplateDataMergesReservedVars(t *testing.T) {
	s := &Session{Vars: map[string]string{
		"foo":       "bar",
		"connected": "true",
		"last_id":   "abc123",
		"status":    "ok",
	}}
	data := s.TemplateData()
	for k, want := range map[string]string{
		"foo":       "bar",
		"connected": "true",
		"last_id":   "abc123",
		"status":    "ok",
	} {
		if got := data[k]; got != want {
			t.Errorf("data[%q] = %q, want %q", k, got, want)
		}
	}
}
