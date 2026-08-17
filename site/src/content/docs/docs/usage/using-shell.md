---
title: Using Shell
description: A detailed guide to maelsink shell — readline editing, history, aliases/abbreviations, redirection, templating, the editor integration, and scripting.
---

`maelsink shell` is an interactive/scriptable REPL over the same REST API and SMTP port used everywhere else in maelsink — it's a pure client, never touching storage directly, and every local (non-API) builtin works even with no server reachable at all.

```
maelsink shell [flags]
```

## Readline editing and history

The shell uses a readline-style line editor (`internal/shell/lineedit`) with standard Emacs-style editing (cursor movement, kill/yank, etc.) plus:

- **`--history-file string`** (`-Y`) — path to the persisted history file (default: a platform-appropriate location). Written with `0600` permissions.
- **`--history-size int`** (`-z`, default `5000`) — max lines retained.
- Adjacent-duplicate lines are not persisted twice.
- **Secrets are never written to disk**: any history line containing `--api-key`, `--auth-pass`, or `--auth-user` (`internal/shell/history.go`'s `redactedFlags`) is kept in the in-memory session history (so you can still recall it with arrow-up during the session) but is filtered out before the history file is written.

The `history` builtin shows numbered command history and supports `--clear`/`--search`.

## Aliases and abbreviations

- **`alias <name> <expansion>`** — defines a textual alias, expanded whenever `<name>` appears as the first word of a line. `unalias <name>` removes one; both support `--global` (session-wide vs. persisted) and `--erase`/`--all`.
- **`abbr <trigger> <expansion>`** — like an alias, but expands as soon as it's followed by the configured trigger key rather than only at the start of a line. Controlled by **`--abbr-trigger-key string`** (`-G`, default `space`; also accepts `tab`, `enter`, or `none` to disable trigger-key expansion entirely). `unabbr` removes one.

```
maelsink> alias ll "list --limit 5"
maelsink> ll
```

Live-verified — after defining the alias above in a scripted session, invoking `ll` ran `list --limit 5` correctly.

## Redirection

The shell supports shell-like output redirection on any command, confirmed directly in `internal/shell/eval.go`'s `SplitRedirection`:

- `command > file` — truncates `file` and writes the command's output there.
- `command >> file` — appends to `file`.

A bare `>`/`>>` with no trailing path, or one appearing anywhere but as the second-to-last token, is a parse error rather than being silently ignored.

```
maelsink> list > messages.txt
maelsink> stats >> report.log
```

## Templating

Every command line goes through `{{ }}` Go-template expansion before being tokenized (unless disabled), backed by a large FuncMap covering identifiers (`uuid`, `ulid`, `nanoid`, ...), random generators, a full Faker-style set (`fName`, `fEmail`, `fCompany`, `fCreditCard`, binary file generators like `fPNG`/`fPDF`/`fXLSX`, ...), curated Sprig string/date/encoding helpers, and ANSI color helpers. See the [Shell Functions Reference](/maelsink/docs/shell-functions-reference/) for the complete list.

- **`--seed int64`** (`-S`, default `0` = random per session) — seeds the template engine's PRNG. A fixed seed makes every random template function deterministic and reproducible across runs — useful for golden-file tests or reproducing a specific generated message.
- **`--template-enabled`** (`-t`, default `true`) / **`--no-template`** (`-N`, shorthand for `--template-enabled=false`) — turn expansion off entirely (e.g. to send literal `{{ }}` text without it being interpreted).
- Escape a literal `{{` with `\{{` when you need it to survive expansion.
- **`--template-unsafe-funcs`** (`-Z`, default `false`) — three functions (`env`, `expandenv`, `getHostByName`) are absent from the FuncMap unless this flag is set, since they can leak host environment/DNS information into a generated message.

```
maelsink> echo {{ uuidv7 }} {{ fEmail }}
maelsink> template {{ upper "hi" }}
HI
```

Live-verified — `template` ran fully offline (target API unreachable) and printed `HI`, since templating is a purely local operation with no server dependency.

## Editor integration

The `edit` builtin and the **Ctrl-X Ctrl-E** keybinding (confirmed in `internal/shell/lineedit/editor.go` and `keys.go` — Ctrl-X is intercepted to detect the chord, since readline has no native binding for it) both write the current buffer to a temp file, restore the terminal to cooked mode, exec the resolved editor (`shell.editor` / `$VISUAL` / `$EDITOR` / `vi`, or `notepad` on Windows), wait for it to exit, and read the result back.

This is deliberately **load, don't execute** — unlike bash's edit-and-run-command behavior, the edited text is loaded into the next prompt's line buffer (interactive) or printed (non-interactive), never run automatically. `edit -f/--file <path>` instead edits a file in place and confirms the path, without touching the prompt buffer.

## Scripting: `-e`, `--script`, and piped stdin

Three non-interactive modes, all routed through the same evaluator as the interactive REPL, with readline/history/abbreviation-expansion switched off:

```
maelsink shell -e 'list --limit 5' -e 'stats'
maelsink shell --script commands.txt
echo 'list' | maelsink shell
```

- **`-e`/`--execute string`** (repeatable) — run each given command in order, then exit.
- **`--script string`** — run a file of commands, one per line, then exit.
- Piped stdin with no `-e`/`--script` and no TTY runs each line the same way.
- **`--exit-on-error`** (`-Q`, default `false`) — abort the whole script on the first failing command instead of continuing.
- Exit code is the last command's status.

## Searching and filtering

The `list` builtin (alias `ls`) mirrors `maelsink list`'s flags exactly, since both are thin clients over the same `GET /api/v1/messages` endpoint:

- `-q, --query` — full-text search query, supporting the same FTS5 syntax as the Web UI search bar (see [Advanced Search Patterns](/maelsink/docs/usage/advanced-search-patterns/))
- `--from`, `--to`, `--subject` — substring filters against the corresponding header
- `--since`, `--until` — RFC3339 date-range bounds, inclusive at both ends
- `--sort` — `received_at_desc` (default) or `received_at_asc`
- `-n, --limit`, `--offset` — pagination
- `--ids` — print matching message IDs only, one per line
- `--format` — output as `table` (default), `json`, or `yaml`

See the [Shell Builtin Reference](/maelsink/docs/shell-builtin-reference/) for the complete, generated flag list.

```
maelsink> list --q="(receipt OR invoice) AND acme"
maelsink> list --from=bob --since=2026-08-17T00:00:00Z --sort=received_at_asc
```

Filters compose the same way as `maelsink list` and the REST API: every flag is combined with AND. Aliases and abbreviations (above) apply to `list` invocations as well, so a frequently repeated filter combination can be bound to a short name for the session, or persisted with `--global`. See [Using CLI](/maelsink/docs/usage/using-cli/) for the full set of filters, including the REST-only parameters (`tag`, `read`, `has_attachments`, `parse_warning`) not yet exposed as `list` flags.

## Offline / local-only operation

The shell starts and runs every local builtin (`set`, `alias`, `template`, `config`, `sh`, ...) with **no reachable server at all** — only API-backed builtins (`list`, `send`, `stats`, ...) fail, and they fail distinguishably:

```
$ maelsink shell --api http://127.0.0.1:19099 -e 'list'
Error: error: could not reach maelsink API at http://127.0.0.1:19099: dial tcp 127.0.0.1:19099: connect: connection refused
```

That "could not reach maelsink API" wording is distinct from an HTTP-level error (like a 404), so scripts can tell "server unreachable" apart from "server responded with an error" on captured stderr. The default prompt template (`maelsink{{ if not .connected }} (offline){{ end }}> `) reflects this live via the reserved `connected` variable.
