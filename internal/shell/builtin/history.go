package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// History implements the "history" builtin (SPEC.md §7.5.4). It reads
// s.History, which internal/shell.Run / cmd/shell.go set post-construction
// (see internal/shell/session.go's SetHistory). If unset, it errors clearly
// rather than panicking.
type History struct{}

func (History) Name() string      { return "history" }
func (History) Aliases() []string { return []string{"hist"} }
func (History) Short() string     { return "Show numbered command history" }

func (History) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("history", pflag.ContinueOnError)
	fs.IntP("limit", "n", 0, "show only the last N entries (0 = all)")
	fs.Bool("clear", false, "clear session history")
	fs.String("search", "", "only show entries containing this substring")
	fs.IntP("edit", "e", 0, "open history entry <num> (as numbered by a plain 'history') in $EDITOR — same load-don't-execute behavior as the \"edit\" builtin")
	return fs
}

func (b History) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	fs.SetOutput(s.Out)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if s.History == nil {
		return fmt.Errorf("history: not available in this session")
	}

	if n, _ := fs.GetInt("edit"); n > 0 {
		line, ok := s.History.At(n)
		if !ok {
			return fmt.Errorf("history -e: no entry %d (run 'history' with no flags to see valid numbers)", n)
		}
		result, err := shell.RunEditor(ctx, s.Cfg.Editor, line)
		if err != nil {
			return err
		}
		return loadEditResultOrPrint(s, result)
	}

	if clear, _ := fs.GetBool("clear"); clear {
		*s.History = shell.History{}
		return nil
	}

	limit, _ := fs.GetInt("limit")
	search, _ := fs.GetString("search")

	lines := s.History.Lines()
	filtered := make([]string, 0, len(lines))
	for _, l := range lines {
		if search != "" && !strings.Contains(l, search) {
			continue
		}
		filtered = append(filtered, l)
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	start := len(lines) - len(filtered) + 1
	for i, l := range filtered {
		fmt.Fprintf(s.Out, "%5d  %s\n", start+i, l)
	}
	return nil
}
