package shell

import (
	"context"
	"strings"

	"github.com/0funct0ry/maelsink/internal/config"
	"github.com/0funct0ry/maelsink/internal/shell/lineedit"
)

// editBuffer implements the Ctrl-X Ctrl-E line-edit integration
// (lineedit.Config.EditFunc): it delegates to RunEditor (writes buffer to
// a temp file, execs the resolved editor against it — inheriting the
// process's own stdio, since the spawned editor manages its own terminal
// raw-mode setup on the shared tty, same as any interactive-editor-from-a-
// shell invocation; no explicit "release the terminal" call is needed on
// the lineedit.Editor side, and none is exposed by
// github.com/ergochat/readline's Instance either), then strips the
// trailing newline every editor writes on save — this result becomes a
// single-line readline buffer (lineedit.Config.EditFunc's contract), so a
// literal trailing "\n" would otherwise land in the buffer as an embedded
// newline, visually pushing the cursor onto a phantom second line rather
// than to the end of the edited text. Per SPEC.md §7.5.9, the result
// replaces the line buffer but is not auto-submitted.
func editBuffer(cfgEditor, buffer string) (string, error) {
	result, err := RunEditor(context.Background(), cfgEditor, buffer)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(result, "\r\n"), nil
}

// DefaultNewEditor builds the func(*Session) (*lineedit.Editor, error)
// wiring cmd/shell.go passes as Options.NewEditor: it constructs a real
// lineedit.Config from cfg and the session's live state (completion source,
// abbreviations, editor invocation) and returns lineedit.New(cfg).
func DefaultNewEditor(cfg config.Shell) func(*Session) (*lineedit.Editor, error) {
	return func(s *Session) (*lineedit.Editor, error) {
		lcfg := lineedit.Config{
			Prompt: cfg.Prompt,
			PromptFunc: func() string {
				return renderPrompt(s)
			},
			HistoryFile:    cfg.HistoryFile,
			AbbrTriggerKey: cfg.AbbrTriggerKey,
			Color:          cfg.Color,
			Source:         newSessionCompletionAdapter(s),
			EditFunc: func(buffer string) (string, error) {
				return editBuffer(s.Cfg.Editor, buffer)
			},
			Abbrs: func() map[string]string {
				return s.Abbrs
			},
		}
		return lineedit.New(lcfg)
	}
}

// renderPrompt renders cfg.Prompt as a template against the session's
// current variables, so the prompt can reflect live state (e.g. an
// "(offline)" indicator via $connected). Rendering errors fall back to the
// raw, unrendered prompt string rather than failing the whole read loop.
func renderPrompt(s *Session) string {
	if s.Tmpl == nil {
		return s.Cfg.Prompt
	}
	rendered, err := ExpandTemplate(s.Tmpl, s.TemplateData(), s.Cfg.Prompt, s.Cfg.TemplateEnabled)
	if err != nil {
		return s.Cfg.Prompt
	}
	return rendered
}
