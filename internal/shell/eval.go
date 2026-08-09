package shell

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// ErrUnterminatedQuote is returned by Tokenize when a quoted string is not
// closed before the end of the line.
var ErrUnterminatedQuote = errors.New("unterminated quote")

// ExitError is returned by the "exit"/"quit" builtin (internal/shell/builtin)
// to signal that the whole shell session should stop with the given process
// exit code, as distinct from an ordinary command failure. Callers of Eval
// (internal/shell.Run's runLines/runReader/runScriptFile) detect it via
// errors.As and stop their loop, propagating Code as the process exit code.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit requested with code %d", e.Code)
}

const maxAliasExpansions = 16

// ExpandAliases performs first-word alias substitution on line. If the
// first word of line matches a key in aliases, it is replaced by that
// alias's value and the line is re-scanned (the expansion's own first word
// may itself be an alias). Expansion is capped at maxAliasExpansions total
// substitutions; a self-referential expansion (expanding X yields a line
// whose first word is again X) stops after that one substitution rather
// than looping forever.
func ExpandAliases(aliases map[string]string, line string) (string, error) {
	if len(aliases) == 0 {
		return line, nil
	}
	current := line
	seenSelf := make(map[string]bool)
	for i := 0; i < maxAliasExpansions; i++ {
		trimmed := strings.TrimLeft(current, " \t")
		if trimmed == "" {
			return current, nil
		}
		first, rest := splitFirstWord(trimmed)
		val, ok := aliases[first]
		if !ok {
			return current, nil
		}
		expanded := val
		if rest != "" {
			expanded = val + rest
		}
		newFirst, _ := splitFirstWord(strings.TrimLeft(expanded, " \t"))
		if newFirst == first {
			// self-referential: substitute once, then stop.
			return expanded, nil
		}
		if seenSelf[first] {
			return expanded, nil
		}
		seenSelf[first] = true
		current = expanded
	}
	return current, nil
}

// splitFirstWord returns the first whitespace-delimited word of s and the
// remainder of s starting at the whitespace following that word (including
// the leading whitespace), so callers can reconstruct "word"+rest.
func splitFirstWord(s string) (word, rest string) {
	i := strings.IndexAny(s, " \t")
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i:]
}

// templateEscapeSentinel is substituted for the literal sequence `\{{`
// before template rendering, then restored to `{{` afterward, so users can
// escape template expansion for a literal `{{`.
const templateEscapeSentinel = "\x00MAELSINK_LITERAL_OPEN\x00"

// ExpandTemplate renders line through the shell template engine when
// enabled is true. The `\{{` escape sequence is protected from expansion
// and restored to a literal `{{` in the output. When enabled is false, line
// is returned unchanged.
func ExpandTemplate(engine *tmpl.Engine, data map[string]string, line string, enabled bool) (string, error) {
	if !enabled {
		return line, nil
	}
	escaped := strings.ReplaceAll(line, `\{{`, templateEscapeSentinel)
	rendered, err := engine.Render(escaped, data)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(rendered, templateEscapeSentinel, "{{"), nil
}

// Tokenize splits line into whitespace-separated tokens, POSIX-like:
// 'single quotes' preserve everything literally (no escapes recognized
// inside them); "double quotes" preserve spaces, but backslash still
// escapes '"' and '\' inside them; outside of any quoting, backslash
// escapes the following character (including a space, which then does not
// act as a separator). An unterminated quote returns ErrUnterminatedQuote.
func Tokenize(line string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	haveToken := false

	runes := []rune(line)
	i := 0
	for i < len(runes) {
		c := runes[i]
		switch {
		case c == '\'':
			haveToken = true
			j := i + 1
			closed := false
			for j < len(runes) {
				if runes[j] == '\'' {
					closed = true
					break
				}
				j++
			}
			if !closed {
				return nil, ErrUnterminatedQuote
			}
			cur.WriteString(string(runes[i+1 : j]))
			i = j + 1
		case c == '"':
			haveToken = true
			j := i + 1
			closed := false
			for j < len(runes) {
				if runes[j] == '\\' && j+1 < len(runes) && (runes[j+1] == '"' || runes[j+1] == '\\') {
					cur.WriteRune(runes[j+1])
					j += 2
					continue
				}
				if runes[j] == '"' {
					closed = true
					break
				}
				cur.WriteRune(runes[j])
				j++
			}
			if !closed {
				return nil, ErrUnterminatedQuote
			}
			i = j + 1
		case c == '\\':
			haveToken = true
			if i+1 < len(runes) {
				cur.WriteRune(runes[i+1])
				i += 2
			} else {
				// trailing lone backslash: keep it literally
				cur.WriteRune('\\')
				i++
			}
		case c == ' ' || c == '\t':
			if haveToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				haveToken = false
			}
			i++
		default:
			haveToken = true
			cur.WriteRune(c)
			i++
		}
	}
	if haveToken {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}

