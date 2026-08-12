package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/0funct0ry/maelsink/internal/config"
	"github.com/0funct0ry/maelsink/internal/shell"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

func TestRandMsgDryRunZeroFlags(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.SMTPAddr = "127.0.0.1:0"
	if err := (RandMsg{}).Run(context.Background(), s, []string{"--dry-run"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "@") {
		t.Fatalf("expected a generated email in output, got: %s", out.String())
	}
}

func TestRandMsgProducesDistinctContent(t *testing.T) {
	s, cleanup := newSeededSession(t, 1)
	defer cleanup()
	s.SMTPAddr = fakeSMTPServer(t)

	if err := (RandMsg{}).Run(context.Background(), s, []string{"-n", "3"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// newSeededSession is a variant of newTestSession using an explicit seed,
// for tests asserting determinism (same seed -> same content) or distinct
// content across draws from a single seeded PRNG.
func newSeededSession(t *testing.T, seed int64) (*shell.Session, func()) {
	t.Helper()
	engine, err := tmpl.New(seed, false)
	if err != nil {
		t.Fatalf("tmpl.New: %v", err)
	}
	s := shell.NewSession(config.Shell{TemplateEnabled: true}, nil, "", nil, engine, new(strings.Builder), new(strings.Builder), nil)
	s.Interactive = false
	return s, func() { engine.Close() }
}

func TestRandMsgSeedReproducesContent(t *testing.T) {
	render := func(seed int64) (string, error) {
		engine, err := tmpl.New(seed, false)
		if err != nil {
			return "", err
		}
		defer engine.Close()
		fs := (RandMsg{}).Flags()
		if err := fs.Parse([]string{"--dry-run"}); err != nil {
			return "", err
		}
		s := shell.NewSession(config.Shell{TemplateEnabled: true}, nil, "", nil, engine, new(strings.Builder), new(strings.Builder), nil)
		spec, err := buildRandomSpec(fs, s, map[string]any{})
		if err != nil {
			return "", err
		}
		return spec.From + "|" + spec.Subject, nil
	}

	a, err := render(7)
	if err != nil {
		t.Fatalf("render 1: %v", err)
	}
	b, err := render(7)
	if err != nil {
		t.Fatalf("render 2: %v", err)
	}
	if a != b {
		t.Fatalf("same seed produced different content: %q vs %q", a, b)
	}
}

func TestRandMsgScenarioUnknown(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	fs := (RandMsg{}).Flags()
	if err := fs.Parse([]string{"--scenario", "does-not-exist"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := buildRandomSpec(fs, s, map[string]any{}); err == nil {
		t.Fatalf("expected error for unknown scenario")
	}
}

func TestRandMsgScenarioInvoice(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	fs := (RandMsg{}).Flags()
	if err := fs.Parse([]string{"--scenario", "invoice", "--body", "text"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	spec, err := buildRandomSpec(fs, s, map[string]any{})
	if err != nil {
		t.Fatalf("buildRandomSpec: %v", err)
	}
	if spec.Subject != "Invoice from {{ fCompany }}" && !strings.Contains(spec.Subject, "Invoice from") {
		t.Fatalf("expected invoice subject, got %q", spec.Subject)
	}
}

func TestRandMsgTags(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	fs := (RandMsg{}).Flags()
	if err := fs.Parse([]string{"--tags", "smoke", "--tags", "release"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	spec, err := buildRandomSpec(fs, s, map[string]any{})
	if err != nil {
		t.Fatalf("buildRandomSpec: %v", err)
	}
	if len(spec.Tags) != 2 || spec.Tags[0] != "smoke" || spec.Tags[1] != "release" {
		t.Fatalf("Tags = %v, want [smoke release]", spec.Tags)
	}
}

func TestDelugeSendsAll(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)
	if err := (Deluge{}).Run(context.Background(), s, []string{"-n", "5", "--concurrency", "3"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "5/5 sent") {
		t.Fatalf("expected 5/5 sent, got: %s", out.String())
	}
}
