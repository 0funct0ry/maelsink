package tmpl

import (
	"strconv"
	"strings"
)

// expandBareRegex rewrites bare-pattern `{{ regex <pattern> }}` template
// actions — where <pattern> is everything up to the next literal "}}",
// taken verbatim rather than parsed as text/template argument syntax — into
// the equivalent quoted call `{{ regex "<pattern>" }}`, so RE2 metacharacters
// like `{2,4}` don't need escaping and the pattern doesn't need surrounding
// quotes. The already-quoted form `{{ regex "pattern" }}` is left untouched
// (detected by the first non-space character after the "regex" keyword being
// a quote), so patterns containing a literal "}}" still have an escape hatch.
func expandBareRegex(text string) string {
	const keyword = "regex"
	var out strings.Builder
	i := 0
	for {
		start := strings.Index(text[i:], "{{")
		if start == -1 {
			out.WriteString(text[i:])
			break
		}
		start += i
		out.WriteString(text[i:start])

		end := strings.Index(text[start:], "}}")
		if end == -1 {
			out.WriteString(text[start:])
			break
		}
		end += start

		action := text[start+2 : end]
		trimmed := strings.TrimLeft(action, " \t")
		if !strings.HasPrefix(trimmed, keyword) {
			out.WriteString(text[start : end+2])
			i = end + 2
			continue
		}
		rest := trimmed[len(keyword):]
		if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
			// Not the "regex" action itself (e.g. a longer identifier that
			// happens to start with "regex").
			out.WriteString(text[start : end+2])
			i = end + 2
			continue
		}
		pattern := strings.TrimLeft(rest, " \t")
		if pattern == "" || pattern[0] == '"' || pattern[0] == '`' {
			// Already quoted (or no argument at all) — leave untouched.
			out.WriteString(text[start : end+2])
			i = end + 2
			continue
		}
		pattern = strings.TrimRight(pattern, " \t")

		out.WriteString("{{ ")
		out.WriteString(keyword)
		out.WriteString(" ")
		out.WriteString(strconv.Quote(pattern))
		out.WriteString(" }}")

		i = end + 2
	}
	return out.String()
}
