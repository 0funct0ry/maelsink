package builtin

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Health implements the "health" builtin (SPEC.md §7.5.4). A non-"ok"
// status returns a non-nil error so the command's own exit status reflects
// server degradation.
type Health struct{}

func (Health) Name() string      { return "health" }
func (Health) Aliases() []string { return nil }
func (Health) Short() string     { return "Server liveness/readiness" }

func (Health) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("health", pflag.ContinueOnError)
	fs.String("format", "table", "output format: table|json|yaml")
	return fs
}

func (b Health) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	format, _ := fs.GetString("format")

	h, err := s.Client.Health(ctx)
	if err != nil {
		return clientError(s, err)
	}
	if err := writeFormatted(s.Out, format, h, func(w io.Writer) error {
		fmt.Fprintf(w, "Status:\t%s\n", h.Status)
		fmt.Fprintf(w, "DB:\t%s\n", h.DB)
		fmt.Fprintf(w, "SMTP:\t%s\n", h.SMTP)
		return nil
	}); err != nil {
		return err
	}
	if h.Status != "ok" {
		return fmt.Errorf("health: server status is %q", h.Status)
	}
	return nil
}
