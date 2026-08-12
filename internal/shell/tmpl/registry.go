package tmpl

import (
	"fmt"
	"reflect"
	"text/template"
)

// Category groups template functions for the "functions"/"fns"/"funcs"
// builtin (internal/shell/builtin) and "template --funcs" (SPEC.md
// §7.5.7.1). Every registered function carries exactly one Category.
type Category string

const (
	CategoryIdentifiers Category = "identifiers" // uuid, ulid, messageID, ...
	CategoryGenerate    Category = "generate"    // regex, oneOf, randInt, fake*, ...
	CategoryString      Category = "string"      // upper, trim, join, ...
	CategoryDate        Category = "date"        // now, date, ...
	CategoryEncoding    Category = "encoding"    // b64enc, ...
	CategoryEmail       Category = "email"       // attach, fileOf, messageID, mimeWord, ...
	CategoryFiles       Category = "files"       // readFile, glob, basename, ...
	CategoryAnsi        Category = "ansi"        // ansi, red, blue, reset, ...
)

// categoryOrder is the fixed declaration order used to group the
// "functions" builtin's bare listing (SPEC.md §7.5.7.1).
var categoryOrder = []Category{
	CategoryIdentifiers,
	CategoryGenerate,
	CategoryString,
	CategoryDate,
	CategoryEncoding,
	CategoryEmail,
	CategoryFiles,
	CategoryAnsi,
}

// FuncDoc documents one registered template function: the same fields the
// SPEC.md §7.5.7 tables document by hand, plus the actual function value.
// Registry (built per-Engine, since most Fn values are bound methods on a
// specific seeded *Engine) is the single source of truth: FuncMap() derives
// text/template's map from it by reflection, and every introspection
// surface (functions/fns/funcs, template --funcs, registry_test.go's
// coverage test) reads the same slice — no hardcoded switch/case
// duplicating the function list in a second place.
type FuncDoc struct {
	Name        string
	Category    Category
	Args        string
	Returns     string
	Description string
	Fn          any
}

// buildRegistry assembles this Engine's FuncDoc registry from every
// category's *Docs() method. Adding a function is one FuncDoc entry in the
// relevant funcs_*.go file — never a new case in a listing command.
func (e *Engine) buildRegistry() []FuncDoc {
	var reg []FuncDoc
	reg = append(reg, e.idDocs()...)
	reg = append(reg, e.primDocs()...)
	reg = append(reg, e.fakeDocs()...)
	reg = append(reg, e.domainDocs()...)
	reg = append(reg, e.binaryDocs()...)
	reg = append(reg, e.docxDocs()...)
	reg = append(reg, e.stringDocs()...)
	reg = append(reg, e.dateDocs()...)
	reg = append(reg, e.encodingDocs()...)
	reg = append(reg, e.emailDocs()...)
	reg = append(reg, e.filesDocs()...)
	reg = append(reg, ansiDocs()...)
	if e.unsafe {
		reg = append(reg, e.unsafeSprigDocs()...)
	}
	return reg
}

// Registry returns every template function documented and registered for
// this Engine.
func (e *Engine) Registry() []FuncDoc {
	return e.registry
}

// FuncMap returns the text/template.FuncMap derived from this Engine's
// Registry — the map actually handed to text/template.Funcs().
func (e *Engine) FuncMap() template.FuncMap {
	fm := make(template.FuncMap, len(e.registry))
	for _, d := range e.registry {
		if d.Fn == nil {
			panic(fmt.Sprintf("tmpl: registry entry %q has a nil Fn", d.Name))
		}
		if reflect.TypeOf(d.Fn).Kind() != reflect.Func {
			panic(fmt.Sprintf("tmpl: registry entry %q's Fn is not a function", d.Name))
		}
		fm[d.Name] = d.Fn
	}
	return fm
}
