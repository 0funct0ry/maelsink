package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Alias implements the "alias [name[=value]]" builtin (SPEC.md §7.5.4).
//
// --global's "substitute anywhere on the line, not just the first word"
// semantics are a lineedit/eval-pipeline concern (ExpandAliases currently
// only substitutes the first word); this builtin records the flag as a
// marker so a future eval.go enhancement can honor it, but does not itself
// change substitution behavior — a documented minimal-scope gap for this
// phase, matching the plan's guidance for --global on set/abbr too.
type Alias struct{}

func (Alias) Name() string      { return "alias" }
func (Alias) Aliases() []string { return nil }
func (Alias) Short() string     { return "Define or list textual command aliases" }

func (Alias) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("alias", pflag.ContinueOnError)
	fs.Bool("global", false, "substitute anywhere on the line, not just the first word")
	fs.Bool("erase", false, "remove the named alias")
	return fs
}

func (b Alias) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	erase, _ := fs.GetBool("erase")
	pos := fs.Args()

	if len(pos) == 0 {
		for _, k := range sortedKeys(s.Aliases) {
			fmt.Fprintf(s.Out, "%s=%s\n", k, s.Aliases[k])
		}
		return nil
	}

	name, value, hasEq := strings.Cut(pos[0], "=")
	if erase {
		delete(s.Aliases, name)
		return nil
	}
	if !hasEq {
		if v, ok := s.Aliases[name]; ok {
			fmt.Fprintf(s.Out, "%s=%s\n", name, v)
			return nil
		}
		return fmt.Errorf("alias: %q is not defined", name)
	}
	s.Aliases[name] = value
	return nil
}

// Unalias implements the "unalias <name>... [--all]" builtin.
type Unalias struct{}

func (Unalias) Name() string      { return "unalias" }
func (Unalias) Aliases() []string { return nil }
func (Unalias) Short() string     { return "Remove one or more aliases" }

func (Unalias) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("unalias", pflag.ContinueOnError)
	fs.Bool("all", false, "remove every alias")
	return fs
}

func (b Unalias) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	all, _ := fs.GetBool("all")
	if all {
		s.Aliases = make(map[string]string)
		return nil
	}
	names := fs.Args()
	if len(names) == 0 {
		return fmt.Errorf("unalias: requires at least one <name>, or --all")
	}
	for _, n := range names {
		delete(s.Aliases, n)
	}
	return nil
}
