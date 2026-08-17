// Command docgen dumps the shell builtin registry and template FuncMap
// registry as JSON, for the M11.0 docs-site generator
// (site/scripts/gen-shell-docs.mjs) to consume. It exists because the
// registries live in Go code (internal/shell/builtin, internal/shell/tmpl)
// and aren't otherwise introspectable from the Node-based doc pipeline.
//
// This is dev/build tooling only — it is not part of the maelsink binary
// and is not imported by any cmd/ package.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/0funct0ry/maelsink/internal/shell/builtin"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
	"github.com/spf13/pflag"
)

// builtinDoc is one entry in the "builtins" array of the output JSON.
type builtinDoc struct {
	Name    string    `json:"name"`
	Aliases []string  `json:"aliases"`
	Flags   []flagDoc `json:"flags"`
}

// flagDoc describes a single pflag on a builtin's FlagSet.
type flagDoc struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand"`
	Usage     string `json:"usage"`
	DefValue  string `json:"defValue"`
}

// funcDoc is one entry in the "functions" array of the output JSON.
type funcDoc struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Args        string `json:"args"`
	Returns     string `json:"returns"`
	Description string `json:"description"`
	Unsafe      bool   `json:"unsafe"`
}

type output struct {
	Builtins  []builtinDoc `json:"builtins"`
	Functions []funcDoc    `json:"functions"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "docgen:", err)
		os.Exit(1)
	}
}

func run() error {
	var out output

	for _, b := range builtin.All() {
		bd := builtinDoc{
			Name:    b.Name(),
			Aliases: b.Aliases(),
		}
		if bd.Aliases == nil {
			bd.Aliases = []string{}
		}
		fs := b.Flags()
		var flags []flagDoc
		fs.VisitAll(func(f *pflag.Flag) {
			flags = append(flags, flagDoc{
				Name:      f.Name,
				Shorthand: f.Shorthand,
				Usage:     f.Usage,
				DefValue:  f.DefValue,
			})
		})
		bd.Flags = flags
		if bd.Flags == nil {
			bd.Flags = []flagDoc{}
		}
		out.Builtins = append(out.Builtins, bd)
	}

	// Build the template engine twice: once safe-only, once with unsafe
	// funcs enabled. Anything present in the unsafe registry but absent
	// from the safe one is gated behind --template-unsafe-funcs/-Z.
	safeEngine, err := tmpl.New(1, false)
	if err != nil {
		return fmt.Errorf("building safe template engine: %w", err)
	}
	unsafeEngine, err := tmpl.New(1, true)
	if err != nil {
		return fmt.Errorf("building unsafe template engine: %w", err)
	}

	safeNames := make(map[string]bool)
	for _, d := range safeEngine.Registry() {
		safeNames[d.Name] = true
	}

	for _, d := range unsafeEngine.Registry() {
		out.Functions = append(out.Functions, funcDoc{
			Name:        d.Name,
			Category:    string(d.Category),
			Args:        d.Args,
			Returns:     d.Returns,
			Description: d.Description,
			Unsafe:      !safeNames[d.Name],
		})
	}

	sort.Slice(out.Builtins, func(i, j int) bool { return out.Builtins[i].Name < out.Builtins[j].Name })

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
