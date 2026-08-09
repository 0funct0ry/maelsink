package builtin

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Stats implements the "stats" builtin (SPEC.md §7.5.4).
type Stats struct{}

func (Stats) Name() string      { return "stats" }
func (Stats) Aliases() []string { return nil }
func (Stats) Short() string     { return "Message count, storage size, oldest/newest timestamps" }

func (Stats) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("stats", pflag.ContinueOnError)
	fs.String("format", "table", "output format: table|json|yaml")
	return fs
}

func (b Stats) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	format, _ := fs.GetString("format")

	st, err := s.Client.Stats(ctx)
	if err != nil {
		return clientError(s, err)
	}
	return writeFormatted(s.Out, format, st, func(w io.Writer) error {
		fmt.Fprintf(w, "Total Messages:\t%d\n", st.TotalMessages)
		fmt.Fprintf(w, "Total Size:\t%d bytes\n", st.TotalSizeBytes)
		if st.OldestReceivedAt != nil {
			fmt.Fprintf(w, "Oldest:\t%s\n", *st.OldestReceivedAt)
		}
		if st.NewestReceivedAt != nil {
			fmt.Fprintf(w, "Newest:\t%s\n", *st.NewestReceivedAt)
		}
		return nil
	})
}
