// Package builtin implements every shell.Builtin from SPEC.md §7.5.4: the
// command table the interactive/scriptable maelsink shell dispatches
// against. It is a pure client of internal/cliclient (REST + SMTP) plus
// internal/shell's own Session/Registry/History/tmpl types — it must never
// import internal/store, internal/smtp, internal/api, or internal/webui.
package builtin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell"
)

// Describable is optionally implemented by a shell.Builtin to provide a
// one-line description for the "help" builtin's summary listing. Builtins
// that don't implement it render with an empty description.
type Describable interface {
	Short() string
}

// writeFormatted renders v to w according to format ("table", "json",
// "yaml", or any other value treated as "table"). renderTable is called
// only for the "table" case, since table rendering is shape-specific per
// builtin (list vs. show vs. stats all render differently).
func writeFormatted(w io.Writer, format string, v any, renderTable func(io.Writer) error) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case "yaml":
		enc := yaml.NewEncoder(w)
		defer enc.Close()
		return enc.Encode(v)
	default:
		if renderTable != nil {
			return renderTable(w)
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
}

// clientError prints FormatClientError(err) to s.Err and returns an error
// so the command's exit status reflects failure, without double-printing
// the underlying error text via Cobra/Eval's own error path (builtins are
// responsible for their own output per shell.Builtin's contract).
func clientError(s *shell.Session, err error) error {
	msg := shell.FormatClientError(err)
	fmt.Fprintln(s.Err, msg)
	return fmt.Errorf("%s", msg)
}

// sortedKeys returns the sorted keys of m.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// confirmPrompt writes prompt to s.Out and reads one line from s.In,
// returning true only if the trimmed, lowercased response is "y" or "yes".
func confirmPrompt(s *shell.Session, prompt string) bool {
	fmt.Fprint(s.Out, prompt)
	if s.In == nil {
		return false
	}
	scanner := bufio.NewScanner(s.In)
	if !scanner.Scan() {
		return false
	}
	resp := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return resp == "y" || resp == "yes"
}

// renderSummaries writes msgs per format, defaulting to cliclient's table
// renderer.
func renderSummaries(w io.Writer, format string, msgs []cliclient.MessageSummary) error {
	return writeFormatted(w, format, msgs, func(w io.Writer) error {
		if len(msgs) == 0 {
			fmt.Fprintln(w, "No messages.")
			return nil
		}
		cliclient.RenderTable(w, msgs)
		return nil
	})
}

// loadEditResultOrPrint applies SPEC.md §7.5.9's "load, don't execute"
// contract to arbitrary edited text: in an interactive session, it stages
// result as the next prompt's line buffer (Session.PendingBuffer); in
// batch modes, it prints result to s.Out instead, since there's no line
// buffer to seed. Shared by the "edit" builtin's bare (no -f) case and the
// "history -e/--edit <num>" case, which both edit scratch text destined
// for the prompt rather than an actual file on disk (contrast with "edit
// -f <path>", which edits path in place via shell.RunEditorOnFile and has
// no buffer/print step at all).
func loadEditResultOrPrint(s *shell.Session, result string) error {
	if s.Interactive {
		// The result becomes a single-line prompt buffer; strip the
		// trailing newline every editor writes on save so it doesn't show
		// up as an embedded newline (a phantom second, empty prompt line
		// with the cursor stuck there instead of at the end of the text).
		s.SetPendingBuffer(strings.TrimRight(result, "\r\n"))
		return nil
	}
	_, err := s.Out.Write([]byte(result))
	return err
}
