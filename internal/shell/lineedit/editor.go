package lineedit

import (
	"errors"
	"io"

	"github.com/ergochat/readline"
)

// ErrLineCleared is never actually returned by Editor.ReadLine — Ctrl-C is
// handled internally by looping and re-prompting (see ReadLine below) so
// that callers never have to special-case it. It's kept as a documented,
// exported sentinel in case a future caller wants to detect "the user hit
// Ctrl-C" without ReadLine swallowing it; today nothing produces it.
var ErrLineCleared = errors.New("lineedit: line cleared (ctrl-c)")

// Config configures an Editor. See the field comments for the design
// decisions each one implies — several of them (HistoryFile, EditFunc,
// Abbrs) exist specifically to keep lineedit decoupled from internal/shell.
type Config struct {
	// Prompt is a static fallback prompt, used only if PromptFunc is nil.
	Prompt string

	// PromptFunc, when set, is called fresh before each ReadLine to obtain
	// the current prompt text. internal/shell owns rendering shell.prompt
	// as a template against session variables (so it can reflect live
	// state such as "(offline)"); lineedit just displays whatever string
	// it's handed.
	PromptFunc func() string

	// HistoryFile is intentionally UNUSED by lineedit — see the comment on
	// New() for why. It is kept on Config purely so callers/documentation
	// have somewhere to note the configured path; passing it here has no
	// effect on readline's own history file.
	HistoryFile string

	// AbbrTriggerKey is one of "space", "tab", "enter", or "none".
	AbbrTriggerKey string

	// Color is "auto", "always", or "never" — see ResolveColor.
	Color string

	// Source backs tab completion.
	Source CompletionSource

	// EditFunc is invoked on Ctrl-X Ctrl-E with the current buffer
	// contents. lineedit knows nothing about $EDITOR/$VISUAL or session
	// config, so this callback is entirely internal/shell's
	// responsibility: write the buffer to a temp file, restore cooked
	// terminal mode, exec the editor, wait, and return the reloaded
	// contents. The returned string REPLACES the buffer; it is not
	// submitted automatically — the user still presses Enter.
	EditFunc func(buffer string) (string, error)

	// Abbrs returns the current abbreviation map (name -> expansion).
	// lineedit calls it fresh on every trigger keypress so it always sees
	// live state without owning any mutable abbreviation storage itself.
	Abbrs func() map[string]string
}

// Editor wraps a *readline.Instance, adding maelsink's abbreviation
// expansion, Ctrl-X Ctrl-E editor integration, and Ctrl-Space
// literal-trigger handling on top of readline's own emacs-style defaults.
type Editor struct {
	inst   *readline.Instance
	cfg    Config
	color  bool
	trig   rune
	trigOK bool

	// pendingCtrlX/pendingEdit/literalNext are the small pieces of state
	// the FuncFilterInputRune/Listener pair need to share across
	// keypresses. See the comment on New() for why they're implemented
	// this way rather than as readline key bindings.
	pendingCtrlX bool
	pendingEdit  bool
	literalNext  bool
}

// New builds an Editor from cfg.
//
// History file ownership: ergochat/readline's own Config.HistoryFile, if
// set, makes readline open/append/rewrite that file itself (see its
// history.go: opHistory.historyUpdatePath / rewriteLocked). That mechanism
// has no concept of maelsink's secret-redaction rule (§7.5.8 — lines
// containing --api-key/--auth-pass/--auth-user must never reach disk), and
// running it alongside internal/shell.History's own persistence would mean
// two independent writers disagreeing about what belongs in the file.
// So: lineedit ALWAYS passes HistoryFile: "" to readline (rewriteLocked
// returns immediately when that's empty — confirmed by reading
// github.com/ergochat/readline@v0.1.3/history.go — so no file is ever
// touched by readline itself) and sets DisableAutoSaveHistory: true so
// readline doesn't even auto-push submitted lines into its in-memory list.
// internal/shell.History is therefore the SOLE source of truth for what
// gets persisted to disk. The in-memory recall list (Ctrl-P/Ctrl-R) is
// populated exclusively via Editor.AddHistoryLine, which the shell.go
// caller invokes after shell.History.Add() has already decided a line
// belongs in the session's recall list — so both views stay in sync with
// exactly one policy (shell.History's) governing membership.
func New(cfg Config) (*Editor, error) {
	e := &Editor{cfg: cfg}
	e.color = ResolveColor(cfg.Color)
	e.trig, e.trigOK = triggerRune(cfg.AbbrTriggerKey)

	rlCfg := &readline.Config{
		Prompt:                 cfg.Prompt,
		HistoryFile:            "", // see New()'s doc comment
		DisableAutoSaveHistory: true,
		AutoComplete:           NewCompleter(cfg.Source),
		Listener:               e.listener,
		FuncFilterInputRune:    e.filterInputRune,
	}

	inst, err := readline.NewFromConfig(rlCfg)
	if err != nil {
		return nil, err
	}
	e.inst = inst
	return e, nil
}

