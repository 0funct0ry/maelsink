package builtin

import (
	"context"
	"strings"
	"testing"
)

func TestFunctions_ListsKnownNames(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Functions{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, name := range []string{"fakeEmail", "uuid", "regex", "ansiRed", "upper"} {
		if !strings.Contains(out.String(), name) {
			t.Errorf("expected listing to contain %q", name)
		}
	}
}

// TestFunctions_NoUnavailableHelpText proves every function in the
// listing has a real description — regression test for the reported bug
// where ~200 sprig functions showed "Help text unavailable." (fixed by
// tmpl/docs_sprig.go's full sprig FuncMap documentation).
func TestFunctions_NoUnavailableHelpText(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Functions{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.Contains(out.String(), "unavailable") {
		t.Errorf("at least one function still has no real help text:\n%s", out.String())
	}
}

func TestFunctions_DetailForMaelsinkFunction(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Functions{}).Run(context.Background(), s, []string{"fakeEmail"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "fakeEmail") || !strings.Contains(out.String(), "email") {
		t.Errorf("out = %q", out.String())
	}
}

// TestFunctions_DetailForSprigFunctionHasRealDocs proves sprig functions
// get real, specific documentation (tmpl.docs_sprig.go) rather than the
// generic "sprig utility function, see the docs" fallback — every real
// FuncMap entry has one (internal/shell/tmpl's TestDocs_CoversEveryFuncMapEntry
// guards this at the source).
func TestFunctions_DetailForSprigFunctionHasRealDocs(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Functions{}).Run(context.Background(), s, []string{"upper"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "Uppercases") {
		t.Errorf("expected a real description, got %q", out.String())
	}
	if strings.Contains(out.String(), "sprig utility function — see") {
		t.Errorf("should not fall back to the generic note when a real doc exists, got %q", out.String())
	}
}

func TestFunctions_UnknownName(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	if err := (Functions{}).Run(context.Background(), s, []string{"totallyNotAFunction"}); err == nil {
		t.Fatal("expected error for unknown function name")
	}
}
