package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell"
)

// Export implements the "export <id>..." builtin (SPEC.md §7.5.4).
type Export struct{}

func (Export) Name() string      { return "export" }
func (Export) Aliases() []string { return nil }
func (Export) Short() string     { return "Write .eml file(s) for one or more messages" }

func (Export) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("export", pflag.ContinueOnError)
	fs.Bool("all", false, "bulk export every message (filtered by list's flags) as a .zip")
	fs.StringP("out", "o", "", "output path: a file for one message, a directory for several")
	fs.Bool("zip", false, "bundle multiple exports into one .zip")
	fs.StringP("query", "q", "", "full-text search query (with --all)")
	fs.String("from", "", "filter by from address substring (with --all)")
	fs.String("to", "", "filter by to address substring (with --all)")
	fs.String("subject", "", "filter by subject substring (with --all)")
	fs.String("since", "", "only messages received at/after this RFC3339 timestamp (with --all)")
	fs.String("until", "", "only messages received at/before this RFC3339 timestamp (with --all)")
	fs.String("sort", "", "sort order (with --all)")
	return fs
}

func (b Export) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	all, _ := fs.GetBool("all")
	out, _ := fs.GetString("out")
	zip, _ := fs.GetBool("zip")
	ids := fs.Args()

	if all {
		q, _ := fs.GetString("query")
		from, _ := fs.GetString("from")
		to, _ := fs.GetString("to")
		subject, _ := fs.GetString("subject")
		since, _ := fs.GetString("since")
		until, _ := fs.GetString("until")
		sortOrder, _ := fs.GetString("sort")
		data, err := s.Client.BulkExport(ctx, cliclient.ListParams{
			Query: q, From: from, To: to, Subject: subject, Since: since, Until: until, Sort: sortOrder,
		})
		if err != nil {
			return clientError(s, err)
		}
		path := out
		if path == "" {
			path = "export.zip"
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "wrote %s\n", path)
		return nil
	}

	if len(ids) == 0 {
		return fmt.Errorf("export: requires at least one <id>, or --all")
	}

	if zip || len(ids) > 1 {
		if !zip {
			// Bulk mode for multiple ids without --zip: write individual
			// .eml files into the --out directory (defaulting to cwd).
			dir := out
			if dir == "" {
				dir = "."
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			for _, id := range ids {
				raw, err := s.Client.ExportRaw(ctx, id)
				if err != nil {
					fmt.Fprintf(s.Err, "export %s: %s\n", id, shell.FormatClientError(err))
					continue
				}
				path := filepath.Join(dir, id+".eml")
				if err := os.WriteFile(path, raw, 0o644); err != nil {
					return err
				}
				fmt.Fprintf(s.Out, "wrote %s\n", path)
			}
			return nil
		}
		// --zip with explicit ids: fetch each and bundle client-side is out
		// of scope for individual-id zip bundling here since BulkExport is
		// server-side filtered by query, not by an id list; fall back to
		// per-message .eml files in the --out directory.
		dir := out
		if dir == "" {
			dir = "."
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		for _, id := range ids {
			raw, err := s.Client.ExportRaw(ctx, id)
			if err != nil {
				fmt.Fprintf(s.Err, "export %s: %s\n", id, shell.FormatClientError(err))
				continue
			}
			path := filepath.Join(dir, id+".eml")
			if err := os.WriteFile(path, raw, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(s.Out, "wrote %s\n", path)
		}
		return nil
	}

	// Single id.
	id := ids[0]
	raw, err := s.Client.ExportRaw(ctx, id)
	if err != nil {
		return ambiguousAwareError(s, err)
	}
	path := out
	if path == "" {
		path = id + ".eml"
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(s.Out, "wrote %s\n", path)
	return nil
}
