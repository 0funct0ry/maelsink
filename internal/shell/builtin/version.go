package builtin

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
	"github.com/0funct0ry/maelsink/internal/version"
)

// Version implements the "version" builtin (SPEC.md §7.5.4).
type Version struct{}

func (Version) Name() string      { return "version" }
func (Version) Aliases() []string { return nil }
func (Version) Short() string     { return "Server (and shell binary's own) version info" }

func (Version) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("version", pflag.ContinueOnError)
	fs.String("format", "table", "output format: table|json|yaml")
	fs.Bool("local", false, "skip the API call, print the shell binary's own build info")
	return fs
}

func (b Version) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	format, _ := fs.GetString("format")
	local, _ := fs.GetBool("local")

	if local {
		info := version.Get()
		return writeFormatted(s.Out, format, info, func(w io.Writer) error {
			fmt.Fprintf(w, "Version:\t%s\n", info.Version)
			fmt.Fprintf(w, "Commit:\t%s\n", info.Commit)
			fmt.Fprintf(w, "Build date:\t%s\n", info.BuildDate)
			fmt.Fprintf(w, "Go:\t%s\n", info.Go)
			return nil
		})
	}

	v, err := s.Client.Version(ctx)
	if err != nil {
		return clientError(s, err)
	}
	return writeFormatted(s.Out, format, v, func(w io.Writer) error {
		fmt.Fprintf(w, "Server Version:\t%s\n", v.Version)
		fmt.Fprintf(w, "Server Commit:\t%s\n", v.Commit)
		fmt.Fprintf(w, "Server Build date:\t%s\n", v.BuildDate)
		fmt.Fprintf(w, "Server Go:\t%s\n", v.Go)
		local := version.Get()
		fmt.Fprintf(w, "Shell Version:\t%s\n", local.Version)
		fmt.Fprintf(w, "Shell Commit:\t%s\n", local.Commit)
		fmt.Fprintf(w, "Shell Build date:\t%s\n", local.BuildDate)
		fmt.Fprintf(w, "Shell Go:\t%s\n", local.Go)
		return nil
	})
}
