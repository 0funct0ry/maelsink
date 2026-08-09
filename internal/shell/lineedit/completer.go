package lineedit

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// idFetchTimeout bounds how long the completer will wait on
// RecentMessageIDs/AttachmentIDs. readline's AutoCompleter.Do does not hand
// us a context, so we build one internally.
const idFetchTimeout = 300 * time.Millisecond

// Completer implements github.com/ergochat/readline's AutoCompleter
// interface (Do([]rune, int) ([][]rune, int)), dispatching on cursor
// context per SPEC.md §7.5.10's completion table.
type Completer struct {
	Source   CompletionSource
	msgCache *IDCache
	attCache map[string]*IDCache
}

// NewCompleter builds a Completer backed by src. Recent-message-ID and
// attachment-ID lookups are cached per SPEC.md §7.5.10's ~2s TTL.
func NewCompleter(src CompletionSource) *Completer {
	return &Completer{
		Source:   src,
		msgCache: NewIDCache(2 * time.Second),
		attCache: make(map[string]*IDCache),
	}
}

// Do implements readline.AutoCompleter. It converts our word-oriented
// Complete result into the (suffixes, sharedLength) shape readline expects.
func (c *Completer) Do(line []rune, pos int) ([][]rune, int) {
	candidates, wordStart := c.Complete(line, pos)
	if len(candidates) == 0 {
		return nil, 0
	}
	word := string(line[wordStart:pos])
	out := make([][]rune, 0, len(candidates))
	for _, cand := range candidates {
		if !strings.HasPrefix(cand, word) {
			continue
		}
		out = append(out, []rune(cand[len(word):]))
	}
	return out, pos - wordStart
}

// Complete is the pure, directly-testable dispatch logic: given the whole
// line and the cursor offset (in runes), it returns full candidate words
// (not suffixes) and the rune offset where the current word begins.
func (c *Completer) Complete(line []rune, pos int) (candidates []string, wordStart int) {
	if pos < 0 || pos > len(line) {
		pos = len(line)
	}
	head := string(line[:pos])

	wordStart = lastWordStart(head)
	word := head[wordStart:]

	tokens := strings.Fields(head[:wordStart])

	switch {
	case strings.HasPrefix(word, "$"):
		return prefixFilter(varCandidates(c.Source.VarNames()), word), wordStart

	case insideTemplate(head):
		if strings.HasPrefix(word, ".") {
			return prefixFilter(dotVarCandidates(c.Source.VarNames()), word), wordStart
		}
		return prefixFilter(c.Source.TemplateFuncNames(), word), wordStart

	case len(tokens) == 0:
		return prefixFilter(c.Source.BuiltinNames(), word), wordStart

	default:
		prev := tokens[len(tokens)-1]

		if looksLikePathFlag(prev) {
			return c.pathCandidates(word), wordStart
		}

		if !strings.HasPrefix(word, "-") {
			if cands := c.idPositionCandidates(tokens); cands != nil {
				return prefixFilter(cands, word), wordStart
			}
		}

		if isBuiltin(c.Source, prev) {
			return prefixFilter(c.Source.FlagsFor(prev), word), wordStart
		}

		return nil, wordStart
	}
}

// lastWordStart returns the rune index (within head) where the current word
// begins, splitting on whitespace. This is a simple heuristic — it does not
// understand quoting — which is acceptable for tab completion.
func lastWordStart(head string) int {
	r := []rune(head)
	i := len(r)
	for i > 0 && !isSpaceRune(r[i-1]) {
		i--
	}
	return i
}

func isSpaceRune(r rune) bool {
	return r == ' ' || r == '\t'
}

func varCandidates(names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = "$" + n
	}
	return out
}

// dotVarCandidates prefixes each variable name with "." for completion
// inside a {{ }} template expression after a bare dot (e.g. ".Foo").
func dotVarCandidates(names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = "." + n
	}
	return out
}

func isBuiltin(src CompletionSource, name string) bool {
	for _, b := range src.BuiltinNames() {
		if b == name {
			return true
		}
	}
	return false
}

// looksLikePathFlag is the "simple heuristic" called for by the plan: the
// previous token looks like a flag (starts with "-") and its name suggests
// a filesystem path (contains file/path/out/dir/attach).
func looksLikePathFlag(prevToken string) bool {
	if !strings.HasPrefix(prevToken, "-") {
		return false
	}
	lower := strings.ToLower(prevToken)
	for _, hint := range []string{"file", "path", "out", "dir", "attach"} {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

// idPositionBuiltins lists the builtins whose trailing positional args are
// message/attachment IDs, per SPEC.md §7.5.10's <id>/[attId] rows.
var idPositionBuiltins = map[string]bool{
	"show":       true,
	"delete":     true,
	"attachment": true,
}

// idPositionCandidates returns best-effort ID completions when the current
// word looks like it's in an <id> or [attId] position. tokens is the list
// of already-typed tokens before the current word (tokens[0] is the
// command name).
func (c *Completer) idPositionCandidates(tokens []string) []string {
	if len(tokens) == 0 {
		return nil
	}
	cmd := tokens[0]
	if !idPositionBuiltins[cmd] {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), idFetchTimeout)
	defer cancel()

	if cmd == "attachment" && len(tokens) >= 2 {
		// tokens: ["attachment", "<msgID>", ...] — current word is the
		// attachment position once the message ID token is already present.
		msgID := tokens[1]
		cache := c.attCache[msgID]
		if cache == nil {
			cache = NewIDCache(2 * time.Second)
			c.attCache[msgID] = cache
		}
		return cache.Get(ctx, func(ctx context.Context) []string {
			return c.Source.AttachmentIDs(ctx, msgID)
		})
	}

	// show/delete <id>, or "attachment <msgID>" itself: message-ID position.
	return c.msgCache.Get(ctx, func(ctx context.Context) []string {
		return c.Source.RecentMessageIDs(ctx)
	})
}

// insideTemplate reports whether head (the line up to the cursor) has an
// unclosed "{{" — i.e. the cursor is inside a template expression. This is
// a simple heuristic scan, not a real template parser.
func insideTemplate(head string) bool {
	open := strings.LastIndex(head, "{{")
	if open < 0 {
		return false
	}
	return !strings.Contains(head[open:], "}}")
}

// pathCandidates lists directory entries under filepath.Dir(word), for
// filesystem completion of a path-valued flag.
func (c *Completer) pathCandidates(word string) []string {
	dir := filepath.Dir(word)
	if word == "" {
		dir = "."
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	prefix := dir
	if prefix != "" && !strings.HasSuffix(prefix, string(filepath.Separator)) && dir != "." {
		prefix += string(filepath.Separator)
	} else if dir == "." {
		prefix = ""
	}

	out := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += string(filepath.Separator)
		}
		out = append(out, prefix+name)
	}
	sort.Strings(out)
	return out
}

// prefixFilter returns the subset of candidates that start with prefix,
// sorted for deterministic output.
func prefixFilter(candidates []string, prefix string) []string {
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	sort.Strings(out)
	return out
}
