package builtin

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Delete implements the "delete <id>..." builtin (SPEC.md §7.5.4).
type Delete struct{}

func (Delete) Name() string      { return "delete" }
func (Delete) Aliases() []string { return []string{"rm", "del"} }
func (Delete) Short() string     { return "Delete one or more messages" }

func (Delete) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("delete", pflag.ContinueOnError)
	fs.Bool("all", false, "delete every message (same as clear)")
	fs.BoolP("yes", "y", false, "skip confirmation")
	return fs
}

func (b Delete) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	all, _ := fs.GetBool("all")
	yes, _ := fs.GetBool("yes")

	if all {
		return runClear(ctx, s, yes)
	}

	ids := fs.Args()
	if len(ids) == 0 {
		return fmt.Errorf("delete: requires at least one <id>, or --all")
	}

	var failed int
	for _, id := range ids {
		if err := s.Client.Delete(ctx, id); err != nil {
			fmt.Fprintf(s.Err, "delete %s: %s\n", id, shell.FormatClientError(err))
			failed++
			continue
		}
		fmt.Fprintf(s.Out, "deleted %s\n", id)
	}
	if failed > 0 {
		return fmt.Errorf("delete: %d of %d failed", failed, len(ids))
	}
	return nil
}

// runClear implements the delete-all path shared by "delete --all" and
// "clear": destructive-command safety per SPEC.md §7.5.4 — interactive
// sessions get a confirmation prompt (using Stats() for the count),
// non-interactive sessions hard-error immediately without --yes.
func runClear(ctx context.Context, s *shell.Session, yes bool) error {
	if !yes {
		if !s.Interactive {
			return fmt.Errorf("delete --all requires --yes in non-interactive mode")
		}
		st, err := s.Client.Stats(ctx)
		n := 0
		if err == nil {
			n = st.TotalMessages
		}
		if !confirmPrompt(s, fmt.Sprintf("This will delete %d messages. Continue? [y/N] ", n)) {
			fmt.Fprintln(s.Out, "aborted")
			return nil
		}
	}
	if err := s.Client.Clear(ctx); err != nil {
		return clientError(s, err)
	}
	fmt.Fprintln(s.Out, "all messages deleted")
	return nil
}
