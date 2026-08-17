package msgspec

import (
	"testing"

	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

func TestBuildRandomSpecDefaults(t *testing.T) {
	engine, err := tmpl.New(1, false)
	if err != nil {
		t.Fatalf("tmpl.New: %v", err)
	}
	defer engine.Close()

	spec, err := BuildRandomSpec(engine, map[string]any{}, ContentParams{})
	if err != nil {
		t.Fatalf("BuildRandomSpec: %v", err)
	}
	if spec.From == "" || len(spec.To) == 0 || spec.To[0] == "" {
		t.Fatalf("expected default From/To to be generated, got %+v", spec)
	}
	if spec.Subject == "" {
		t.Fatalf("expected default Subject to be generated")
	}
}

func TestBuildRandomSpecUnknownScenario(t *testing.T) {
	engine, err := tmpl.New(1, false)
	if err != nil {
		t.Fatalf("tmpl.New: %v", err)
	}
	defer engine.Close()

	if _, err := BuildRandomSpec(engine, map[string]any{}, ContentParams{Scenario: "not-a-real-scenario"}); err == nil {
		t.Fatalf("expected error for unknown scenario")
	}
}

func TestBuildRandomSpecScenarioSeedsSubjectAndBody(t *testing.T) {
	engine, err := tmpl.New(1, false)
	if err != nil {
		t.Fatalf("tmpl.New: %v", err)
	}
	defer engine.Close()

	spec, err := BuildRandomSpec(engine, map[string]any{}, ContentParams{Scenario: "Invoice", Body: "text"})
	if err != nil {
		t.Fatalf("BuildRandomSpec: %v", err)
	}
	if spec.Subject != "Invoice from {{ fCompany }}" && spec.Text == "" {
		t.Fatalf("expected scenario-seeded subject/text, got subject=%q text=%q", spec.Subject, spec.Text)
	}
}

func TestBuildRandomSpecBodyModes(t *testing.T) {
	engine, err := tmpl.New(1, false)
	if err != nil {
		t.Fatalf("tmpl.New: %v", err)
	}
	defer engine.Close()

	spec, err := BuildRandomSpec(engine, map[string]any{}, ContentParams{Body: "both"})
	if err != nil {
		t.Fatalf("BuildRandomSpec: %v", err)
	}
	if spec.Text == "" || spec.HTML == "" {
		t.Fatalf("expected both Text and HTML for --body both, got %+v", spec)
	}
}
