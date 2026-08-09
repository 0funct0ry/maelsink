package lineedit

import "testing"

func TestTriggerRune(t *testing.T) {
	tests := []struct {
		mode   string
		wantR  rune
		wantOK bool
	}{
		{"space", ' ', true},
		{"tab", '\t', true},
		{"enter", '\r', true},
		{"none", 0, false},
		{"", 0, false},
		{"bogus", 0, false},
	}
	for _, tt := range tests {
		r, ok := triggerRune(tt.mode)
		if r != tt.wantR || ok != tt.wantOK {
			t.Errorf("triggerRune(%q) = (%q, %v), want (%q, %v)", tt.mode, r, ok, tt.wantR, tt.wantOK)
		}
	}
}

func TestMatchAbbr(t *testing.T) {
	abbrs := map[string]string{
		"l":  "list",
		"ls": "list --limit 5",
	}

	tests := []struct {
		word    string
		wantExp string
		wantOK  bool
	}{
		{"l", "list", true},
		{"ls", "list --limit 5", true},
		{"list", "", false}, // no partial/reverse matches
		{"", "", false},
		{"LS", "", false}, // exact match only, case sensitive
	}

	for _, tt := range tests {
		exp, ok := matchAbbr(tt.word, abbrs)
		if exp != tt.wantExp || ok != tt.wantOK {
			t.Errorf("matchAbbr(%q) = (%q, %v), want (%q, %v)", tt.word, exp, ok, tt.wantExp, tt.wantOK)
		}
	}
}

func TestWordBeforeCursor(t *testing.T) {
	tests := []struct {
		line      string
		pos       int
		wantWord  string
		wantStart int
	}{
		{"list --limit", 4, "list", 0},
		{"list --limit", 13, "--limit", 5},
		{"", 0, "", 0},
		{"foo bar", 7, "bar", 4},
		{"foo bar", 3, "foo", 0},
	}
	for _, tt := range tests {
		word, start := wordBeforeCursor([]rune(tt.line), tt.pos)
		if word != tt.wantWord || start != tt.wantStart {
			t.Errorf("wordBeforeCursor(%q, %d) = (%q, %d), want (%q, %d)",
				tt.line, tt.pos, word, start, tt.wantWord, tt.wantStart)
		}
	}
}

func TestIsFirstWord(t *testing.T) {
	if !isFirstWord([]rune("list"), 0) {
		t.Error("expected word at start of line to be first word")
	}
	if isFirstWord([]rune("list --limit"), 5) {
		t.Error("expected word after another token to not be first word")
	}
	if !isFirstWord([]rune("   list"), 3) {
		t.Error("expected leading whitespace to still count as first word")
	}
}
