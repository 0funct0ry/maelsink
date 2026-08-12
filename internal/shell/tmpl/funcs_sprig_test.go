package tmpl

import "testing"

var unsafeSprigFuncs = []string{"env", "expandenv", "getHostByName"}

func TestSprigUnsafeFuncsExcludedByDefault(t *testing.T) {
	e, err := New(1, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	fm := e.FuncMap()
	for _, name := range unsafeSprigFuncs {
		if _, ok := fm[name]; ok {
			t.Errorf("expected %q to be absent when unsafe=false", name)
		}
	}
}

func TestSprigUnsafeFuncsIncludedWhenEnabled(t *testing.T) {
	e, err := New(1, true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	fm := e.FuncMap()
	for _, name := range unsafeSprigFuncs {
		if _, ok := fm[name]; !ok {
			t.Errorf("expected %q to be present when unsafe=true", name)
		}
	}
}
