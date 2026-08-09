package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Attachment implements the "attachment <id> [attId|index]" builtin
// (SPEC.md §7.5.4).
type Attachment struct{}

func (Attachment) Name() string      { return "attachment" }
func (Attachment) Aliases() []string { return []string{"att"} }
func (Attachment) Short() string     { return "List or download a message's attachments" }

func (Attachment) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("attachment", pflag.ContinueOnError)
	fs.StringP("out", "o", "", "output path (file, or directory with --all)")
	fs.Bool("all", false, "download every attachment into the --out directory")
	fs.Bool("stdout", false, "write raw bytes to stdout")
	return fs
}

func (b Attachment) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	pos := fs.Args()
	if len(pos) == 0 {
		return fmt.Errorf("attachment: requires <id>")
	}
	id := pos[0]
	out, _ := fs.GetString("out")
	all, _ := fs.GetBool("all")
	toStdout, _ := fs.GetBool("stdout")

	detail, err := s.Client.Get(ctx, id)
	if err != nil {
		return ambiguousAwareError(s, err)
	}

	if len(pos) < 2 && !all {
		// List attachments.
		if len(detail.Attachments) == 0 {
			fmt.Fprintln(s.Out, "No attachments.")
			return nil
		}
		fmt.Fprintln(s.Out, "INDEX\tID\tFILENAME\tCONTENT-TYPE\tSIZE")
		for i, a := range detail.Attachments {
			fmt.Fprintf(s.Out, "%d\t%s\t%s\t%s\t%d\n", i+1, a.ID, a.Filename, a.ContentType, a.SizeBytes)
		}
		return nil
	}

	if all {
		dir := out
		if dir == "" {
			dir = "."
		}
		info, statErr := os.Stat(dir)
		if statErr == nil && !info.IsDir() {
			return fmt.Errorf("attachment --all: --out %q is not a directory", dir)
		}
		if statErr != nil {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
		for _, a := range detail.Attachments {
			data, _, filename, err := s.Client.Attachment(ctx, id, a.ID)
			if err != nil {
				fmt.Fprintf(s.Err, "attachment %s: %s\n", a.ID, shell.FormatClientError(err))
				continue
			}
			path := filepath.Join(dir, filename)
			if err := os.WriteFile(path, data, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(s.Out, "wrote %s\n", path)
		}
		return nil
	}

	// Resolve second arg: bare integer = 1-based index; otherwise an
	// attachment ID (or prefix, resolved server-side).
	attArg := pos[1]
	attID := attArg
	if idx, err := strconv.Atoi(attArg); err == nil {
		if idx < 1 || idx > len(detail.Attachments) {
			return fmt.Errorf("attachment: index %d out of range (1-%d)", idx, len(detail.Attachments))
		}
		attID = detail.Attachments[idx-1].ID
	}

	data, _, filename, err := s.Client.Attachment(ctx, id, attID)
	if err != nil {
		return ambiguousAwareError(s, err)
	}

	if toStdout {
		_, werr := s.Out.Write(data)
		return werr
	}

	path := out
	if path == "" {
		path = "./" + filename
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(s.Out, "wrote %s\n", path)
	return nil
}
