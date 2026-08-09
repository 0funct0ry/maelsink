package builtin

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Help implements the "help [command]" builtin (SPEC.md §7.5.4). Builtins
// may optionally implement Describable to contribute a one-line
// description to the no-args summary listing; those that don't render with
// an empty description rather than requiring every file to implement it.
type Help struct {
	Registry *shell.Registry
}

func (Help) Name() string      { return "help" }
func (Help) Aliases() []string { return []string{"?"} }
func (Help) Short() string     { return "List builtins, or show one builtin's flag help" }

func (Help) Flags() *pflag.FlagSet {
	return pflag.NewFlagSet("help", pflag.ContinueOnError)
}

func (b Help) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	reg := b.Registry
	if reg == nil {
		reg = s.Registry
	}
	if reg == nil {
		return fmt.Errorf("help: no command registry attached")
	}

	pos := fs.Args()
	if len(pos) > 0 {
		bi, ok := reg.Resolve("", pos[0])
		if !ok {
			return fmt.Errorf("help: unknown command %q", pos[0])
		}
		fmt.Fprintf(s.Out, "%s\n", bi.Name())
		if d, ok := bi.(Describable); ok && d.Short() != "" {
			fmt.Fprintf(s.Out, "  %s\n", d.Short())
		}
		fmt.Fprint(s.Out, bi.Flags().FlagUsages())
		return nil
	}

	for _, name := range reg.Names() {
		bi, _ := reg.Resolve("", name)
		desc := ""
		if d, ok := bi.(Describable); ok {
			desc = d.Short()
		}
		fmt.Fprintf(s.Out, "%-14s %s\n", name, desc)
	}
	return nil
}
