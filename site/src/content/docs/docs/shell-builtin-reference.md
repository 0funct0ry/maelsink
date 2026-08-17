---
title: "Shell Builtin Reference"
description: "Every builtin command in the maelsink interactive shell, generated from the builtin registry."
---

This page is generated from `internal/shell/builtin`'s registry (`go run ./tools/docgen`) — the flag list for each builtin is always in sync with the shell binary; it is never hand-maintained.

## abbr

Local-only — define an abbreviation, expanded on a trigger key.

- `--erase` (default `false`) — remove the named abbreviation
- `--global` (default `false`) — substitute anywhere on the line, not just the first word

## alias

Local-only — define a command alias.

- `--erase` (default `false`) — remove the named alias
- `--global` (default `false`) — substitute anywhere on the line, not just the first word

## attachment

Aliases: `att`

GET /api/v1/messages/:id/attachments/:attId — download an attachment.

- `--all` (default `false`) — download every attachment into the --out directory
- `-o, --out` — output path (file, or directory with --all)
- `--stdout` (default `false`) — write raw bytes to stdout

## blast

Sends a burst of generated messages over SMTP (same SMTP override flags as send).

- `-A, --attachment-size` (default `10KB`) — size of each generated attachment
- `-a, --attachments` (default `0`) — number of generated attachments to include
- `-W, --auth-pass` — override SMTP AUTH password for this invocation
- `-U, --auth-user` — override SMTP AUTH username for this invocation
- `-C, --bcc` — Bcc address (repeatable)
- `-B, --body` (default `random`) — body source: text|html|both|random
- `-c, --cc` — Cc address (repeatable)
- `-d, --dry-run` (default `false`) — render but do not send; print the result
- `-f, --from` — From address (default: a fake email)
- `-r, --recipients` (default `10`) — number of generated recipients
- `-S, --scenario` — seed subject/body from a canned example scenario (see: example --list)
- `-H, --smtp-host` — override the session's SMTP host for this invocation
- `-P, --smtp-port` (default `0`) — override the session's SMTP port for this invocation
- `--smtp-tls` — override transport security for this invocation: none|starttls|implicit
- `--smtp-tls-insecure-skip-verify` (default `false`) — accept a self-signed/dev SMTP TLS certificate without verification for this invocation
- `-x, --split` (default `to`) — recipient distribution: to|cc|bcc|mixed
- `-s, --subject` — Subject (default: a fake subject, or the scenario's)
- `-T, --tags` — tag to attach to the message (repeatable)
- `-t, --to` — To address (default: a fake email)

## clear

DELETE /api/v1/messages?confirm=true — delete every stored message.

- `-y, --yes` (default `false`) — skip confirmation

## config

Local-only — get/set/list/save shell session config.

- `--force` (default `false`) — on `save`, create the file if it does not already exist
- `--format` (default `table`) — output format for `list`: table|json|yaml

## delete

Aliases: `rm`, `del`

DELETE /api/v1/messages/:id, or clear's bulk path with --all.

- `--all` (default `false`) — delete every message (same as clear)
- `-y, --yes` (default `false`) — skip confirmation

## deluge

Sends a sustained stream of generated messages over SMTP (same SMTP override flags as send).

- `-A, --attachment-size` (default `10KB`) — size of each generated attachment
- `-a, --attachments` (default `0`) — number of generated attachments to include
- `-W, --auth-pass` — override SMTP AUTH password for this invocation
- `-U, --auth-user` — override SMTP AUTH username for this invocation
- `-C, --bcc` — Bcc address (repeatable)
- `-B, --body` (default `random`) — body source: text|html|both|random
- `-c, --cc` — Cc address (repeatable)
- `-j, --concurrency` (default `10`) — max parallel SMTP connections
- `-n, --count` (default `100`) — number of messages to send
- `-f, --from` — From address (default: a fake email)
- `-S, --scenario` — seed subject/body from a canned example scenario (see: example --list)
- `-H, --smtp-host` — override the session's SMTP host for this invocation
- `-P, --smtp-port` (default `0`) — override the session's SMTP port for this invocation
- `--smtp-tls` — override transport security for this invocation: none|starttls|implicit
- `--smtp-tls-insecure-skip-verify` (default `false`) — accept a self-signed/dev SMTP TLS certificate without verification for this invocation
- `-s, --subject` — Subject (default: a fake subject, or the scenario's)
- `-T, --tags` — tag to attach to the message (repeatable)
- `-t, --to` — To address (default: a fake email)

## echo

Local-only — prints text back to the shell.

- `-n, --no-newline` (default `false`) — don't print the trailing newline

## edit

Local-only — open the last command (or a script) in $VISUAL/$EDITOR.

- `-f, --file` — edit this file directly, in place (default: edit scratch text for the prompt buffer)

## example

Local-only — emits a canned example message (eml or json).

- `--format` (default `eml`) — output format: eml|json
- `--index` (default `0`) — pick a specific canned example (1-based; default: random)
- `--list` (default `false`) — list the canned examples instead of generating one
- `-o, --out` — output path (default: a generated path under the session's temp dir)

## exit

Aliases: `quit`

Local-only — exit the shell.

_No flags._

## export

GET /api/v1/messages/:id/export, or a bulk zip export with --all.

- `--all` (default `false`) — bulk export every message (filtered by list's flags) as a .zip
- `--from` — filter by from address substring (with --all)
- `-o, --out` — output path: a file for one message, a directory for several
- `-q, --query` — full-text search query (with --all)
- `--since` — only messages received at/after this RFC3339 timestamp (with --all)
- `--sort` — sort order (with --all)
- `--subject` — filter by subject substring (with --all)
- `--to` — filter by to address substring (with --all)
- `--until` — only messages received at/before this RFC3339 timestamp (with --all)
- `--zip` (default `false`) — bundle multiple exports into one .zip

## functions

Aliases: `fns`, `funcs`

Local-only — lists the template FuncMap registry (see the functions reference below).

- `-s, --search` — case-insensitive substring match against name or description

## health

GET /api/v1/health — health check.

- `--format` (default `table`) — output format: table|json|yaml

## help

Aliases: `?`

Local-only — show help for a builtin.

_No flags._

## history

Aliases: `hist`

Local-only — inspect or clear shell command history.

- `--clear` (default `false`) — clear session history
- `-e, --edit` (default `0`) — open history entry <num> (as numbered by a plain 'history') in $EDITOR — same load-don't-execute behavior as the "edit" builtin
- `-n, --limit` (default `0`) — show only the last N entries (0 = all)
- `--search` — only show entries containing this substring

## intmsg

Generates and sends an "interesting" message over SMTP (same SMTP override flags as send).

- `-A, --attachment-size` (default `10KB`) — size of each generated attachment
- `-a, --attachments` (default `0`) — number of generated attachments to include
- `-W, --auth-pass` — override SMTP AUTH password for this invocation
- `-U, --auth-user` — override SMTP AUTH username for this invocation
- `-g, --background` (default `false`) — run detached from the prompt; prints a job id, usable with --stop
- `-C, --bcc` — Bcc address (repeatable)
- `-B, --body` (default `random`) — body source: text|html|both|random
- `-b, --burst-interval` (default `100ms`) — spacing between messages within a burst, for --profile bursty
- `-k, --burst-size` (default `5`) — messages per burst, for --profile bursty
- `-c, --cc` — Cc address (repeatable)
- `-n, --count` (default `0`) — stop after this many messages (0: unbounded)
- `-d, --duration` (default `0s`) — stop after this long (0: unbounded)
- `-f, --from` — From address (default: a fake email)
- `-i, --interval` (default `1s`) — mean inter-message gap (or the quiet-period gap for --profile bursty)
- `-j, --jitter` (default `0`) — jitter around --interval: a duration (e.g. 200ms) or a percentage (e.g. 20%)
- `-l, --list` (default `false`) — list background intmsg runs and their live status
- `-p, --profile` (default `steady`) — interval distribution: steady|poisson|bursty
- `-q, --quiet` (default `false`) — suppress per-message send confirmations
- `-r, --rate` (default `0`) — mean messages per second (alternative to --interval)
- `-S, --scenario` — seed subject/body from a canned example scenario (see: example --list)
- `-H, --smtp-host` — override the session's SMTP host for this invocation
- `-P, --smtp-port` (default `0`) — override the session's SMTP port for this invocation
- `--smtp-tls` — override transport security for this invocation: none|starttls|implicit
- `--smtp-tls-insecure-skip-verify` (default `false`) — accept a self-signed/dev SMTP TLS certificate without verification for this invocation
- `-I, --stats-interval` (default `5s`) — how often to print a running summary line
- `-X, --stop` — stop a --background run by job id (from its startup message) and print its summary
- `-s, --subject` — Subject (default: a fake subject, or the scenario's)
- `-T, --tags` — tag to attach to the message (repeatable)
- `-t, --to` — To address (default: a fake email)
- `-e, --until-error` (default `false`) — stop on the first SMTP failure instead of logging and continuing

## list

Aliases: `ls`

GET /api/v1/messages — list/filter stored messages.

- `--format` (default `table`) — output format: table|json|yaml
- `--from` — filter by from address substring
- `--ids` (default `false`) — print IDs only, one per line
- `-n, --limit` (default `50`) — max messages to return (max 500)
- `--offset` (default `0`) — pagination offset
- `-q, --query` — full-text search query
- `--since` — only messages received at/after this RFC3339 timestamp
- `--sort` — sort order: received_at_desc|received_at_asc
- `--subject` — filter by subject substring
- `--to` — filter by to address substring
- `--until` — only messages received at/before this RFC3339 timestamp

## prompt

Local-only — inspects or resets the shell prompt state.

- `--reset` (default `false`) — restore the built-in default prompt

## randmsg

Generates and sends a random message over SMTP (same SMTP override flags as send).

- `-A, --attachment-size` (default `10KB`) — size of each generated attachment
- `-a, --attachments` (default `0`) — number of generated attachments to include
- `-W, --auth-pass` — override SMTP AUTH password for this invocation
- `-U, --auth-user` — override SMTP AUTH username for this invocation
- `-C, --bcc` — Bcc address (repeatable)
- `-B, --body` (default `random`) — body source: text|html|both|random
- `-c, --cc` — Cc address (repeatable)
- `-j, --concurrency` (default `1`) — max parallel SMTP connections
- `-n, --count` (default `1`) — number of messages to send
- `-d, --dry-run` (default `false`) — render but do not send; print the result
- `-f, --from` — From address (default: a fake email)
- `-S, --scenario` — seed subject/body from a canned example scenario (see: example --list)
- `-H, --smtp-host` — override the session's SMTP host for this invocation
- `-P, --smtp-port` (default `0`) — override the session's SMTP port for this invocation
- `--smtp-tls` — override transport security for this invocation: none|starttls|implicit
- `--smtp-tls-insecure-skip-verify` (default `false`) — accept a self-signed/dev SMTP TLS certificate without verification for this invocation
- `-s, --subject` — Subject (default: a fake subject, or the scenario's)
- `-T, --tags` — tag to attach to the message (repeatable)
- `-t, --to` — To address (default: a fake email)

## send

Sends directly over SMTP via the shell's SMTP client (cliclient.SendTLS) — not a REST call.

- `--attach` — attach a file (repeatable; `::`-joined paths from the attach template func are split)
- `--auth-pass` — override SMTP AUTH password for this invocation
- `--auth-user` — override SMTP AUTH username for this invocation
- `--bcc` — Bcc address (repeatable)
- `--body-file` — render this file as the message body; headers come from flags
- `--cc` — Cc address (repeatable)
- `--concurrency` (default `1`) — max parallel SMTP connections
- `--count` (default `1`) — number of messages to send
- `--dir` — attach every regular file in this directory
- `--dry-run` (default `false`) — render but do not send; print the result(s)
- `--eml` — send this file verbatim as RFC 5322 (not templated)
- `--from` — From address
- `--html` — HTML body
- `--json` — read a cliclient.MessageSpec-shaped JSON file, templating its string fields
- `--recursive` (default `false`) — with --dir, walk subdirectories too
- `--smtp-host` — override the session's SMTP host for this invocation
- `--smtp-port` (default `0`) — override the session's SMTP port for this invocation
- `--smtp-tls` — override transport security for this invocation: none|starttls|implicit
- `--smtp-tls-insecure-skip-verify` (default `false`) — accept a self-signed/dev SMTP TLS certificate without verification for this invocation
- `--subject` — Subject
- `--template` — render this file's full RFC 5322 message via the template engine
- `--text` — plain-text body
- `--to` — To address (repeatable)

## set

Local-only — set a shell variable.

- `--format` (default `table`) — output format for bare `set` (same as vars): table|json|yaml
- `--from-command` — run a builtin ("<name> [args...]") and capture its stdout into the variable
- `--global` (default `false`) — also persist this variable on the next `config save`

## sh

Local-only — run a raw shell command.

- `--quiet` (default `false`) — suppress output; exit status still lands in last_error

## show

Aliases: `get`

GET /api/v1/messages/:id — show one message's detail/body/headers.

- `--format` (default `table`) — output format: table|json|yaml
- `-o, --out` — write the selected part to this file instead of stdout
- `--part` (default `all`) — part to show: html|text|raw|headers|all

## source

Aliases: `.`

Local-only — execute a script file in the current session.

- `--quiet` (default `false`) — suppress per-line echo

## stats

GET /api/v1/stats — server stats (message count, storage size, uptime).

- `--format` (default `table`) — output format: table|json|yaml

## template

Aliases: `tmpl`

Local-only — inspect template settings/registry (--funcs).

- `-f, --file` — read the template from this file instead of the positional arg
- `--funcs` (default `false`) — list every registered template function name
- `--seed` (default `0`) — one-shot seed override for this render only

## unabbr

Local-only — remove an abbreviation.

- `--all` (default `false`) — remove every abbreviation

## unalias

Local-only — remove a command alias.

- `--all` (default `false`) — remove every alias

## unset

Local-only — unset a shell variable.

_No flags._

## vars

Local-only — list shell variables.

- `--format` (default `table`) — output format: table|json|yaml

## version

GET /api/v1/version, or local build info with --local.

- `--format` (default `table`) — output format: table|json|yaml
- `--local` (default `false`) — skip the API call, print the shell binary's own build info

## weirdmsg

Generates and sends a deliberately malformed message over SMTP (same SMTP override flags as send).

- `-W, --auth-pass` — override SMTP AUTH password for this invocation
- `-U, --auth-user` — override SMTP AUTH username for this invocation
- `-d, --depth` (default `5`) — chain length for --kind thread
- `-f, --from` — From address (default: a fake email)
- `-k, --kind` (default `random`) — bounce|malformed|huge|unicode|spoof|thread|invite|random
- `-s, --size` (default `10MB`) — body size for --kind huge
- `-H, --smtp-host` — override the session's SMTP host for this invocation
- `-P, --smtp-port` (default `0`) — override the session's SMTP port for this invocation
- `--smtp-tls` — override transport security for this invocation: none|starttls|implicit
- `--smtp-tls-insecure-skip-verify` (default `false`) — accept a self-signed/dev SMTP TLS certificate without verification for this invocation
- `-t, --to` — To address (default: a fake email)

