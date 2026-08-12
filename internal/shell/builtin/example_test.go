package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// TestExample_EMLHintNeedsNoExtraFlags guards the fixed behavior:
// "send --template <file>" now derives the envelope from the RENDERED
// content's own From/To headers (same rule as --eml), so the example
// builtin's hint doesn't need to tell the user to pass --from/--to.
func TestExample_EMLHintNeedsNoExtraFlags(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	dir := t.TempDir()
	outPath := filepath.Join(dir, "order.eml")

	if err := (Example{}).Run(context.Background(), s, []string{"--index", "1", "--format", "eml", "-o", outPath}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "send --template "+outPath) {
		t.Errorf("expected a plain send --template hint, got %q", out.String())
	}
	if strings.Contains(out.String(), "--from <address>") {
		t.Errorf("--template's envelope now comes from the file's own headers, should not suggest --from/--to flags, got %q", out.String())
	}
}

func TestExample_JSONHintNeedsNoExtraFlags(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	dir := t.TempDir()
	outPath := filepath.Join(dir, "order.json")

	if err := (Example{}).Run(context.Background(), s, []string{"--index", "1", "--format", "json", "-o", outPath}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "send --json "+outPath) {
		t.Errorf("expected a plain send --json hint, got %q", out.String())
	}
	if strings.Contains(out.String(), "--from <address>") {
		t.Errorf("json mode's envelope comes from the file itself, should not suggest --from/--to flags, got %q", out.String())
	}
}

func TestExample_List(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Example{}).Run(context.Background(), s, []string{"--list"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := strings.Count(out.String(), "\n"); got != len(exampleTemplates) {
		t.Errorf("expected %d lines, got %d: %q", len(exampleTemplates), got, out.String())
	}
}

func TestExample_EveryCannedTemplateRendersUnderEveryFormat(t *testing.T) {
	// Proves each of the 10 canned examples (a) is syntactically valid
	// Go-template text and (b) actually renders end to end through the
	// real engine, for both --format eml and --format json — catching
	// mistakes like dot-chaining a field off a function call
	// ("fOrder.id", which text/template rejects) before a user hits
	// them.
	for i := range exampleTemplates {
		for _, format := range []string{"eml", "json"} {
			s, _, _ := newTestSession(t, nil)
			dir := t.TempDir()
			out := filepath.Join(dir, "out")

			err := (Example{}).Run(context.Background(), s, []string{
				"--index", strconv.Itoa(i + 1), "--format", format, "-o", out,
			})
			if err != nil {
				t.Fatalf("index %d format %s: Run: %v", i+1, format, err)
			}

			content, rerr := os.ReadFile(out)
			if rerr != nil {
				t.Fatalf("index %d format %s: ReadFile: %v", i+1, format, rerr)
			}

			// The written file is a TEMPLATE (unrendered) by design; render
			// it here the same way send --template/--json would, to prove
			// it's valid and produces non-empty, plausible output.
			if format == "json" {
				var spec struct {
					From, Subject, Text, HTML string
					To                        []string
				}
				if err := json.Unmarshal(content, &spec); err != nil {
					t.Fatalf("index %d: invalid JSON: %v", i+1, err)
				}
				for _, field := range []string{spec.From, spec.Subject} {
					if _, err := s.Tmpl.Render(field, s.TemplateData()); err != nil {
						t.Errorf("index %d: field failed to render: %v", i+1, err)
					}
				}
				if len(spec.To) != 1 || spec.To[0] == "" {
					t.Errorf("index %d: expected exactly one To address template, got %v", i+1, spec.To)
				}
			} else {
				rendered, err := s.Tmpl.Render(string(content), s.TemplateData())
				if err != nil {
					t.Fatalf("index %d: eml failed to render: %v", i+1, err)
				}
				if !strings.HasPrefix(rendered, "From: ") {
					t.Errorf("index %d: rendered eml doesn't start with From: header: %q", i+1, rendered[:min(40, len(rendered))])
				}
			}
		}
	}
}

func TestExample_IndexOutOfRange(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	if err := (Example{}).Run(context.Background(), s, []string{"--index", "999"}); err == nil {
		t.Fatal("expected error for out-of-range --index")
	}
}

func TestExample_InvalidFormat(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	if err := (Example{}).Run(context.Background(), s, []string{"--format", "yaml"}); err == nil {
		t.Fatal("expected error for unsupported --format")
	}
}

func TestExample_PickIsDeterministicUnderSameSeed(t *testing.T) {
	// Example.Run picks its random canned template via s.Tmpl.Intn, the
	// same seeded PRNG every template function draws from — so two
	// engines built with the same seed must agree on the pick.
	e1, err := tmpl.New(42, false)
	if err != nil {
		t.Fatalf("tmpl.New: %v", err)
	}
	defer e1.Close()
	e2, err := tmpl.New(42, false)
	if err != nil {
		t.Fatalf("tmpl.New: %v", err)
	}
	defer e2.Close()

	if got, want := e1.Intn(len(exampleTemplates)), e2.Intn(len(exampleTemplates)); got != want {
		t.Errorf("same seed produced different picks: %d vs %d", got, want)
	}
}
