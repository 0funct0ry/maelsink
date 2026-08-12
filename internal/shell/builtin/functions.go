package builtin

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
	"github.com/0funct0ry/maelsink/internal/shell/lineedit"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// categoryOrder is the fixed declaration order used to group the bare
// listing (SPEC.md §7.5.7.1).
var categoryOrder = []tmpl.Category{
	tmpl.CategoryIdentifiers,
	tmpl.CategoryGenerate,
	tmpl.CategoryString,
	tmpl.CategoryDate,
	tmpl.CategoryEncoding,
	tmpl.CategoryEmail,
	tmpl.CategoryFiles,
	tmpl.CategoryAnsi,
}

// descriptionColumnWidth caps the wrapped description column so the table
// stays readable in a normal terminal width even when the function name
// column is short.
const descriptionColumnWidth = 70

// Functions implements the "functions [name|category]" builtin: lists every
// template function available to {{ }} expressions (SPEC.md §7.5.7), grouped
// by Category (§7.5.7.1), filters to one category, shows detailed help for
// one function, or (-s/--search) greps name+description across the
// registry. "fns"/"funcs" are aliases of the same handler. This is the
// discovery counterpart to the "template" builtin's debug-render use case —
// "template --funcs" already lists bare names; "functions" adds one-line
// summaries, category grouping, and per-function detail.
type Functions struct{}

func (Functions) Name() string      { return "functions" }
func (Functions) Aliases() []string { return []string{"fns", "funcs"} }
func (Functions) Short() string     { return "List template functions, or show help for one" }

func (Functions) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("functions", pflag.ContinueOnError)
	fs.StringP("search", "s", "", "case-insensitive substring match against name or description")
	return fs
}

func (b Functions) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	fs.SetOutput(s.Out)
	if err := fs.Parse(args); err != nil {
		return err
	}
	pos := fs.Args()
	search, _ := fs.GetString("search")

	reg := s.Tmpl.Registry()
	color := lineedit.ResolveColor(s.Cfg.Color)

	if len(pos) == 0 {
		if search != "" {
			printFlat(s, searchRegistry(reg, search), color)
			return nil
		}
		printGrouped(s, reg, color)
		return nil
	}

	name := pos[0]

	if isCategory(reg, name) {
		cat := tmpl.Category(name)
		filtered := filterByCategory(reg, cat)
		if search != "" {
			filtered = searchRegistry(filtered, search)
		}
		printFlat(s, filtered, color)
		return nil
	}

	for _, d := range reg {
		if d.Name == name {
			fmt.Fprintf(s.Out, "%s\n\n  {{ %s %s }}\n\n  returns: %s\n\n  %s\n", d.Name, d.Name, d.Args, d.Returns, d.Description)
			return nil
		}
	}

	return fmt.Errorf("functions: no such function or category %q", name)
}

func isCategory(reg []tmpl.FuncDoc, name string) bool {
	for _, d := range reg {
		if string(d.Category) == name {
			return true
		}
	}
	return false
}

func filterByCategory(reg []tmpl.FuncDoc, cat tmpl.Category) []tmpl.FuncDoc {
	var out []tmpl.FuncDoc
	for _, d := range reg {
		if d.Category == cat {
			out = append(out, d)
		}
	}
	return out
}

func searchRegistry(reg []tmpl.FuncDoc, term string) []tmpl.FuncDoc {
	term = strings.ToLower(term)
	var out []tmpl.FuncDoc
	for _, d := range reg {
		if strings.Contains(strings.ToLower(d.Name), term) || strings.Contains(strings.ToLower(d.Description), term) {
			out = append(out, d)
		}
	}
	return out
}

// newFunctionsTable returns a table.Writer preconfigured with the two-column
// (name, description) layout shared by the grouped and flat listings: the
// name column colored and left-aligned, the description column word-wrapped
// to descriptionColumnWidth.
func newFunctionsTable(color bool) table.Writer {
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.Style().Options.SeparateRows = false
	nameColors := text.Colors{}
	if color {
		nameColors = text.Colors{text.FgCyan, text.Bold}
	}
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Colors: nameColors},
		{Number: 2, WidthMax: descriptionColumnWidth, WidthMaxEnforcer: text.WrapText},
	})
	return t
}

func printFlat(s *shell.Session, docs []tmpl.FuncDoc, color bool) {
	sorted := append([]tmpl.FuncDoc(nil), docs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	t := newFunctionsTable(color)
	for _, d := range sorted {
		t.AppendRow(table.Row{d.Name, d.Description})
	}
	fmt.Fprintln(s.Out, t.Render())
}

func printGrouped(s *shell.Session, reg []tmpl.FuncDoc, color bool) {
	for _, cat := range categoryOrder {
		filtered := filterByCategory(reg, cat)
		if len(filtered) == 0 {
			continue
		}
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })

		header := string(cat) + ":"
		if color {
			header = text.Colors{text.FgYellow, text.Bold}.Sprint(header)
		}
		fmt.Fprintln(s.Out, header)

		t := newFunctionsTable(color)
		for _, d := range filtered {
			t.AppendRow(table.Row{d.Name, d.Description})
		}
		fmt.Fprintln(s.Out, t.Render())
		fmt.Fprintln(s.Out)
	}
}
