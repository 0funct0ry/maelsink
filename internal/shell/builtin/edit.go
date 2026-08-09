package builtin

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Edit implements the "edit" builtin (SPEC.md §7.5.4). It has two distinct
// modes, both sharing the same $EDITOR resolution chain but with different
// destinations for the result:
//
//   - "edit" (no -f): starts from an empty scratch buffer, edits it, and
//     — per SPEC.md §7.5.9 — LOADS the result into the next prompt's line
//     buffer without executing it (interactive sessions), or prints it to
//     s.Out (batch modes, which have no line buffer to seed). This is the
//     same "load, don't execute" contract as Ctrl-X Ctrl-E.
//   - "edit -f <path>" / "edit --file <path>": edits path DIRECTLY, in
//     place — whatever the editor saves lands in that file. The result is
//     NOT loaded into the prompt buffer and NOT printed, since the file
//     itself already holds the outcome; Run just confirms the path.
type Edit struct{}

func (Edit) Name() string      { return "edit" }
func (Edit) Aliases() []string { return nil }
func (Edit) Short() string     { return "Edit a file in place, or edit scratch text for the prompt" }

func (Edit) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("edit", pflag.ContinueOnError)
	fs.StringP("file", "f", "", "edit this file directly, in place (default: edit scratch text for the prompt buffer)")
	return fs
}

func (b Edit) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	fs.SetOutput(s.Out)
	if err := fs.Parse(args); err != nil {
		return err
	}
	file, _ := fs.GetString("file")

	if file != "" {
		if err := shell.RunEditorOnFile(ctx, s.Cfg.Editor, file); err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "edit: saved %s\n", file)
		return nil
	}

	result, err := shell.RunEditor(ctx, s.Cfg.Editor, "")
	if err != nil {
		return err
	}
	return loadEditResultOrPrint(s, result)
}

// ResolveEditor implements the shell.editor -> $VISUAL -> $EDITOR -> "vi"
// ("notepad" on Windows) fallback chain (SPEC.md §7.5.4/§7.5.9).
func ResolveEditor(cfgEditor string) string {
	return shell.ResolveEditor(cfgEditor)
}
