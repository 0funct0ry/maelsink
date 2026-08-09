package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/0funct0ry/maelsink/internal/shell"
)

func TestHelp_ListsAll(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.SetRegistry(shell.NewRegistry(All()...))

	if err := (Help{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "list") || !strings.Contains(out.String(), "send") {
		t.Errorf("out = %q", out.String())
	}
}

func TestHelp_OneCommand(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.SetRegistry(shell.NewRegistry(All()...))

	if err := (Help{}).Run(context.Background(), s, []string{"list"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "--limit") {
		t.Errorf("out = %q", out.String())
	}
}
