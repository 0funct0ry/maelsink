package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Echo implements the "echo" builtin: a quick way to inspect session
// variables and template functions without the ceremony of the "template"
// builtin's -f/--funcs/--seed flags. It does no template rendering of its
// own — Eval's pipeline (SPEC.md §7.5.3) already expands {{ }} expressions
// against the session's variables and the full tmpl.Engine FuncMap on the
// WHOLE LINE before tokenization, so by the time Echo.Run sees its args
// they are already rendered, exactly like every other builtin's arguments
// (e.g. `send --subject "{{ fSubject }}"`). `echo {{ .connected }} {{
// fEmail }}` therefore "just works" with no special-casing here.
type Echo struct{}

func (Echo) Name() string      { return "echo" }
func (Echo) Aliases() []string { return nil }
func (Echo) Short() string {
	return "Print args (after template expansion) — inspect vars/funcs quickly"
}

func (Echo) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("echo", pflag.ContinueOnError)
	fs.BoolP("no-newline", "n", false, "don't print the trailing newline")
	return fs
}

func (b Echo) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	fs.SetOutput(s.Out)
	if err := fs.Parse(args); err != nil {
		return err
	}
	noNewline, _ := fs.GetBool("no-newline")

	text := strings.Join(fs.Args(), " ")
	if noNewline {
		_, err := fmt.Fprint(s.Out, text)
		return err
	}
	_, err := fmt.Fprintln(s.Out, text)
	return err
}
