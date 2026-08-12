package builtin

import (
	"context"
	"strings"
	"testing"
)

func TestFunctions_GroupedListing(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Functions{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := out.String()
	for _, header := range []string{"identifiers:", "generate:", "email:"} {
		if !strings.Contains(got, header) {
			t.Errorf("expected grouped listing to contain category header %q, got:\n%s", header, got)
		}
	}
	for _, name := range []string{"uuid", "regex", "attach"} {
		if !strings.Contains(got, name) {
			t.Errorf("expected listing to contain %q", name)
		}
	}
}

func TestFunctions_NoUnavailableHelpText(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Functions{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.Contains(out.String(), "unavailable") {
		t.Errorf("at least one function still has no real help text:\n%s", out.String())
	}
}

func TestFunctions_CategoryFilter(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Functions{}).Run(context.Background(), s, []string{"generate"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "oneOf") || !strings.Contains(got, "fName") {
		t.Errorf("expected generate category listing to contain oneOf/fName, got:\n%s", got)
	}
	if strings.Contains(got, "identifiers:") {
		t.Errorf("category-filtered listing should not be grouped by header, got:\n%s", got)
	}
	if strings.Contains(got, "uuid") {
		t.Errorf("category filter for 'generate' should not include 'uuid' (identifiers category), got:\n%s", got)
	}
}

func TestFunctions_DetailForMaelsinkFunction(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Functions{}).Run(context.Background(), s, []string{"fEmail"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "fEmail") || !strings.Contains(out.String(), "email") {
		t.Errorf("out = %q", out.String())
	}
}

func TestFunctions_DetailForSprigFunctionHasRealDocs(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Functions{}).Run(context.Background(), s, []string{"upper"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "Uppercases") {
		t.Errorf("expected a real description, got %q", out.String())
	}
}

func TestFunctions_UnknownNameOrCategory(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	err := (Functions{}).Run(context.Background(), s, []string{"totallyNotAFunction"})
	if err == nil {
		t.Fatal("expected error for unknown function/category name")
	}
	if !strings.Contains(err.Error(), "no such function or category") {
		t.Errorf("err = %v, want mention of 'no such function or category'", err)
	}
}

func TestFunctions_Search(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Functions{}).Run(context.Background(), s, []string{"-s", "email"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := out.String()
	for _, name := range []string{"messageID", "mimeWord", "attach", "fileOf"} {
		if !strings.Contains(got, name) {
			t.Errorf("expected -s email to match %q, got:\n%s", name, got)
		}
	}
}

func TestFunctions_AliasesBehaveIdentically(t *testing.T) {
	if got := (Functions{}).Aliases(); len(got) != 2 || got[0] != "fns" || got[1] != "funcs" {
		t.Fatalf("Aliases() = %v, want [fns funcs]", got)
	}

	s1, out1, _ := newTestSession(t, nil)
	s2, out2, _ := newTestSession(t, nil)
	if err := (Functions{}).Run(context.Background(), s1, []string{"-s", "email"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if err := (Functions{}).Run(context.Background(), s2, []string{"-s", "email"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out1.String() != out2.String() {
		t.Errorf("expected identical output across two invocations, got %q vs %q", out1.String(), out2.String())
	}
}
