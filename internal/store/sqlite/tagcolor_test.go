package sqlite

import "testing"

// TestHashTagColor_MatchesFrontend asserts hashTagColor is bit-for-bit
// identical to web/src/lib/tagColor.ts's hashString+PALETTE lookup for a
// handful of known tag names, computed by hand-running the same FNV-1a-ish
// hash mod 8 against tagColor.ts's PALETTE order (indigo, emerald, amber,
// rose, cyan, fuchsia, lime, orange).
func TestHashTagColor_MatchesFrontend(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"smoke", "lime"},
		{"release", "lime"},
		{"regression", "indigo"},
		{"", "fuchsia"},
	}
	for _, c := range cases {
		got := hashTagColor(c.name)
		if got != c.want {
			t.Errorf("hashTagColor(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}
