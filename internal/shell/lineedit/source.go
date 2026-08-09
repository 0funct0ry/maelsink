// Package lineedit wraps github.com/ergochat/readline to provide the
// maelsink interactive shell's line editing experience: history recall,
// tab completion, abbreviation expansion, and a couple of key bindings
// that differ from readline's emacs-style defaults (§7.5.9 of SPEC.md).
//
// This package is a leaf with respect to internal/shell: it must never
// import internal/shell, since internal/shell will import lineedit to
// power its interactive REPL. Decoupling is achieved via the
// CompletionSource interface below, which internal/shell adapts its
// *Session/*Registry types to.
package lineedit

import "context"

// CompletionSource supplies the data a completer needs without lineedit
// having to know anything about internal/shell's Session, Registry, or API
// client types.
type CompletionSource interface {
	// BuiltinNames returns every builtin/alias/abbreviation name completable
	// as the first word of a line.
	BuiltinNames() []string

	// FlagsFor returns the long and short flag spellings (e.g. "--limit",
	// "-n") accepted by the named builtin. Returns nil for an unknown name.
	FlagsFor(builtinName string) []string

	// VarNames returns session and reserved variable names, without the
	// leading "$".
	VarNames() []string

	// RecentMessageIDs returns a best-effort list of recently seen message
	// IDs for <id>-position completion. Implementers are responsible for
	// their own ~2s TTL caching (see IDCache) and must return quickly and
	// quietly (nil, not an error) when offline.
	RecentMessageIDs(ctx context.Context) []string

	// AttachmentIDs returns attachment IDs/indices for the message named
	// msgID, for [attId]-position completion. Same offline/latency
	// contract as RecentMessageIDs.
	AttachmentIDs(ctx context.Context, msgID string) []string

	// TemplateFuncNames returns the names available inside {{ }} template
	// expressions (§7.5.7).
	TemplateFuncNames() []string
}
