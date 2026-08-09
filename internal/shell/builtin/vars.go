package builtin

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Vars implements the "vars" builtin (SPEC.md §7.5.4): lists session
// variables, reserved ones included.
type Vars struct{}

func (Vars) Name() string      { return "vars" }
func (Vars) Aliases() []string { return nil }
func (Vars) Short() string     { return "List session variables" }

func (Vars) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("vars", pflag.ContinueOnError)
	fs.String("format", "table", "output format: table|json|yaml")
	return fs
}

func (b Vars) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	format, _ := fs.GetString("format")
	return printVars(s, format)
}

// printVars renders s.Vars (skipping the internal "__global__:" marker
// convention — see set.go's --global doc comment) per format.
func printVars(s *shell.Session, format string) error {
	clean := make(map[string]string, len(s.Vars))
	for k, v := range s.Vars {
		if isGlobalMarkerKey(k) {
			continue
		}
		clean[k] = v
	}
	return writeFormatted(s.Out, format, clean, func(w io.Writer) error {
		for _, k := range sortedKeys(clean) {
			fmt.Fprintf(w, "%s\t%s\n", k, clean[k])
		}
		return nil
	})
}
