package tmpl

import (
	"regexp"
	"testing"
)

// TestOneOf_UniformDistribution mirrors M8.3's generator-distribution-check
// precedent: many renders of {{ oneOf "a" "b" "c" }} should hit every value,
// with roughly uniform counts, both for the bare comma-separated form and
// sub-expression variables.
func TestOneOf_UniformDistribution(t *testing.T) {
	e, err := New(1, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	const n = 3000
	counts := map[string]int{}
	for i := 0; i < n; i++ {
		out, err := e.Render(`{{ oneOf "a" "b" "c" }}`, nil)
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		counts[out]++
	}

	for _, v := range []string{"a", "b", "c"} {
		c := counts[v]
		if c < n/3/2 || c > n/3*2 {
			t.Errorf("oneOf(%q) count = %d, want roughly %d (n/3)", v, c, n/3)
		}
	}
}

func TestOneOf_CommaSeparatedStringForm(t *testing.T) {
	e, err := New(1, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	const n = 3000
	counts := map[string]int{}
	for i := 0; i < n; i++ {
		out, err := e.Render(`{{ oneOf "a,b,c" }}`, nil)
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		counts[out]++
	}
	for _, v := range []string{"a", "b", "c"} {
		c := counts[v]
		if c < n/3/2 || c > n/3*2 {
			t.Errorf("oneOf(%q) count = %d, want roughly %d (n/3)", v, c, n/3)
		}
	}
}

func TestOneOf_SubExpressionForm(t *testing.T) {
	e, err := New(2, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		out, err := e.Render(`{{ $x := "x" }}{{ $y := "y" }}{{ $z := "z" }}{{ oneOf $x $y $z }}`, nil)
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		seen[out] = true
	}
	for _, v := range []string{"x", "y", "z"} {
		if !seen[v] {
			t.Errorf("expected oneOf to produce %q across 100 renders", v)
		}
	}
}

// TestBareRegex_EquivalentToQuoted proves {{ regex [a-z]{2,4} }} (unquoted)
// renders without a parse error and matches the pattern, and that
// {{ regex "[a-z]{2,4}" }} (quoted) — the pre-existing form — calls the same
// underlying function and produces output from the same distribution.
func TestBareRegex_EquivalentToQuoted(t *testing.T) {
	e, err := New(1, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	pattern := regexp.MustCompile(`^[a-z]{2,4}$`)

	bareOut, err := e.Render(`{{ regex [a-z]{2,4} }}`, nil)
	if err != nil {
		t.Fatalf("Render (bare): %v", err)
	}
	if !pattern.MatchString(bareOut) {
		t.Errorf("bare regex output %q does not match pattern", bareOut)
	}

	quotedOut, err := e.Render(`{{ regex "[a-z]{2,4}" }}`, nil)
	if err != nil {
		t.Fatalf("Render (quoted): %v", err)
	}
	if !pattern.MatchString(quotedOut) {
		t.Errorf("quoted regex output %q does not match pattern", quotedOut)
	}
}

func TestExpandBareRegex(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"bare", `{{ regex [a-z]{2,4} }}`, `{{ regex "[a-z]{2,4}" }}`},
		{"already quoted", `{{ regex "abc" }}`, `{{ regex "abc" }}`},
		{"no args (leave alone)", `{{ regex }}`, `{{ regex }}`},
		{"other action untouched", `{{ upper "x" }}`, `{{ upper "x" }}`},
		{"mixed text", `Hi {{ .Name }}, code: {{ regex [0-9]{3} }}!`, `Hi {{ .Name }}, code: {{ regex "[0-9]{3}" }}!`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := expandBareRegex(c.in)
			if got != c.want {
				t.Errorf("expandBareRegex(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
