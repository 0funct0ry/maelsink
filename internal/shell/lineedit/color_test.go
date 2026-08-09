package lineedit

import "testing"

func TestResolveColor(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		noColor  string
		isTTY    bool
		expected bool
	}{
		{"always regardless of tty", "always", "", false, true},
		{"always regardless of NO_COLOR", "always", "1", false, true},
		{"never regardless of tty", "never", "", true, false},
		{"auto tty no NO_COLOR", "auto", "", true, true},
		{"auto non-tty", "auto", "", false, false},
		{"auto tty with NO_COLOR", "auto", "1", true, false},
		{"empty mode behaves like auto, tty", "", "", true, true},
		{"empty mode behaves like auto, non-tty", "", "", false, false},
		{"unrecognized mode behaves like auto", "bogus", "", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.noColor != "" {
				t.Setenv("NO_COLOR", tt.noColor)
			} else {
				t.Setenv("NO_COLOR", "")
				// Setenv with "" still sets the var to an empty string,
				// which os.Getenv treats identically to unset for our
				// purposes (ResolveColor checks != "").
			}

			old := isTerminalFunc
			isTerminalFunc = func() bool { return tt.isTTY }
			defer func() { isTerminalFunc = old }()

			got := ResolveColor(tt.mode)
			if got != tt.expected {
				t.Errorf("ResolveColor(%q) with NO_COLOR=%q isTTY=%v = %v, want %v",
					tt.mode, tt.noColor, tt.isTTY, got, tt.expected)
			}
		})
	}
}
