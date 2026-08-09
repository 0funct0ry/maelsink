package cliclient

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	msgs := []MessageSummary{
		{ID: "msg_1", From: "a@b.com", Subject: "hi"},
		{ID: "msg_2", From: "c@d.com", Subject: "there"},
	}

	var buf bytes.Buffer
	if err := RenderTemplate(&buf, msgs, "{{.ID}}: {{.From}} ({{.Subject}})"); err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}

	want := "msg_1: a@b.com (hi)\nmsg_2: c@d.com (there)\n"
	if buf.String() != want {
		t.Fatalf("got %q, want %q", buf.String(), want)
	}
}

func TestRenderTemplate_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderTemplate(&buf, nil, "{{.ID}}"); err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no output for empty message list, got %q", buf.String())
	}
}

func TestRenderTemplate_InvalidSyntax(t *testing.T) {
	var buf bytes.Buffer
	err := RenderTemplate(&buf, []MessageSummary{{ID: "msg_1"}}, "{{.ID")
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parsing --format template") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenderTemplate_UnknownField(t *testing.T) {
	var buf bytes.Buffer
	err := RenderTemplate(&buf, []MessageSummary{{ID: "msg_1"}}, "{{.NoSuchField}}")
	if err == nil {
		t.Fatal("expected execution error")
	}
	if !strings.Contains(err.Error(), "executing --format template") {
		t.Fatalf("unexpected error: %v", err)
	}
}
