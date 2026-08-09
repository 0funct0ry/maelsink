package lineedit

// This file documents how each of SPEC.md §7.5.9's key bindings maps onto
// github.com/ergochat/readline's actual behavior, verified by reading
// operation.go in github.com/ergochat/readline@v0.1.3.
//
//   - Ctrl-X Ctrl-E: not a binding readline knows about natively (it has no
//     concept of emacs-style two-key chords beyond what's hardcoded for
//     escape sequences). Implemented in editor.go via a small state machine
//     split across FuncFilterInputRune (swallows the bare Ctrl-X so it
//     never becomes a literal control byte in the buffer) and Listener
//     (fires EditFunc once the second key of the chord, Ctrl-E, is
//     observed). See editor.go's filterInputRune/listener/runEditFunc.
//
//   - Tab: readline's own CharTab case already drives AutoComplete.Do
//     to completer.go's Completer — no custom binding needed.
//
//   - Trigger key (space/tab/enter) for abbreviation expansion: handled in
//     editor.go's listener for space/tab (Listener sees the buffer with
//     the trigger character already inserted); "enter" is a post-hoc
//     rewrite in Editor.ReadLine because readline suppresses the Listener
//     callback on the Enter keypress that submits the line (see
//     operation.go's readline() loop: the Listener call is skipped once
//     `result` — the line being returned — has been set).
//
//   - Ctrl-Space: readline has no built-in concept of it. On the
//     terminals checked, Ctrl-Space arrives as a NUL (0) rune.
//     filterInputRune translates NUL to a plain space and sets a
//     one-shot flag that the listener checks to skip expansion for that
//     specific keypress, so it inserts a literal trigger character
//     without expanding.
//
//   - Ctrl-C: readline's own CharInterrupt case already clears the buffer,
//     prints its InterruptPrompt, and returns readline.ErrInterrupt from
//     ReadLine — it does NOT exit. Editor.ReadLine (editor.go) simply
//     loops and calls readline.Instance.ReadLine again on that error, so
//     from every caller's perspective Ctrl-C never surfaces as an error at
//     all, matching "discard the current line and show a fresh prompt —
//     does not exit". No additional binding is needed; this file exists to
//     record that the default behavior already satisfies the spec.
//
//   - Ctrl-D: readline's own CharEOT case already returns io.EOF only when
//     the buffer is empty (deleting a character forward otherwise) — see
//     operation.go's MetaDeleteKey/CharEOT case. Again, no custom binding
//     needed; Editor.ReadLine passes io.EOF straight through.
