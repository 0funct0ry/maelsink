package tmpl

import (
	"fmt"

	"github.com/Masterminds/sprig/v3"
)

// sprigMVPDocs is the curated, email-generation-focused subset of sprig kept
// for M8.6's <100-function budget: string/list/dict utilities and the two
// date/encoding functions most likely to come up composing an email. Each
// entry is looked up by name from the full sprig.FuncMap() so the actual
// behavior always tracks the vendored sprig version — only the curated name
// list and doc text live here.
//
// Dropped: the crypto/hash and TLS cert generator groups, every must*
// error-returning variant, deprecated aliases (date_in_zone, trimall, ...),
// semver, URL parsing, deep-reflection helpers (deepCopy/deepEqual/kindIs/
// typeIs/typeOf/kindOf), and most list/dict/date/flow utilities rarely
// relevant to composing an email (dig, omit, pluck, toDecimal, duration*,
// sortAlpha, reverse, uniq, without, compact, merge, seq/until, floor/ceil/
// mod/float64, coalesce/fail, and more) — see PROMPTS.md's M8.6 entry for
// the full rationale.
func sprigMVPDocs() []struct {
	name, category, args, desc string
} {
	return []struct {
		name, category, args, desc string
	}{
		{"upper", string(CategoryString), "s", "Uppercases s."},
		{"lower", string(CategoryString), "s", "Lowercases s."},
		{"title", string(CategoryString), "s", "Title-cases s (capitalizes each word)."},
		{"trim", string(CategoryString), "s", "Removes leading/trailing whitespace from s."},
		{"trimPrefix", string(CategoryString), "prefix, s", "Removes prefix from s if present."},
		{"trimSuffix", string(CategoryString), "suffix, s", "Removes suffix from s if present."},
		{"trunc", string(CategoryString), "n, s", "Truncates s to n characters (negative n truncates from the left)."},
		{"replace", string(CategoryString), "old, new, s", "Replaces every occurrence of old with new in s."},
		{"contains", string(CategoryString), "substr, s", "Reports whether s contains substr."},
		{"hasPrefix", string(CategoryString), "prefix, s", "Reports whether s starts with prefix."},
		{"hasSuffix", string(CategoryString), "suffix, s", "Reports whether s ends with suffix."},
		{"add", string(CategoryString), "a, b, ...", "Sums its integer arguments."},
		{"join", string(CategoryString), "sep, list", "Joins list's elements with sep."},
		{"split", string(CategoryString), "sep, s", "Splits s on sep, returning a dict of _0, _1, ..."},
		{"default", string(CategoryString), "fallback, value", "Returns value, or fallback if value is empty."},
		{"ternary", string(CategoryString), "truthy, falsy, cond", "Returns truthy if cond is true, else falsy."},
		{"list", string(CategoryString), "a, b, ...", "Builds a list from its arguments."},
		{"dict", string(CategoryString), "k1, v1, k2, v2, ...", "Builds a dict (map[string]any) from alternating keys/values."},
		{"now", string(CategoryDate), "", "The current time.Time."},
		{"date", string(CategoryDate), "fmt, t", "Formats t using a reference-time layout string."},
		{"b64enc", string(CategoryEncoding), "s", "Base64-encodes s."},
		{"b64dec", string(CategoryEncoding), "s", "Base64-decodes s."},
	}
}

// stringDocs, dateDocs, and encodingDocs each return their category's slice
// of sprigMVPDocs, resolving Fn by name from sprig's real FuncMap.
func (e *Engine) stringDocs() []FuncDoc   { return sprigCategoryDocs(CategoryString) }
func (e *Engine) dateDocs() []FuncDoc     { return sprigCategoryDocs(CategoryDate) }
func (e *Engine) encodingDocs() []FuncDoc { return sprigCategoryDocs(CategoryEncoding) }

func sprigCategoryDocs(cat Category) []FuncDoc {
	fm := sprig.FuncMap()
	var out []FuncDoc
	for _, d := range sprigMVPDocs() {
		if d.category != string(cat) {
			continue
		}
		fn, ok := fm[d.name]
		if !ok {
			panic(fmt.Sprintf("tmpl: sprig function %q not found in sprig.FuncMap()", d.name))
		}
		out = append(out, FuncDoc{Name: d.name, Category: cat, Args: d.args, Description: d.desc, Fn: fn})
	}
	return out
}

// unsafeSprigDocs documents env/expandenv/getHostByName — sprig functions
// that touch the host environment or network, registered only when the
// Engine was constructed with unsafeFuncs=true (--template-unsafe-funcs).
func (e *Engine) unsafeSprigDocs() []FuncDoc {
	fm := sprig.FuncMap()
	return []FuncDoc{
		{Name: "env", Category: CategoryString, Args: "name", Returns: "string",
			Description: "The value of environment variable `name`. Removed by default; restored only under --template-unsafe-funcs.", Fn: fm["env"]},
		{Name: "expandenv", Category: CategoryString, Args: "s", Returns: "string",
			Description: "Expands $VAR references in s from the environment. Removed by default; restored only under --template-unsafe-funcs.", Fn: fm["expandenv"]},
		{Name: "getHostByName", Category: CategoryString, Args: "host", Returns: "string",
			Description: "Resolves host to an IP address (performs a DNS lookup). Removed by default; restored only under --template-unsafe-funcs.", Fn: fm["getHostByName"]},
	}
}
