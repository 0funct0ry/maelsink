package builtin

import (
	"context"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Clear implements the "clear" builtin: alias-equivalent to "delete --all"
// (SPEC.md §7.5.4).
type Clear struct{}

func (Clear) Name() string      { return "clear" }
func (Clear) Aliases() []string { return nil }
func (Clear) Short() string     { return "Delete every message (alias for delete --all)" }

func (Clear) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("clear", pflag.ContinueOnError)
	fs.BoolP("yes", "y", false, "skip confirmation")
	return fs
}

func (b Clear) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	yes, _ := fs.GetBool("yes")
	return runClear(ctx, s, yes)
}
