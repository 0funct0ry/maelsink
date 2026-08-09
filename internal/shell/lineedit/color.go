package lineedit

import (
	"os"

	"github.com/mattn/go-isatty"
)

// isTerminalFunc is a seam so tests can fake "is stdout a real TTY" without
// depending on the actual process's stdout.
var isTerminalFunc = func() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// ResolveColor implements the color-gating rule from SPEC.md §7.5.10:
// "auto" (the default) enables color only when stdout is a TTY and NO_COLOR
// is unset; "always" and "never" force the decision regardless of terminal
// or environment.
func ResolveColor(mode string) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default: // "auto", "", or anything unrecognized
		if os.Getenv("NO_COLOR") != "" {
			return false
		}
		return isTerminalFunc()
	}
}
