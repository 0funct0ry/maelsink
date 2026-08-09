package builtin

import (
	"context"
	"strings"
	"testing"
)

func TestPrompt_ShowsCurrent(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.Cfg.Prompt = "maelsink> "
	if err := (Prompt{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "maelsink> ") {
		t.Errorf("out = %q", out.String())
	}
}

func TestPrompt_SetsTemplate(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Prompt{}).Run(context.Background(), s, []string{"{{ .connected }}", ">"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if s.Cfg.Prompt != "{{ .connected }} >" {
		t.Errorf("Cfg.Prompt = %q", s.Cfg.Prompt)
	}
	if !strings.Contains(out.String(), "template: {{ .connected }} >") {
		t.Errorf("out = %q", out.String())
	}
	// $connected is initialized to "" (empty, i.e. false) at session
	// construction — never an absent/invalid map key, and never the
	// literal string "false" (which Go template truthiness would treat as
	// TRUE, since any non-empty string is truthy) — so referencing it
	// directly, including passed into a function like ansiCyan, never
	// errors even before RefreshConnected has run.
	if !strings.Contains(out.String(), "preview:   >") {
		t.Errorf("out = %q", out.String())
	}
}

func TestPrompt_DefaultShowsOfflineWhenDisconnected(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.Cfg.Prompt = defaultPromptTemplate
	if err := (Prompt{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Checking the rendered "preview:" line specifically, not the whole
	// output — the "template:" line always echoes the literal, unrendered
	// "(offline)" text as part of the template syntax itself.
	preview := previewLine(t, out.String())
	if !strings.Contains(preview, "(offline)") {
		t.Errorf("expected the rendered preview to show (offline) when $connected is empty/false, got %q", preview)
	}
}

func TestPrompt_DefaultHidesOfflineWhenConnected(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.Cfg.Prompt = defaultPromptTemplate
	s.SetVar("connected", "true")
	if err := (Prompt{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// The "template:" line always echoes the raw stored template, which
	// literally contains the substring "(offline)" as Go-template syntax
	// regardless of whether it renders — only the "preview:" line (the
	// actually-rendered result) should be checked here.
	preview := previewLine(t, out.String())
	if strings.Contains(preview, "(offline)") {
		t.Errorf("expected no (offline) marker in the rendered preview when $connected is \"true\", got %q", preview)
	}
}

// previewLine extracts the "preview:  ..." line from prompt's output.
func previewLine(t *testing.T, out string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "preview:") {
			return line
		}
	}
	t.Fatalf("no preview line found in output: %q", out)
	return ""
}

func TestPrompt_VariablesAndFunctionsRenderInPreview(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.SetVar("env", "staging")
	if err := (Prompt{}).Run(context.Background(), s, []string{"[{{ .env }}] maelsink>"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "preview:  [staging] maelsink>") {
		t.Errorf("out = %q", out.String())
	}
}

func TestPrompt_Reset(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	s.Cfg.Prompt = "custom> "
	if err := (Prompt{}).Run(context.Background(), s, []string{"--reset"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if s.Cfg.Prompt != defaultPromptTemplate {
		t.Errorf("Cfg.Prompt = %q, want default", s.Cfg.Prompt)
	}
}
