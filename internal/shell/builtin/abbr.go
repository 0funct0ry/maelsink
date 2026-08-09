package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Abbr implements the "abbr [name[=value]]" builtin (SPEC.md §7.5.4). See
// Alias's doc comment for the --global minimal-scope note, which applies
// identically here.
type Abbr struct{}

func (Abbr) Name() string      { return "abbr" }
func (Abbr) Aliases() []string { return nil }
func (Abbr) Short() string     { return "Define or list trigger-key abbreviations" }

func (Abbr) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("abbr", pflag.ContinueOnError)
	fs.Bool("global", false, "substitute anywhere on the line, not just the first word")
	fs.Bool("erase", false, "remove the named abbreviation")
	return fs
}

func (b Abbr) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	erase, _ := fs.GetBool("erase")
	pos := fs.Args()

	if len(pos) == 0 {
		for _, k := range sortedKeys(s.Abbrs) {
			fmt.Fprintf(s.Out, "%s=%s\n", k, s.Abbrs[k])
		}
		return nil
	}

	name, value, hasEq := strings.Cut(pos[0], "=")
	if erase {
		delete(s.Abbrs, name)
		return nil
	}
	if !hasEq {
		if v, ok := s.Abbrs[name]; ok {
			fmt.Fprintf(s.Out, "%s=%s\n", name, v)
			return nil
		}
		return fmt.Errorf("abbr: %q is not defined", name)
	}
	s.Abbrs[name] = value
	return nil
}

// Unabbr implements the "unabbr <name>... [--all]" builtin.
type Unabbr struct{}

func (Unabbr) Name() string      { return "unabbr" }
func (Unabbr) Aliases() []string { return nil }
func (Unabbr) Short() string     { return "Remove one or more abbreviations" }

func (Unabbr) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("unabbr", pflag.ContinueOnError)
	fs.Bool("all", false, "remove every abbreviation")
	return fs
}

func (b Unabbr) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	all, _ := fs.GetBool("all")
	if all {
		s.Abbrs = make(map[string]string)
		return nil
	}
	names := fs.Args()
	if len(names) == 0 {
		return fmt.Errorf("unabbr: requires at least one <name>, or --all")
	}
	for _, n := range names {
		delete(s.Abbrs, n)
	}
	return nil
}
