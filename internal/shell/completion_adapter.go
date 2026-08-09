package shell

import (
	"context"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell/lineedit"
)

// sessionCompletionAdapter implements lineedit.CompletionSource by wrapping
// a *Session/*Registry, so internal/shell/lineedit never needs to import
// internal/shell (which would create an import cycle, since internal/shell
// imports lineedit to power its interactive REPL).
type sessionCompletionAdapter struct {
	session *Session
}

// newSessionCompletionAdapter builds a lineedit.CompletionSource backed by s.
func newSessionCompletionAdapter(s *Session) lineedit.CompletionSource {
	return &sessionCompletionAdapter{session: s}
}

// BuiltinNames returns every builtin/alias name resolvable against s's
// registry, plus any session-defined aliases/abbreviations.
func (a *sessionCompletionAdapter) BuiltinNames() []string {
	var names []string
	if a.session.Registry != nil {
		names = append(names, a.session.Registry.Names()...)
	}
	for alias := range a.session.Aliases {
		names = append(names, alias)
	}
	for abbr := range a.session.Abbrs {
		names = append(names, abbr)
	}
	return names
}

// FlagsFor returns the long and short flag spellings accepted by the named
// builtin, or nil if builtinName does not resolve.
func (a *sessionCompletionAdapter) FlagsFor(builtinName string) []string {
	if a.session.Registry == nil {
		return nil
	}
	b, ok := a.session.Registry.Resolve(a.session.CommandPrefix, builtinName)
	if !ok {
		return nil
	}
	var flags []string
	b.Flags().VisitAll(func(f *pflag.Flag) {
		flags = append(flags, "--"+f.Name)
		if f.Shorthand != "" {
			flags = append(flags, "-"+f.Shorthand)
		}
	})
	return flags
}

// VarNames returns session variable names plus reserved vars.
func (a *sessionCompletionAdapter) VarNames() []string {
	names := make([]string, 0, len(a.session.Vars)+1)
	for k := range a.session.Vars {
		names = append(names, k)
	}
	return names
}

// RecentMessageIDs returns a best-effort list of recently seen message IDs,
// silently returning nil on any client error or missing client (offline
// no-op, per lineedit.CompletionSource's contract).
func (a *sessionCompletionAdapter) RecentMessageIDs(ctx context.Context) []string {
	if a.session.Client == nil {
		return nil
	}
	resp, err := a.session.Client.List(ctx, cliclient.ListParams{Limit: 50})
	if err != nil || resp == nil {
		return nil
	}
	ids := make([]string, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		ids = append(ids, m.ID)
	}
	return ids
}

// AttachmentIDs returns attachment IDs for msgID, silently returning nil on
// any client error or missing client.
func (a *sessionCompletionAdapter) AttachmentIDs(ctx context.Context, msgID string) []string {
	if a.session.Client == nil {
		return nil
	}
	detail, err := a.session.Client.Get(ctx, msgID)
	if err != nil || detail == nil {
		return nil
	}
	ids := make([]string, 0, len(detail.Attachments))
	for _, att := range detail.Attachments {
		ids = append(ids, att.ID)
	}
	return ids
}

// TemplateFuncNames returns the names available inside {{ }} template
// expressions.
func (a *sessionCompletionAdapter) TemplateFuncNames() []string {
	if a.session.Tmpl == nil {
		return nil
	}
	fm := a.session.Tmpl.FuncMap()
	names := make([]string, 0, len(fm))
	for name := range fm {
		names = append(names, name)
	}
	return names
}
