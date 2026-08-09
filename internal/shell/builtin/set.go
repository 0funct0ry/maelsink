package builtin

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// globalMarkerPrefix is the synthetic var-key convention used to record
// that a variable was set with --global, pending a real persistence
// mechanism landing alongside the "config save" builtin. This is a
// documented minimal-scope choice (see plan-M4.1.md phase 5's "set.go"
// notes): it avoids modifying internal/shell/session.go's exported Vars
// map shape, at the cost of "config save" needing to know this convention
// too (config.go does).
const globalMarkerPrefix = "__global__:"

func isGlobalMarkerKey(k string) bool { return strings.HasPrefix(k, globalMarkerPrefix) }

// Set implements the "set [k[=v]]" builtin (SPEC.md §7.5.4).
type Set struct{}

func (Set) Name() string      { return "set" }
func (Set) Aliases() []string { return nil }
func (Set) Short() string     { return "Assign or read a session variable" }

func (Set) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("set", pflag.ContinueOnError)
	fs.String("format", "table", "output format for bare `set` (same as vars): table|json|yaml")
	fs.String("from-command", "", `run a builtin ("<name> [args...]") and capture its stdout into the variable`)
	fs.Bool("global", false, "also persist this variable on the next `config save`")
	return fs
}

func (b Set) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	format, _ := fs.GetString("format")
	fromCommand, _ := fs.GetString("from-command")
	global, _ := fs.GetBool("global")

	pos := fs.Args()

	if len(pos) == 0 && fromCommand == "" {
		return printVars(s, format)
	}

	if len(pos) == 0 {
		return fmt.Errorf("set --from-command requires a variable name (set <k> --from-command \"...\")")
	}

	arg := pos[0]
	key, value, hasEq := strings.Cut(arg, "=")

	if fromCommand != "" {
		if hasEq {
			return fmt.Errorf("set: cannot combine k=v with --from-command")
		}
		out, err := runCapture(ctx, s, fromCommand)
		if err != nil {
			return err
		}
		s.SetVar(key, strings.TrimRight(out, "\n"))
		if global {
			s.SetVar(globalMarkerPrefix+key, "true")
		}
		return nil
	}

	if !hasEq {
		// Bare "set k": per SPEC.md §7.5.4 this reads the value from
		// $EDITOR. The full interactive $EDITOR-based read needs the
		// lineedit/cmd/shell.go editor callback, which this package does
		// not have standalone access to (see edit.go for the
		// standalone-batch case that IS fully implemented). For this
		// phase, print the current value if set.
		// TODO(edit-integration): wire the full $EDITOR-based read once
		// cmd/shell.go's editor callback is available to this builtin.
		if v, ok := s.GetVar(key); ok {
			fmt.Fprintln(s.Out, v)
			return nil
		}
		return fmt.Errorf("set: %q is not set (bare `set k` prints the current value; $EDITOR-based assignment lands in a later phase)", key)
	}

	s.SetVar(key, value)
	if global {
		s.SetVar(globalMarkerPrefix+key, "true")
	}
	return nil
}

// runCapture runs commandLine (tokenized the same way as any shell line,
// via shell.Tokenize) as a builtin invocation against s.Registry, capturing
// its stdout into a string. Returns a clear error if s.Registry is unset.
func runCapture(ctx context.Context, s *shell.Session, commandLine string) (string, error) {
	if s.Registry == nil {
		return "", fmt.Errorf("set --from-command: not available in this session (no command registry attached)")
	}
	tokens, err := shell.Tokenize(commandLine)
	if err != nil {
		return "", err
	}
	if len(tokens) == 0 {
		return "", fmt.Errorf("set --from-command: empty command")
	}

	var buf bytes.Buffer
	origOut := s.Out
	s.Out = &buf
	defer func() { s.Out = origOut }()

	err = shell.Dispatch(ctx, s, s.Registry, tokens)
	return buf.String(), err
}
