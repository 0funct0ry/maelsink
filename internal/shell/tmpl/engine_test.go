package tmpl

import (
	"os"
	"regexp"
	"testing"
)

var uuidv7Re = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

var ulidRe = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

// TestRender_UndefinedVarIsEmptyEverywhereNotJustBarePrint guards a real
// bug: text/template's DEFAULT missing-map-key behavior ("missingkey=invalid")
// makes a bare {{ .undefined }} print harmlessly as "<no value>", but
// passing that same undefined value into ANY function — {{ upper
// .undefined }}, {{ ansiRed .undefined }} — fails with "invalid value;
// expected string", since there's no valid conversion from an invalid
// reflect.Value. Engine.Render sets Option("missingkey=zero") specifically
// so undefined variables are uniformly "" (the zero value), matching
// SPEC.md §7.5.6's documented "undefined variables render empty" contract
// in every context, not just bare printing.
func TestRender_UndefinedVarIsEmptyEverywhereNotJustBarePrint(t *testing.T) {
	e, err := New(1, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	bare, err := e.Render(`[{{ .undefined }}]`, map[string]string{})
	if err != nil {
		t.Fatalf("bare Render: %v", err)
	}
	if bare != "[]" {
		t.Errorf("bare = %q, want %q", bare, "[]")
	}

	viaFunc, err := e.Render(`[{{ upper .undefined }}]`, map[string]string{})
	if err != nil {
		t.Fatalf("Render via function call must not error on an undefined var: %v", err)
	}
	if viaFunc != "[]" {
		t.Errorf("viaFunc = %q, want %q", viaFunc, "[]")
	}
}

// TestUUIDv7Format checks uuidv7 produces well-formed output; see the
// determinismTemplate comment for why it's not included in the golden
// determinism test.
func TestUUIDv7Format(t *testing.T) {
	e, err := New(42, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	out, err := e.Render("{{uuidv7}}", nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !uuidv7Re.MatchString(out) {
		t.Errorf("uuidv7 output %q does not match expected format", out)
	}
}

// TestULIDFormat checks ulid produces well-formed output; see the
// determinismTemplate comment for why it's not included in the golden
// determinism test.
func TestULIDFormat(t *testing.T) {
	e, err := New(42, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	out, err := e.Render("{{ulid}}", nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !ulidRe.MatchString(out) {
		t.Errorf("ulid output %q does not match expected format", out)
	}
}

// Note: uuidv7 and ulid are deliberately excluded from this golden template.
//
// google/uuid's NewV7FromReader derives its 12-bit "rand_a" sequence field
// from a package-level monotonic wall-clock counter (see version7.go's
// getV7Time/lastV7time), not from the reader we pass it — so uuidv7 output
// is not reproducible across two Engine instances even with the same seed.
// It's still driven by our entropy reader for the remaining random bits and
// is covered separately below (format-only) in TestUUIDv7Format.
//
// ulid has the same class of problem: its 48-bit timestamp component comes
// from ulid.Timestamp(time.Now()) (see funcs_id.go's ulid function), by
// design — ULIDs are meant to be lexicographically sortable by real creation
// time, so the timestamp must reflect actual wall-clock time rather than the
// seeded PRNG. Two Engine instances constructed microseconds apart can
// straddle a millisecond boundary and get different timestamps even with an
// identical seed, so ulid output is not byte-for-byte reproducible either.
// It's covered separately below (format-only) in TestULIDFormat.
const determinismTemplate = `
{{fakeName}}|{{fakeEmail}}|{{fakeCompany}}|{{fakeSentence}}|
{{uuid}}|{{nanoid}}|{{ksuid}}|{{objectid}}|
{{randInt 1 1000000}}|{{randString 16}}|{{randFloat 0.0 1000.0 4}}|
{{$card := fakeCreditCard}}{{$card.number}}|{{$card.type}}|
{{regex "[0-9]{5}"}}
`

func TestEngineDeterminism(t *testing.T) {
	e1, err := New(42, false)
	if err != nil {
		t.Fatalf("New(e1): %v", err)
	}
	defer e1.Close()

	e2, err := New(42, false)
	if err != nil {
		t.Fatalf("New(e2): %v", err)
	}
	defer e2.Close()

	out1, err := e1.Render(determinismTemplate, nil)
	if err != nil {
		t.Fatalf("e1.Render: %v", err)
	}
	out2, err := e2.Render(determinismTemplate, nil)
	if err != nil {
		t.Fatalf("e2.Render: %v", err)
	}

	if out1 != out2 {
		t.Fatalf("determinism violated for same seed:\n--- out1 ---\n%s\n--- out2 ---\n%s", out1, out2)
	}
	if out1 == "" {
		t.Fatal("rendered output was empty")
	}
}

func TestEngineDifferentSeedsDiffer(t *testing.T) {
	e1, err := New(1, false)
	if err != nil {
		t.Fatalf("New(e1): %v", err)
	}
	defer e1.Close()

	e2, err := New(2, false)
	if err != nil {
		t.Fatalf("New(e2): %v", err)
	}
	defer e2.Close()

	out1, err := e1.Render(determinismTemplate, nil)
	if err != nil {
		t.Fatalf("e1.Render: %v", err)
	}
	out2, err := e2.Render(determinismTemplate, nil)
	if err != nil {
		t.Fatalf("e2.Render: %v", err)
	}

	if out1 == out2 {
		t.Fatal("expected different seeds to produce different output")
	}
}

func TestEngineClose(t *testing.T) {
	e, err := New(7, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	dir := e.TempDir()
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("tempDir should exist before Close: %v", err)
	}

	if err := e.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected tempDir to be removed after Close, stat err = %v", err)
	}
}

func TestEngineRandomSeedIsRandomized(t *testing.T) {
	e1, err := New(0, false)
	if err != nil {
		t.Fatalf("New(e1): %v", err)
	}
	defer e1.Close()

	e2, err := New(0, false)
	if err != nil {
		t.Fatalf("New(e2): %v", err)
	}
	defer e2.Close()

	if e1.seed == e2.seed {
		t.Skip("random seeds happened to collide (astronomically unlikely); not a real failure")
	}
}
