package tmpl

import (
	"fmt"
	"math"
	"text/template"
)

const defaultRandCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// primFuncMap returns primitive randomness template functions, all driven
// by the Engine's seeded math/rand.Rand.
func (e *Engine) primFuncMap() template.FuncMap {
	return template.FuncMap{
		"randInt":    e.randInt,
		"randFloat":  e.randFloat,
		"randBool":   e.randBool,
		"randString": e.randString,
		"randBytes":  e.randBytes,
		"pick":       e.pick,
		"shuffle":    e.shuffle,
		"weighted":   e.weighted,
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

// shuffle returns a pseudo-randomly shuffled copy of list.
func (e *Engine) shuffle(list []any) []any {
	out := make([]any, len(list))
	copy(out, list)
	e.rnd.Shuffle(len(out), func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}

// weighted picks a key from dict pseudo-randomly, weighted by each value
// interpreted as a number.
func (e *Engine) weighted(dict map[string]any) (any, error) {
	if len(dict) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(dict))
	weights := make([]float64, 0, len(dict))
	var total float64

	for k, v := range dict {
		w, err := toFloat(v)
		if err != nil {
			return nil, fmt.Errorf("tmpl: weighted: %w", err)
		}
		if w < 0 {
			w = 0
		}
		keys = append(keys, k)
		weights = append(weights, w)
		total += w
	}

	if total <= 0 {
		return keys[e.rnd.Intn(len(keys))], nil
	}

	r := e.rnd.Float64() * total
	var cum float64
	for i, w := range weights {
		cum += w
		if r <= cum {
			return keys[i], nil
		}
	}
	return keys[len(keys)-1], nil
}

func toFloat(v any) (float64, error) {
	switch n := v.(type) {
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("unsupported weight type %T", v)
	}
}