// filterInputRune is readline's Config.FuncFilterInputRune hook: it runs
// BEFORE a keypress reaches readline's normal dispatch, and can swallow a
// rune entirely by returning ok=false. That makes it the only place we can
// intercept Ctrl-X (24) before readline's default case writes it into the
// buffer as a literal control byte, and the only place we can translate a
// Ctrl-Space keypress (which arrives as NUL/0 on the terminals we've
// checked) into a plain space without it being indistinguishable from a
// real space by the time our Listener sees it.
func (e *Editor) filterInputRune(r rune) (rune, bool) {
	const ctrlX = 24
	const ctrlE = 5
	const nul = 0

	if e.pendingCtrlX {
		e.pendingCtrlX = false
		if r == ctrlE {
			// Full Ctrl-X Ctrl-E chord observed. Let Ctrl-E pass through
			// normally (readline's own binding just moves the cursor to
			// end of line, which is a harmless side effect here); flag
			// the Listener to invoke EditFunc once it sees this keypress.
			e.pendingEdit = true
			return r, true
		}
		// Not a chord — the swallowed Ctrl-X is simply lost. This is a
		// documented, minor limitation: Ctrl-X on its own does nothing.
	}

	if r == ctrlX {
		e.pendingCtrlX = true
		return r, false // swallow; wait for the next key
	}

	if r == nul {
		// Ctrl-Space: insert the trigger character literally, without
		// triggering abbreviation expansion.
		e.literalNext = true
		return ' ', true
	}

	return r, true
}

// listener is readline's Config.Listener hook: called after each keypress
// that isn't a line-submitting Enter, with the buffer as it stands AFTER
// that keypress was applied. We use it for two things that both need to
// see/replace the live buffer: abbreviation expansion at the trigger key,
// and firing EditFunc once the Ctrl-X Ctrl-E chord has been observed by
// filterInputRune above (that hook can't touch the buffer itself).
func (e *Editor) listener(line []rune, pos int, key rune) ([]rune, int, bool) {
	if e.pendingEdit {
		e.pendingEdit = false
		return e.runEditFunc(line, pos)
	}

	if e.trigOK && key == e.trig && e.cfg.Abbrs != nil {
		if e.literalNext && key == ' ' {
			e.literalNext = false
			return nil, 0, false
		}
		return e.expandAbbr(line, pos)
	}

	return nil, 0, false
}

// expandAbbr checks the word immediately before the just-typed trigger
// character (which readline has already inserted at line[pos-1]) against
// the current abbreviation map, replacing it in place if it matches.
func (e *Editor) expandAbbr(line []rune, pos int) ([]rune, int, bool) {
	if pos == 0 || len(line) == 0 {
		return nil, 0, false
	}
	// line[:pos] ends with the trigger character itself; look at the word
	// before it.
	word, start := wordBeforeCursor(line, pos-1)
	if word == "" {
		return nil, 0, false
	}
	abbrs := e.cfg.Abbrs()
	expansion, ok := matchAbbr(word, abbrs)
	if !ok {
		return nil, 0, false
	}

	newLine := make([]rune, 0, len(line)+len(expansion))
	newLine = append(newLine, line[:start]...)
	newLine = append(newLine, []rune(expansion)...)
	newLine = append(newLine, line[pos-1:]...) // trigger char + rest of line
	newPos := start + len([]rune(expansion)) + 1
	return newLine, newPos, true
}