// Redirection describes an output redirection parsed from a trailing
// `> file` or `>> file` pair of tokens.
type Redirection struct {
	Path   string
	Append bool
}

// SplitRedirection removes a trailing redirection ("> file" or ">> file",
// as two separate tokens) from tokens and returns the remaining command
// tokens plus the parsed Redirection (nil if none present). A bare ">" or
// ">>" not immediately followed by a path token, or appearing anywhere
// other than as the final two tokens, is an error. A "|" token anywhere is
// an error, since pipelines are not supported (SPEC.md §7.5.14).
func SplitRedirection(tokens []string) ([]string, *Redirection, error) {
	for _, t := range tokens {
		if t == "|" {
			return nil, nil, errors.New("pipelines are not supported")
		}
	}
	// Find any ">"/">>" token; only valid as the second-to-last token.
	for idx, t := range tokens {
		if t == ">" || t == ">>" {
			if idx != len(tokens)-2 {
				return nil, nil, fmt.Errorf("malformed redirection: %q must be followed by exactly one path as the last token", t)
			}
			path := tokens[idx+1]
			if path == "" {
				return nil, nil, fmt.Errorf("malformed redirection: missing path after %q", t)
			}
			return tokens[:idx], &Redirection{Path: path, Append: t == ">>"}, nil
		}
	}
	return tokens, nil, nil
}

// Dispatch resolves tokens[0] via reg.Resolve and, if found, calls its
// Run with the raw remaining tokens (tokens[1:]) — Dispatch does not parse
// flags itself; see the Builtin interface's doc comment for that contract.
// An unresolved command is a hard error; it never falls through to a
// system shell.
func Dispatch(ctx context.Context, s *Session, reg *Registry, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	b, ok := reg.Resolve(s.CommandPrefix, tokens[0])
	if !ok {
		return fmt.Errorf("unknown command: %q", tokens[0])
	}
	err := b.Run(ctx, s, tokens[1:])
	if errors.Is(err, pflag.ErrHelp) {
		// Every builtin parses its own flags by calling its own
		// *pflag.FlagSet.Parse(args) (the Builtin interface's contract —
		// see builtin.go). When the user passes -h/--help, pflag itself
		// already prints the flag usage (to the FlagSet's output, which
		// defaults to os.Stderr) and returns this exact sentinel error.
		// Without this check, that sentinel would propagate up through
		// Eval and get printed a second time as a bare "pflag: help
		// requested" line, and the command would also be treated as a
		// failure (non-zero $status) purely because the user asked for
		// help. Neither is desired: swallow it here, once, for every
		// builtin, rather than requiring each of the ~25 fs.Parse call
		// sites across internal/shell/builtin to special-case it.
		return nil
	}
	return err
}

// Eval runs the full evaluation pipeline (SPEC.md §7.5.3) for one logical
// line: alias expansion, template expansion, tokenization, redirection
// splitting, and dispatch. It always updates s.LastStatus and
// s.Vars["status"]/s.Vars["last_error"] before returning, regardless of
// success or failure.
func Eval(ctx context.Context, s *Session, reg *Registry, rawLine string) error {
	err := evalInner(ctx, s, reg, rawLine)
	UpdateStatus(s, err)
	return err
}

func evalInner(ctx context.Context, s *Session, reg *Registry, rawLine string) error {
	expanded, err := ExpandAliases(s.Aliases, rawLine)
	if err != nil {
		return err
	}

	templated, err := ExpandTemplate(s.Tmpl, s.TemplateData(), expanded, s.Cfg.TemplateEnabled)
	if err != nil {
		return err
	}

	tokens, err := Tokenize(templated)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return nil
	}
	// Comment-only lines: first token starts with '#'.
	if strings.HasPrefix(tokens[0], "#") {
		return nil
	}

	cmdTokens, redir, err := SplitRedirection(tokens)
	if err != nil {
		return err
	}
	if len(cmdTokens) == 0 {
		return nil
	}

	if redir != nil {
		f, ferr := openRedirection(redir)
		if ferr != nil {
			return ferr
		}
		defer f.Close()
		orig := s.Out
		s.Out = f
		defer func() { s.Out = orig }()
	}

	return Dispatch(ctx, s, reg, cmdTokens)
}
