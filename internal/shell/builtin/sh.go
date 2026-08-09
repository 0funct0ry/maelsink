package builtin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Sh implements the "sh <command line...>" builtin (SPEC.md §7.5.4), gated
// by shell.sh_enabled.
type Sh struct{}

func (Sh) Name() string      { return "sh" }
func (Sh) Aliases() []string { return nil }
func (Sh) Short() string     { return "Run a command line through the system shell" }

func (Sh) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("sh", pflag.ContinueOnError)
	fs.Bool("quiet", false, "suppress output; exit status still lands in last_error")
	return fs
}

func (b Sh) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !s.Cfg.ShEnabled {
		return fmt.Errorf("sh: disabled (shell.sh_enabled is false)")
	}
	quiet, _ := fs.GetBool("quiet")

	line := strings.Join(fs.Args(), " ")
	if line == "" {
		return fmt.Errorf("sh: requires a command line")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", line)
	} else {
		shellBin := os.Getenv("SHELL")
		if shellBin == "" {
			shellBin = "/bin/sh"
		}
		cmd = exec.CommandContext(ctx, shellBin, "-c", line)
	}

	if !quiet {
		cmd.Stdout = s.Out
		cmd.Stderr = s.Err
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sh: %w", err)
	}
	return nil
}