// runEditFunc invokes cfg.EditFunc with the current buffer and replaces it
// with the result. Per SPEC.md §7.5.9, the result is loaded, not executed
// — the user still has to press Enter.
func (e *Editor) runEditFunc(line []rune, _ int) ([]rune, int, bool) {
	if e.cfg.EditFunc == nil {
		return nil, 0, false
	}
	result, err := e.cfg.EditFunc(string(line))
	if err != nil {
		// Nothing sensible to do with the error here — the Listener
		// signature has no error return. Leave the buffer untouched.
		return nil, 0, false
	}
	newLine := []rune(result)
	return newLine, len(newLine), true
}

// ReadLine reads one line of input.
//
// It returns io.EOF specifically when Ctrl-D is pressed on an empty
// buffer (readline's own default behavior, confirmed in operation.go:
// the CharEOT case only returns io.EOF when the buffer is already empty;
// otherwise it deletes a character forward).
//
// Ctrl-C does NOT surface as an exit-looking error: readline returns
// readline.ErrInterrupt, having already cleared the buffer and printed a
// fresh interrupt prompt itself, so ReadLine simply loops and reads again
// — from the caller's perspective, Ctrl-C just gives you a fresh empty
// prompt.
func (e *Editor) ReadLine() (string, error) {
	return e.readLine("")
}

// ReadLineWithDefault behaves like ReadLine, but pre-fills the line buffer
// with defaultValue (cursor at the end) before the user starts typing —
// used to load, but not execute, the result of the "edit" builtin or a
// Ctrl-X Ctrl-E chord from a *previous* line into the *next* prompt (see
// Session.PendingBuffer). If defaultValue is empty this is identical to
// ReadLine.
func (e *Editor) ReadLineWithDefault(defaultValue string) (string, error) {
	return e.readLine(defaultValue)
}

func (e *Editor) readLine(defaultValue string) (string, error) {
	for {
		if e.cfg.PromptFunc != nil {
			e.inst.SetPrompt(e.cfg.PromptFunc())
		}
		if defaultValue != "" {
			e.inst.SetDefault(defaultValue)
			defaultValue = "" // only seed the very first attempt of this call
		}

		line, err := e.inst.ReadLine()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				continue
			}
			if errors.Is(err, io.EOF) {
				return "", io.EOF
			}
			return "", err
		}

		if e.trigOK && e.trig == '\r' && e.cfg.Abbrs != nil {
			// "enter" trigger: readline suppresses the Listener callback
			// on the Enter keypress itself (see operation.go's readline()
			// loop — it skips the Listener call once `result` is set), so
			// there is no live hook to intercept. Expand the trailing
			// word post-hoc instead; this is a documented limitation
			// (no visual feedback before submission) of that trigger
			// mode specifically.
			line = expandTrailingWord(line, e.cfg.Abbrs())
		}

		return line, nil
	}
}

// expandTrailingWord applies matchAbbr to the last word of line, used for
// the "enter" trigger mode (see ReadLine's comment above).
func expandTrailingWord(line string, abbrs map[string]string) string {
	runes := []rune(line)
	word, start := wordBeforeCursor(runes, len(runes))
	if word == "" {
		return line
	}
	expansion, ok := matchAbbr(word, abbrs)
	if !ok {
		return line
	}
	return string(runes[:start]) + expansion
}

// AddHistoryLine adds line to readline's in-memory recall list (Ctrl-P/
// Ctrl-R) WITHOUT writing to any file — see New()'s doc comment for why
// this is the only path by which lines enter that list. The shell.go
// caller invokes this after internal/shell.History.Add() has already
// decided the line belongs there, both when seeding a session from a
// previously-persisted history file at startup and for every new line
// entered during the session.
func (e *Editor) AddHistoryLine(line string) {
	_ = e.inst.SaveToHistory(line)
}

// Close releases the underlying terminal state.
func (e *Editor) Close() error {
	return e.inst.Close()
}
