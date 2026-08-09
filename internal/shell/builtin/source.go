package builtin

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Source implements the "source <file>" builtin (SPEC.md §7.5.4): reads and
// evaluates a file of shell commands line by line through the same
// evaluator used by --script, via s.Registry (set post-construction by
// internal/shell.Run / cmd/shell.go).
type Source struct{}

func (Source) Name() string      { return "source" }
func (Source) Aliases() []string { return []string{"."} }
func (Source) Short() string     { return "Evaluate a file of shell commands line by line" }

func (Source) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("source", pflag.ContinueOnError)
	fs.Bool("quiet", false, "suppress per-line echo")
	return fs
}

func (b Source) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	pos := fs.Args()
	if len(pos) == 0 {
		return fmt.Errorf("source: requires <file>")
	}
	if s.Registry == nil {
		return fmt.Errorf("source: not available in this session (no command registry attached)")
	}

	f, err := os.Open(pos[0])
	if err != nil {
		return err
	}
	defer f.Close()

	quiet, _ := fs.GetBool("quiet")

	scanner := bufio.NewScanner(f)
	var lastErr error
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !quiet {
			fmt.Fprintf(s.Out, "+ %s\n", line)
		}
		if err := shell.Eval(ctx, s, s.Registry, line); err != nil {
			lastErr = err
			if s.ExitOnError {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return lastErr
}
