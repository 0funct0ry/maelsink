package lineedit

import "strings"

// triggerRune maps the shell.abbr_trigger_key config value (space|tab|
// enter|none) to the rune readline reports for that keypress. ok is false
// for "none" (disabled) or an unrecognized value.
func triggerRune(mode string) (r rune, ok bool) {
	switch mode {
	case "space":
		return ' ', true
	case "tab":
		return '\t', true
	case "enter":
		return '\r', true // matches readline's CharEnter (13)
	default: // "none" or unrecognized
		return 0, false
	}
}

// matchAbbr is the pure matching logic behind trigger-key abbreviation
// expansion: given the word immediately preceding the cursor and the
// current abbreviation map, it reports whether that word should be
// expanded and, if so, its replacement.
//
// Per SPEC.md §7.5.9, expansion fires only when the word matches an
// abbreviation *exactly*. This package does not distinguish "--global" vs
// first-word-only abbreviations — that policy question requires knowing
// which word position the caller is at relative to the whole line, which
// the plain map[string]string returned by Config.Abbrs cannot express; see
// firstWordOnlyMatch below for the position-aware variant used by editor.go.
func matchAbbr(word string, abbrs map[string]string) (expansion string, ok bool) {
	if word == "" {
		return "", false
	}
	v, found := abbrs[word]
	return v, found
}

// wordBeforeCursor returns the word ending exactly at pos in line, using
// the same whitespace-splitting heuristic as the completer.
func wordBeforeCursor(line []rune, pos int) (word string, start int) {
	if pos < 0 || pos > len(line) {
		pos = len(line)
	}
	i := pos
	for i > 0 && !isSpaceRune(line[i-1]) {
		i--
	}
	return string(line[i:pos]), i
}

// isFirstWord reports whether the word starting at wordStart is the first
// token on the line (i.e. everything before it is blank).
func isFirstWord(line []rune, wordStart int) bool {
	return strings.TrimSpace(string(line[:wordStart])) == ""
}
