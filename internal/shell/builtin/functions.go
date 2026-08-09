package builtin

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// Functions implements the "functions [name]" builtin: lists every
// template function available to {{ }} expressions (SPEC.md §7.5.7), or
// shows detailed help for one. This is the discovery counterpart to the
// "template" builtin's debug-render use case — "template --funcs" already
// lists bare names; "functions" adds one-line summaries and per-function
// detail so a user can find and understand a function without leaving the
// shell.
type Functions struct{}

func (Functions) Name() string      { return "functions" }
func (Functions) Aliases() []string { return []string{"funcs"} }
func (Functions) Short() string     { return "List template functions, or show help for one" }

func (Functions) Flags() *pflag.FlagSet {
	return pflag.NewFlagSet("functions", pflag.ContinueOnError)
}

func (b Functions) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	fs.SetOutput(s.Out)
	if err := fs.Parse(args); err != nil {
		return err
	}
	pos := fs.Args()

	docs := tmpl.Docs()
	names := make([]string, 0, len(s.Tmpl.FuncMap()))
	for name := range s.Tmpl.FuncMap() {
		names = append(names, name)
	}
	sort.Strings(names)

	if len(pos) == 0 {
		for _, name := range names {
			desc := "Help text unavailable."
			if d, ok := docs[name]; ok {
				desc = d.Description
			}
			_, _ = fmt.Fprintf(s.Out, "%-16s %s\n", name, desc)
		}
		return nil
	}

	name := pos[0]
	found := false
	for _, n := range names {
		if n == name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("functions: unknown function %q", name)
	}

	d, ok := docs[name]
	if !ok {
		fmt.Fprintf(s.Out, "%s\n\n  sprig utility function — see https://masterminds.github.io/sprig/ for its signature and behavior.\n", name)
		return nil
	}
	fmt.Fprintf(s.Out, "%s\n\n  {{ %s }}\n\n  %s\n", name, d.Signature, d.Description)
	return nil
}
