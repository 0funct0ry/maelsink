package builtin

import (
	"context"
	"strconv"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Exit implements the "exit"/"quit" builtin (SPEC.md §7.5.4). It signals
// session termination via *shell.ExitError, which internal/shell.Run's
// runLines/runReader/runScriptFile detect via errors.As and use to stop
// their loop with the given code, instead of treating it as an ordinary
// command failure.
type Exit struct{}

func (Exit) Name() string      { return "exit" }
func (Exit) Aliases() []string { return []string{"quit"} }
func (Exit) Short() string     { return "Leave the shell" }

func (Exit) Flags() *pflag.FlagSet {
	return pflag.NewFlagSet("exit", pflag.ContinueOnError)
}

func (b Exit) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	code := s.LastStatus
	if pos := fs.Args(); len(pos) > 0 {
		n, err := strconv.Atoi(pos[0])
		if err != nil {
			return err
		}
		code = n
	}
	return &shell.ExitError{Code: code}
}
