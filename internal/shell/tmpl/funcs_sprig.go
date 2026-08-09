package tmpl

import (
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// unsafeSprigFuncs lists sprig functions that touch the host environment or
// network and are therefore excluded unless the Engine was constructed with
// unsafeFuncs=true.
var unsafeSprigFuncs = []string{"env", "expandenv", "getHostByName"}

// sprigFuncMap returns sprig's FuncMap with the unsafe subset removed unless
// e.unsafe is true.
func (e *Engine) sprigFuncMap() template.FuncMap {
	fm := sprig.FuncMap()
	if e.unsafe {
		return fm
	}
	for _, name := range unsafeSprigFuncs {
		delete(fm, name)
	}
	return fm
}
