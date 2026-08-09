package tmpl

import "text/template"

// ansiFuncMap returns ANSI escape-code helper functions, primarily for use
// in shell.prompt templates (SPEC.md §7.5.10's "colors" requirement) but
// available to any template. They are plain string wrappers with no
// awareness of shell.color/NO_COLOR — same as bash's \[\e[...\]m prompt
// escapes, the caller decides whether to use them at all (SPEC.md §7.5.10
// still gates the SHELL'S OWN chrome — errors, table borders, etc. — via
// shell.color/NO_COLOR through lineedit.ResolveColor; a user who opts into
// ansiRed in their own prompt template is making an explicit choice that
// bypasses that gate, same as raw ANSI codes in a bash PS1 would).
func ansiFuncMap() template.FuncMap {
	return template.FuncMap{
		"ansi":        ansiWrap,
		"ansiReset":   func() string { return ansiReset },
		"ansiBold":    func(s string) string { return ansiWrap("1", s) },
		"ansiDim":     func(s string) string { return ansiWrap("2", s) },
		"ansiRed":     func(s string) string { return ansiWrap("31", s) },
		"ansiGreen":   func(s string) string { return ansiWrap("32", s) },
		"ansiYellow":  func(s string) string { return ansiWrap("33", s) },
		"ansiBlue":    func(s string) string { return ansiWrap("34", s) },
		"ansiMagenta": func(s string) string { return ansiWrap("35", s) },
		"ansiCyan":    func(s string) string { return ansiWrap("36", s) },
		"ansiWhite":   func(s string) string { return ansiWrap("37", s) },
	}
}

const ansiReset = "\x1b[0m"

// ansiWrap wraps s in the given SGR code(s) (e.g. "31" for red, "1;32" for
// bold green), resetting afterward. code is inserted verbatim between
// "\x1b[" and "m", so multiple codes can be combined with ";".
func ansiWrap(code, s string) string {
	return "\x1b[" + code + "m" + s + ansiReset
}
