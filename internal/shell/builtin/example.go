package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Example implements the "example" builtin: generates a templated .eml or
// JSON message-spec file from one of 10 canned scenarios (picked randomly,
// or via --index), for a user to inspect, tweak with "edit -f <path>", and
// send with "send --template <path>" (eml) or "send --json <path>" (json).
// Every string field is left UNRENDERED — the point is to hand the user a
// realistic starting point that still contains live template functions
// they can see and change, not a one-off rendered sample.
type Example struct{}

func (Example) Name() string      { return "example" }
func (Example) Aliases() []string { return nil }
func (Example) Short() string     { return "Generate a templated example .eml/.json to edit and send" }

func (Example) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("example", pflag.ContinueOnError)
	fs.String("format", "eml", "output format: eml|json")
	fs.Int("index", 0, "pick a specific canned example (1-based; default: random)")
	fs.Bool("list", false, "list the canned examples instead of generating one")
	fs.StringP("out", "o", "", "output path (default: a generated path under the session's temp dir)")
	return fs
}

func (b Example) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	fs.SetOutput(s.Out)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if list, _ := fs.GetBool("list"); list {
		for i, ex := range exampleTemplates {
			fmt.Fprintf(s.Out, "%2d  %s\n", i+1, ex.Title)
		}
		return nil
	}

	format, _ := fs.GetString("format")
	if format != "eml" && format != "json" {
		return fmt.Errorf("example: --format must be eml or json, got %q", format)
	}

	idx, _ := fs.GetInt("index")
	if idx != 0 {
		if idx < 1 || idx > len(exampleTemplates) {
			return fmt.Errorf("example: --index out of range (1-%d)", len(exampleTemplates))
		}
		idx--
	} else {
		idx = s.Tmpl.Intn(len(exampleTemplates))
	}
	ex := exampleTemplates[idx]

	var content string
	ext := format
	switch format {
	case "json":
		spec := struct {
			From    string   `json:"from"`
			To      []string `json:"to"`
			Subject string   `json:"subject"`
			Text    string   `json:"text,omitempty"`
			HTML    string   `json:"html,omitempty"`
		}{From: ex.From, To: []string{ex.To}, Subject: ex.Subject, Text: ex.Text, HTML: ex.HTML}
		raw, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return err
		}
		content = string(raw) + "\n"
	default: // eml
		contentType := "text/plain; charset=utf-8"
		body := ex.Text
		if ex.HTML != "" {
			contentType = "text/html; charset=utf-8"
			body = ex.HTML
		}
		content = fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\nContent-Type: %s\n\n%s\n",
			ex.From, ex.To, ex.Subject, contentType, body)
		ext = "eml"
	}

	out, _ := fs.GetString("out")
	if out == "" {
		out = filepath.Join(s.Tmpl.TempDir(), fmt.Sprintf("example-%d.%s", idx+1, ext))
	}
	if err := os.WriteFile(out, []byte(content), 0o600); err != nil {
		return err
	}

	// Both send --template and send --json derive the envelope from the
	// file's own (rendered) From/To content — --template by parsing the
	// rendered message's headers (same rule as --eml), --json from the
	// spec's own "from"/"to" fields — so neither needs extra flags.
	sendHint := "send --template " + out
	if format == "json" {
		sendHint = "send --json " + out
	}
	fmt.Fprintf(s.Out, "%s (%q) written to %s\n", format, ex.Title, out)
	fmt.Fprintf(s.Out, "edit:  edit -f %s\n", out)
	fmt.Fprintf(s.Out, "send:  %s\n", sendHint)
	return nil
}
