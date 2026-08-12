package tmpl

import (
	"math"
	"strings"
)

const defaultRandCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// primDocs documents the primitive randomness template functions, all
// driven by the Engine's seeded math/rand.Rand.
func (e *Engine) primDocs() []FuncDoc {
	return []FuncDoc{
		{Name: "randInt", Category: CategoryGenerate, Args: "min, max int", Returns: "int",
			Description: "Random int in [min,max].", Fn: e.randInt},
		{Name: "randFloat", Category: CategoryGenerate, Args: "min, max float64 [, decimals int]", Returns: "float64",
			Description: "Random float64 in [min,max], optionally rounded to `decimals` places.", Fn: e.randFloat},
		{Name: "randBool", Category: CategoryGenerate, Returns: "bool",
			Description: "Random true/false.", Fn: e.randBool},
		{Name: "randString", Category: CategoryGenerate, Args: "n int [, charset string]", Returns: "string",
			Description: "Random string of length n from charset (default alphanumeric).", Fn: e.randString},
		{Name: "randBytes", Category: CategoryGenerate, Args: "n int", Returns: "[]byte",
			Description: "n random bytes.", Fn: e.randBytes},
		{Name: "pick", Category: CategoryGenerate, Args: "a, b, c, ...", Returns: "any",
			Description: "One of its comma-separated arguments, chosen at random.", Fn: e.pick},
		{Name: "oneOf", Category: CategoryGenerate, Args: `a, b, c, ... | "a,b,c"`, Returns: "any",
			Description: `One of its arguments, chosen at random. Accepts either several comma-separated arguments ({{ oneOf "a" "b" "c" }}) or a single string split on "," ({{ oneOf "a,b,c" }}).`, Fn: e.oneOf},
		{Name: "shuffle", Category: CategoryGenerate, Args: "list", Returns: "list",
			Description: "Returns list with its elements in random order.", Fn: e.shuffle},
	}
}

// randInt returns a pseudo-random int in [min, max] inclusive.
func (e *Engine) randInt(min, max int) int {
	if max <= min {
		return min
	}
	return min + e.rnd.Intn(max-min+1)
}

// randFloat returns a pseudo-random float64 in [min, max], rounded to the
// given number of decimals (default 2).
func (e *Engine) randFloat(min, max float64, decimals ...int) float64 {
	d := 2
	if len(decimals) > 0 && decimals[0] >= 0 {
		d = decimals[0]
	}
	v := min + e.rnd.Float64()*(max-min)
	mult := math.Pow(10, float64(d))
	return math.Round(v*mult) / mult
}

// randBool returns a pseudo-random boolean.
func (e *Engine) randBool() bool {
	return e.rnd.Intn(2) == 1
}

// randString returns a pseudo-random string of length n drawn from charset
// (default alphanumeric).
func (e *Engine) randString(n int, charset ...string) string {
	cs := defaultRandCharset
	if len(charset) > 0 && charset[0] != "" {
		cs = charset[0]
	}
	if n < 0 {
		n = 0
	}
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = cs[e.rnd.Intn(len(cs))]
	}
	return string(buf)
}

// randBytes returns n pseudo-random bytes.
func (e *Engine) randBytes(n int) []byte {
	if n < 0 {
		n = 0
	}
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = byte(e.rnd.Intn(256))
	}
	return buf
}

// pick returns one pseudo-randomly chosen element from list.
func (e *Engine) pick(list ...any) any {
	if len(list) == 0 {
		return nil
	}
	return list[e.rnd.Intn(len(list))]
}

// oneOf returns one pseudo-randomly chosen element from list, except when
// called with exactly one string argument containing a comma — then that
// string is split on "," and one part is chosen instead, so
// {{ oneOf "a,b,c" }} works without needing per-value quoting.
func (e *Engine) oneOf(list ...any) any {
	if len(list) == 1 {
		if s, ok := list[0].(string); ok && strings.Contains(s, ",") {
			parts := strings.Split(s, ",")
			return parts[e.rnd.Intn(len(parts))]
		}
	}
	return e.pick(list...)
}

// shuffle returns a pseudo-randomly shuffled copy of list.
func (e *Engine) shuffle(list []any) []any {
	out := make([]any, len(list))
	copy(out, list)
	e.rnd.Shuffle(len(out), func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}
