package tmpl

// ansiDocs documents ANSI escape-code helper functions, primarily for use
// in shell.prompt templates (SPEC.md §7.5.10's "colors" requirement) but
// available to any template. Every color/style function here is niladic and
// returns the bare SGR escape sequence (not a wrapped string) — the
// intended usage is literal placement in template source, e.g.
// `{{ blue }}warning{{ reset }}`, mirroring how ANSI codes are normally
// written by hand. They are plain string constants with no awareness of
// shell.color/NO_COLOR — same as bash's \[\e[...\]m prompt escapes, the
// caller decides whether to use them at all (SPEC.md §7.5.10 still gates
// the SHELL'S OWN chrome — errors, table borders, etc. — via
// shell.color/NO_COLOR through lineedit.ResolveColor; a user who opts into
// {{ blue }} in their own prompt template is making an explicit choice that
// bypasses that gate, same as raw ANSI codes in a bash PS1 would).
func ansiDocs() []FuncDoc {
	return []FuncDoc{
		{Name: "ansi", Category: CategoryAnsi, Args: "code, text", Returns: "string",
			Description: `Wraps text in the given SGR escape code(s) (e.g. "1;32"), resetting after. For most use, prefer the bare color/style functions below with an explicit {{ reset }}.`, Fn: ansiWrap},
		{Name: "reset", Category: CategoryAnsi, Returns: "string",
			Description: "The bare ANSI reset sequence — pair with any color/style function below, e.g. {{ blue }}text{{ reset }}.", Fn: func() string { return ansiReset }},
		{Name: "bold", Category: CategoryAnsi, Returns: "string",
			Description: "The bare ANSI bold sequence.", Fn: func() string { return ansiCode("1") }},
		{Name: "dim", Category: CategoryAnsi, Returns: "string",
			Description: "The bare ANSI dim sequence.", Fn: func() string { return ansiCode("2") }},
		{Name: "red", Category: CategoryAnsi, Returns: "string",
			Description: "The bare ANSI red foreground sequence.", Fn: func() string { return ansiCode("31") }},
		{Name: "green", Category: CategoryAnsi, Returns: "string",
			Description: "The bare ANSI green foreground sequence.", Fn: func() string { return ansiCode("32") }},
		{Name: "yellow", Category: CategoryAnsi, Returns: "string",
			Description: "The bare ANSI yellow foreground sequence.", Fn: func() string { return ansiCode("33") }},
		{Name: "blue", Category: CategoryAnsi, Returns: "string",
			Description: "The bare ANSI blue foreground sequence.", Fn: func() string { return ansiCode("34") }},
		{Name: "magenta", Category: CategoryAnsi, Returns: "string",
			Description: "The bare ANSI magenta foreground sequence.", Fn: func() string { return ansiCode("35") }},
		{Name: "cyan", Category: CategoryAnsi, Returns: "string",
			Description: "The bare ANSI cyan foreground sequence.", Fn: func() string { return ansiCode("36") }},
		{Name: "white", Category: CategoryAnsi, Returns: "string",
			Description: "The bare ANSI white foreground sequence.", Fn: func() string { return ansiCode("37") }},
	}
}

const ansiReset = "\x1b[0m"

// ansiCode returns the bare SGR escape sequence for code (e.g. "31" for
// red), with no trailing reset — callers place {{ reset }} themselves.
func ansiCode(code string) string {
	return "\x1b[" + code + "m"
}

// ansiWrap wraps s in the given SGR code(s) (e.g. "31" for red, "1;32" for
// bold green), resetting afterward. code is inserted verbatim between
// "\x1b[" and "m", so multiple codes can be combined with ";".
func ansiWrap(code, s string) string {
	return ansiCode(code) + s + ansiReset
}
