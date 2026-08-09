// Package shell implements the interactive/scriptable maelsink shell
// (SPEC.md §7.5): the evaluation pipeline, session state, history, and the
// non-interactive Run() entrypoint. It is a pure client of the existing
// /api/v1 REST surface and SMTP port via internal/cliclient — it must never
// import internal/store, internal/smtp, internal/api, or internal/webui.
package shell

import (
	"context"

	"github.com/spf13/pflag"
)

// Builtin is one shell command (e.g. "list", "send", "exit").
//
// Contract: Dispatch does NOT pre-parse flags on the builtin's behalf — it
// hands Run the raw, unparsed argument slice (tokens[1:] from the command
// line, after alias/template expansion, tokenization, and redirection
// splitting). Each Run implementation owns its own flag parsing: it must
// call its own b.Flags().Parse(args) internally and use the resulting
// Flags().Args() for its positional arguments. This is necessary because
// different builtins need different post-parse positional handling (e.g.
// "list" takes a free-text query, others take none).
//
// Flags() must return a FRESH *pflag.FlagSet on every call, since Run may
// be invoked multiple times over the life of one shell session and a
// pflag.FlagSet is not safely reusable across repeated Parse calls.
type Builtin interface {
	// Name is the canonical command name (e.g. "list").
	Name() string
	// Aliases are additional names that resolve to this builtin (e.g. "ls").
	Aliases() []string
	// Flags returns a fresh FlagSet for this builtin. Called by Run
	// implementations internally; Dispatch does not call it.
	Flags() *pflag.FlagSet
	// Run executes the builtin with raw, unparsed arguments.
	Run(ctx context.Context, s *Session, args []string) error
}

// Registry resolves command tokens (honoring an optional command prefix) to
// registered Builtins, including by alias.
type Registry struct {
	order  []Builtin
	byName map[string]Builtin
}

// NewRegistry builds a Registry from the given builtins, indexing each by
// its Name() and every entry in Aliases().
func NewRegistry(builtins ...Builtin) *Registry {
	r := &Registry{
		byName: make(map[string]Builtin),
	}
	for _, b := range builtins {
		r.order = append(r.order, b)
		r.byName[b.Name()] = b
		for _, a := range b.Aliases() {
			r.byName[a] = b
		}
	}
	return r
}

// Resolve looks up token as a builtin name or alias. If commandPrefix is
// non-empty, only the prefixed form (commandPrefix+token) resolves — the
// bare token does not. If commandPrefix is empty, the bare token resolves
// directly.
func (r *Registry) Resolve(commandPrefix, token string) (Builtin, bool) {
	if r == nil {
		return nil, false
	}
	name := token
	if commandPrefix != "" {
		if len(token) <= len(commandPrefix) || token[:len(commandPrefix)] != commandPrefix {
			return nil, false
		}
		name = token[len(commandPrefix):]
	}
	b, ok := r.byName[name]
	return b, ok
}

// Names returns the canonical Name() of every registered builtin, in
// registration order (aliases are not included).
func (r *Registry) Names() []string {
	if r == nil {
		return nil
	}
	names := make([]string, 0, len(r.order))
	for _, b := range r.order {
		names = append(names, b.Name())
	}
	return names
}
