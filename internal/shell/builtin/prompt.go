package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// defaultPromptTemplate mirrors config.Defaults().Shell.Prompt
// (internal/config/config.go) — duplicated as a literal here (rather than
// importing internal/config just for this constant, which builtin already
// depends on transitively via shell.Session.Cfg) so "prompt --reset" has
// something to reset to. SPEC.md §7.5.10: the {{ if not .connected }}
// clause makes an unreachable server visible without a command having to
// fail first.
const defaultPromptTemplate = "maelsink{{ if not .connected }} (offline){{ end }}> "

// Prompt implements the "prompt [template]" builtin: a friendlier,
// live-previewing way to inspect/change shell.prompt for the rest of the
// session than "config set prompt <value>" (which still works identically,
// since both ultimately just assign s.Cfg.Prompt — read live by
// internal/shell's renderPrompt on every interactive read, SPEC.md
// §7.5.10). The prompt template has full access to session variables
// (e.g. {{ .connected }}) and every template function, including the
// ansi*/color helpers in internal/shell/tmpl/funcs_ansi.go, so a colored,
// variable-driven prompt is just a template like any other.
type Prompt struct{}

func (Prompt) Name() string      { return "prompt" }
func (Prompt) Aliases() []string { return nil }
func (Prompt) Short() string     { return "Show or set shell.prompt (template + variables + colors)" }

func (Prompt) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("prompt", pflag.ContinueOnError)
	fs.Bool("reset", false, "restore the built-in default prompt")
	return fs
}

func (b Prompt) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	fs.SetOutput(s.Out)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if reset, _ := fs.GetBool("reset"); reset {
		s.Cfg.Prompt = defaultPromptTemplate
	} else if pos := fs.Args(); len(pos) > 0 {
		s.Cfg.Prompt = strings.Join(pos, " ")
	}

	fmt.Fprintf(s.Out, "template: %s\n", s.Cfg.Prompt)
	rendered, err := s.Tmpl.Render(s.Cfg.Prompt, s.TemplateData())
	if err != nil {
		fmt.Fprintf(s.Out, "preview:  <template error: %s>\n", err)
		return nil
	}
	fmt.Fprintf(s.Out, "preview:  %s\n", rendered)
	return nil
}
