package builtin

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// Template implements the "template <expr>" builtin (SPEC.md §7.5.4): the
// debugging tool for §7.5.7's FuncMap.
type Template struct{}

func (Template) Name() string      { return "template" }
func (Template) Aliases() []string { return []string{"tmpl"} }
func (Template) Short() string     { return "Render a template expression and print the result" }

func (Template) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("template", pflag.ContinueOnError)
	fs.StringP("file", "f", "", "read the template from this file instead of the positional arg")
	fs.Bool("funcs", false, "list every registered template function name")
	fs.Int64("seed", 0, "one-shot seed override for this render only")
	return fs
}

func (b Template) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}

	if funcs, _ := fs.GetBool("funcs"); funcs {
		reg := s.Tmpl.Registry()
		names := make([]string, 0, len(reg))
		for _, d := range reg {
			names = append(names, d.Name)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintln(s.Out, n)
		}
		return nil
	}

	file, _ := fs.GetString("file")
	seed, _ := fs.GetInt64("seed")

	var expr string
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		expr = string(data)
	} else {
		pos := fs.Args()
		if len(pos) == 0 {
			return fmt.Errorf("template: requires <expr>, or -f/--file")
		}
		expr = pos[0]
	}

	engine := s.Tmpl
	if seed != 0 {
		tmpEngine, err := tmpl.New(seed, s.Cfg.TemplateUnsafeFuncs)
		if err != nil {
			return err
		}
		defer tmpEngine.Close()
		engine = tmpEngine
	}

	rendered, err := engine.Render(expr, s.TemplateData())
	if err != nil {
		return err
	}
	fmt.Fprintln(s.Out, rendered)
	return nil
}
