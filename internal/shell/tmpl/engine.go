// Package tmpl provides a text/template engine seeded with a deterministic
// PRNG, backing a rich set of fake-data, ID, and file-generation template
// functions for use by maelsink's interactive shell (M4.1). This package is
// a leaf: it imports only stdlib plus the fake-data/id/doc-generation
// libraries listed in its go.mod requirements, and must never import any
// other internal/... package.
package tmpl

import (
	"crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// Engine holds a seeded PRNG (and everything derived from it: a Faker
// instance and an io.Reader entropy adapter for uuid/ulid) plus a per-session
// temp directory used by file-generating template functions.
type Engine struct {
	seed    int64
	rnd     *mrand.Rand
	entropy io.Reader
	faker   *gofakeit.Faker
	tempDir string
	unsafe  bool

	// registry is this Engine's FuncDoc registry (registry.go), built once
	// in New() since most Fn values are bound methods on this specific
	// seeded Engine.
	registry []FuncDoc

	// mu guards every use of rnd (directly, or indirectly via faker/entropy,
	// both of which wrap the same *mrand.Rand). math/rand.Rand is not safe
	// for concurrent use, but builtins like "send"/"randmsg"/"deluge" render
	// and send messages from a bounded pool of goroutines (--concurrency),
	// so every entry point that touches rnd must serialize through this
	// lock — held for the whole Render() call, since a template's function
	// calls draw from rnd deep inside template execution.
	mu sync.Mutex
}

// randReader adapts a *mrand.Rand into an io.Reader, so it can serve as the
// entropy source for uuid.NewRandomFromReader / ulid.New / uuid.NewV7FromReader.
type randReader struct {
	r *mrand.Rand
}

func (rr randReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(rr.r.Intn(256))
	}
	return len(p), nil
}

// New constructs an Engine. If seed is 0, a real random seed is chosen once
// (from crypto/rand) so each session gets distinct data; otherwise the given
// seed drives every generator deterministically. New also creates a
// session-unique temp directory under os.TempDir() for binary-file template
// functions to write into.
func New(seed int64, unsafeFuncs bool) (*Engine, error) {
	if seed == 0 {
		var buf [8]byte
		if _, err := rand.Read(buf[:]); err != nil {
			return nil, fmt.Errorf("tmpl: generating random seed: %w", err)
		}
		seed = int64(buf[0]) | int64(buf[1])<<8 | int64(buf[2])<<16 | int64(buf[3])<<24 |
			int64(buf[4])<<32 | int64(buf[5])<<40 | int64(buf[6])<<48 | int64(buf[7])<<56
		if seed == 0 {
			seed = time.Now().UnixNano()
		}
	}

	rnd := mrand.New(mrand.NewSource(seed))
	reader := randReader{r: rnd}
	faker := gofakeit.NewFaker(rnd, false)

	tempDir, err := os.MkdirTemp(os.TempDir(), "maelsink-tmpl-*")
	if err != nil {
		return nil, fmt.Errorf("tmpl: creating temp dir: %w", err)
	}

	e := &Engine{
		seed:    seed,
		rnd:     rnd,
		entropy: reader,
		faker:   faker,
		tempDir: tempDir,
		unsafe:  unsafeFuncs,
	}
	e.registry = e.buildRegistry()
	return e, nil
}

// TempDir returns the session-unique directory used for generated files.
func (e *Engine) TempDir() string {
	return e.tempDir
}

// Intn returns a non-negative pseudo-random int in [0,n) from this Engine's
// seeded PRNG — the same source every template function draws from, so
// callers outside the FuncMap (e.g. the "example" builtin picking one of
// several canned templates) stay deterministic under a fixed --seed too.
func (e *Engine) Intn(n int) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.rnd.Intn(n)
}

// Float64 returns a pseudo-random float64 in [0.0,1.0) from this Engine's
// seeded PRNG, for callers needing a uniform draw (e.g. the "steady" jitter
// model in the intmsg builtin) with the same determinism guarantee as Intn.
func (e *Engine) Float64() float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.rnd.Float64()
}

// ExpFloat64 returns a pseudo-random float64 from the exponential
// distribution with rate 1, from this Engine's seeded PRNG — the building
// block for the intmsg builtin's "poisson" interval profile (an exponential
// inter-arrival time with the given mean is the standard Poisson-process
// model).
func (e *Engine) ExpFloat64() float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.rnd.ExpFloat64()
}

// Render parses text as a text/template using this Engine's FuncMap and
// executes it against data, returning the rendered output.
func (e *Engine) Render(text string, data any) (string, error) {
	// "missingkey=zero" (rather than text/template's own default,
	// "missingkey=invalid") makes an undefined {{ .name }} against a
	// map[string]any/map[string]string data value (as session variables
	// always are — internal/shell.Session.TemplateData) render as that
	// map's zero value (empty string) instead of an invalid reflect.Value.
	// Without this, bare {{ .undefined }} happens to print harmlessly as
	// "<no value>", but passing it into ANY function — {{ upper .undefined
	// }}, anything — fails with "invalid value;
	// expected string", since there is no valid conversion from an invalid
	// reflect.Value to a function parameter type. SPEC.md §7.5.6 documents
	// undefined variables as rendering empty; this option is what actually
	// makes that true in every context, not just bare printing.
	text = expandBareRegex(text)
	tpl, err := template.New("tmpl").Option("missingkey=zero").Funcs(e.FuncMap()).Parse(text)
	if err != nil {
		return "", fmt.Errorf("tmpl: parse: %w", err)
	}

	// Held for the whole Execute call: template functions (fake data, ids,
	// binary/file generators) draw from e.rnd/e.faker/e.entropy deep inside
	// execution, none of which are safe for concurrent use.
	e.mu.Lock()
	defer e.mu.Unlock()

	var buf strings.Builder
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("tmpl: execute: %w", err)
	}
	return buf.String(), nil
}

// Close removes the Engine's temp directory and everything in it.
func (e *Engine) Close() error {
	return os.RemoveAll(e.tempDir)
}

// tempFilePath returns a path under the engine's tempDir for the given base
// filename pattern (which may include a single "*" to be replaced by a
// short random-ish component).
func (e *Engine) tempFilePath(ext string) string {
	name := fmt.Sprintf("%d%s", e.rnd.Int63(), ext)
	return filepath.Join(e.tempDir, name)
}

// writeTempFile writes data to a new file under tempDir with mode 0600 and
// returns its path.
func (e *Engine) writeTempFile(ext string, data []byte) (string, error) {
	path := e.tempFilePath(ext)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("tmpl: writing temp file: %w", err)
	}
	return path, nil
}
