package builtin

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell"
)

// Show implements the "show <id>" builtin (SPEC.md §7.5.4).
type Show struct{}

func (Show) Name() string      { return "show" }
func (Show) Aliases() []string { return []string{"get"} }
func (Show) Short() string     { return "Show a message's full detail" }

func (Show) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("show", pflag.ContinueOnError)
	fs.String("part", "all", "part to show: html|text|raw|headers|all")
	fs.String("format", "table", "output format: table|json|yaml")
	fs.StringP("out", "o", "", "write the selected part to this file instead of stdout")
	return fs
}

func (b Show) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	pos := fs.Args()
	if len(pos) == 0 {
		return fmt.Errorf("show: requires <id>")
	}
	id := pos[0]

	part, _ := fs.GetString("part")
	format, _ := fs.GetString("format")
	out, _ := fs.GetString("out")

	if part == "raw" {
		raw, err := s.Client.ExportRaw(ctx, id)
		if err != nil {
			return ambiguousAwareError(s, err)
		}
		if out != "" {
			return os.WriteFile(out, raw, 0o644)
		}
		_, werr := s.Out.Write(raw)
		return werr
	}

	detail, err := s.Client.Get(ctx, id)
	if err != nil {
		return ambiguousAwareError(s, err)
	}
	s.SetVar("last_id", detail.ID)

	var content string
	switch part {
	case "html":
		content = detail.HTMLBody
	case "text":
		content = detail.TextBody
	case "headers":
		for _, h := range detail.Headers {
			content += fmt.Sprintf("%s: %s\n", h.Name, h.Value)
		}
	case "all", "":
		var w io.Writer = s.Out
		var f *os.File
		if out != "" {
			f, err = os.Create(out)
			if err != nil {
				return err
			}
			defer f.Close()
			w = f
		}
		return writeFormatted(w, format, detail, func(w io.Writer) error {
			cliclient.RenderDetail(w, detail)
			return nil
		})
	default:
		return fmt.Errorf("show: unknown --part %q", part)
	}

	if out != "" {
		return os.WriteFile(out, []byte(content), 0o644)
	}
	fmt.Fprintln(s.Out, content)
	return nil
}

// ambiguousAwareError renders internal/api's ambiguous_id error code as a
// readable "supply more characters" message; any other error falls back to
// clientError's transport-vs-HTTP-error formatting.
func ambiguousAwareError(s *shell.Session, err error) error {
	if shell.IsAmbiguousID(err) {
		msg := "that ID prefix matches more than one message; supply more characters"
		fmt.Fprintln(s.Err, msg)
		return fmt.Errorf("%s", msg)
	}
	return clientError(s, err)
}
