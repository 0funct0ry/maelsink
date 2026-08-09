package builtin

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell"
)

const maxListLimit = 500

// List implements the "list [query]" builtin (SPEC.md §7.5.4).
type List struct{}

func (List) Name() string      { return "list" }
func (List) Aliases() []string { return []string{"ls"} }
func (List) Short() string     { return "List captured messages, newest first" }

func (List) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("list", pflag.ContinueOnError)
	fs.StringP("query", "q", "", "full-text search query")
	fs.String("from", "", "filter by from address substring")
	fs.String("to", "", "filter by to address substring")
	fs.String("subject", "", "filter by subject substring")
	fs.String("since", "", "only messages received at/after this RFC3339 timestamp")
	fs.String("until", "", "only messages received at/before this RFC3339 timestamp")
	fs.IntP("limit", "n", 50, "max messages to return (max 500)")
	fs.Int("offset", 0, "pagination offset")
	fs.String("sort", "", "sort order: received_at_desc|received_at_asc")
	fs.String("format", "table", "output format: table|json|yaml")
	fs.Bool("ids", false, "print IDs only, one per line")
	return fs
}

func (b List) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}

	query, _ := fs.GetString("query")
	if pos := fs.Args(); len(pos) > 0 && query == "" {
		query = pos[0]
	}
	from, _ := fs.GetString("from")
	to, _ := fs.GetString("to")
	subject, _ := fs.GetString("subject")
	since, _ := fs.GetString("since")
	until, _ := fs.GetString("until")
	limit, _ := fs.GetInt("limit")
	offset, _ := fs.GetInt("offset")
	sortOrder, _ := fs.GetString("sort")
	format, _ := fs.GetString("format")
	idsOnly, _ := fs.GetBool("ids")

	if limit > maxListLimit {
		return fmt.Errorf("--limit %d exceeds the max of %d", limit, maxListLimit)
	}

	resp, err := s.Client.List(ctx, cliclient.ListParams{
		Query: query, From: from, To: to, Subject: subject,
		Since: since, Until: until, Limit: limit, Offset: offset, Sort: sortOrder,
	})
	if err != nil {
		return clientError(s, err)
	}

	if len(resp.Messages) > 0 {
		s.SetVar("last_id", resp.Messages[0].ID)
	}

	if idsOnly {
		for _, m := range resp.Messages {
			fmt.Fprintln(s.Out, m.ID)
		}
		return nil
	}

	return renderSummaries(s.Out, format, resp.Messages)
}
