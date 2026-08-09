package builtin

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Unset implements the "unset <k>..." builtin (SPEC.md §7.5.4).
type Unset struct{}

func (Unset) Name() string      { return "unset" }
func (Unset) Aliases() []string { return nil }
func (Unset) Short() string     { return "Remove session variables" }

func (Unset) Flags() *pflag.FlagSet {
	return pflag.NewFlagSet("unset", pflag.ContinueOnError)
}

func (b Unset) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	keys := fs.Args()
	if len(keys) == 0 {
		return fmt.Errorf("unset: requires at least one <k>")
	}
	for _, k := range keys {
		delete(s.Vars, k)
		delete(s.Vars, globalMarkerPrefix+k)
	}
	return nil
}
