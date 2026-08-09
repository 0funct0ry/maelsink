package tmpl

import (
	"sort"
	"testing"
)

// TestDocs_CoversEveryFuncMapEntry guards against the "functions" builtin
// (internal/shell/builtin) falling back to "Help text unavailable." for any
// real template function — every key in FuncMap() (unsafe funcs included,
// since a user running with --template-unsafe-funcs deserves docs too)
// must have a Docs() entry with a non-empty description.
func TestDocs_CoversEveryFuncMapEntry(t *testing.T) {
	e, err := New(1, true) // unsafe=true so env/expandenv/getHostByName are included too
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	docs := Docs()
	var missing []string
	for name := range e.FuncMap() {
		d, ok := docs[name]
		if !ok || d.Description == "" {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%d FuncMap entries have no Docs() entry: %v", len(missing), missing)
	}
}

// TestDocs_NoStaleEntries guards the inverse: every Docs() key should
// correspond to a real FuncMap function, so "functions <name>" detail help
// never describes something that isn't actually callable.
func TestDocs_NoStaleEntries(t *testing.T) {
	e, err := New(1, true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	fm := e.FuncMap()
	var stale []string
	for name := range Docs() {
		if _, ok := fm[name]; !ok {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("%d Docs() entries don't correspond to any real FuncMap function: %v", len(stale), stale)
	}
}
