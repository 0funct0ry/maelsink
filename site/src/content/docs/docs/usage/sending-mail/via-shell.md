---
title: Sending Mail via Shell
description: The maelsink shell's richer send builtin — templates, bulk sends, dry-run — plus randmsg/intmsg/weirdmsg/blast/deluge for generating test traffic.
---

`maelsink shell`'s `send` builtin (`internal/shell/builtin/send.go`) is the richest way to send test mail: it supports every body-source mode from `maelsink send` plus templating, bulk sending with per-message re-rendering, and a dry-run mode. It's a strict superset of the top-level `maelsink send` command's flags, not the same flag set.

## Flags

```
send [--from string] [--to strings] [--cc strings] [--bcc strings]
     [--subject string] [--text string] [--html string]
     [--attach strings] [--dir string] [--recursive]
     [--eml string] [--template string] [--body-file string] [--json string]
     [--count int] [--concurrency int] [--dry-run]
     [--smtp-host string] [--smtp-port int]
     [--auth-user string] [--auth-pass string]
     [--smtp-tls none|starttls|implicit] [--smtp-tls-insecure-skip-verify]
```

`--eml`, `--template`, `--body-file`, and `--json` are mutually exclusive "primary body source" modes — pick at most one. With none of them, `send` builds the message from `--from`/`--to`/`--subject`/`--text`/`--html`/`--attach` flags directly (same shape as the top-level command).

## Starting a shell session

Interactively:

```
$ maelsink shell
maelsink> send --to dev@example.com --from app@example.com --subject hi --text hi
sent 1 message
```

Or scripted with `-e` (repeatable) against a specific target:

```
maelsink shell --api http://127.0.0.1:9090 --smtp-port 1025 \
  -e 'send --to dev@example.com --from app@example.com --subject hi --text hi'
```

## `--count` / `--concurrency`: bulk sends

`--count N` sends N messages; `--concurrency M` bounds how many SMTP connections run in parallel. Each message gets its own fresh template render, so content that depends on `{{.n}}`/`{{.count}}`/random template functions varies per message — but **only when that templated content comes from a file** (`--template` or `--body-file`), since those are re-read and re-rendered on every iteration. Putting `{{ }}` expressions directly in `--subject`/`--text` on the command line doesn't get per-message variance, because the shell's own line-level template expansion resolves them once, before `send` ever sees the flags.

For variance across a batch, put the template expression in a `--body-file`:

```
$ cat body.tmpl
Message {{.n}} of {{.count}}: your code is {{ randInt 1000 9999 }}

$ maelsink shell -e 'send --to test@example.com --from me@example.com --subject batch --body-file body.tmpl --count 3'
3/3 sent
```

Live-verified: the three resulting messages had previews `Message 1 of 3: your code is 5541`, `Message 2 of 3: your code is 1499`, `Message 3 of 3: your code is 9808` — distinct per message, proving per-message re-rendering.

`.n` is the 1-based message index and `.count` is the total, available inside any per-message template render (`--template`/`--body-file`/`--json`).

## `--dry-run`: render without sending

```
$ maelsink shell -e 'send --to test@example.com --from me@example.com --subject "dry run test" --text hi --dry-run'
--- dry run (1 of 1 shown): me@example.com -> [test@example.com] ---
From: me@example.com
To: test@example.com
Subject: dry run test
Date: Mon, 17 Aug 2026 09:25:41 +0000
MIME-Version: 1.0
Content-Type: text/plain; charset=utf-8

hi
```

Live-verified — no message was actually delivered; only the rendered RFC 5322 output was printed.

## `--json`: templated `MessageSpec` file

Like `--file` on the top-level command, but every string field is rendered through the template engine per message, so `{{ }}` expressions inside the JSON *do* get per-message variance under `--count`:

```json
{
  "from": "bob@example.com",
  "to": ["dana@example.com"],
  "subject": "Batch {{.n}} of {{.count}}",
  "text": "hi"
}
```

```
maelsink shell -e 'send --json spec.json --count 5'
```

## `--eml` and `--template`

`--eml <file>` sends the file verbatim as a complete RFC 5322 message — never templated, never re-rendered per count. `--template <file>` is the templated equivalent: the whole file is rendered as a complete message per iteration, and its own `From`/`To` headers (after rendering) determine the SMTP envelope, unless overridden by `--from`/`--to` flags.

## `--dir` / `--recursive`: attaching a whole directory

`--dir path` attaches every regular file in that directory; add `--recursive` to also walk subdirectories. Combine with `--attach` — both contribute to the same attachment list.

## Related builtins for generating test traffic

Alongside `send`, the shell ships builtins purpose-built for load/edge-case testing, all taking the same SMTP override flags (`--smtp-host`, `--smtp-port`, `--auth-user`, `--auth-pass`, `--smtp-tls`, `--smtp-tls-insecure-skip-verify`) as `send`:

| Builtin | Purpose |
|---|---|
| `randmsg` | Send one randomly-generated message |
| `intmsg` | Send random messages at randomized intervals |
| `weirdmsg` | Send one message of an awkward, edge-case shape (malformed/unusual MIME) |
| `blast` | Send one message to many generated recipients |
| `deluge` | Fire N random messages at maximum throughput |

Live example:

```
$ maelsink shell -e 'randmsg' -e 'list'
1/1 sent
ID                        FROM               TO                         SUBJECT                                 SIZE  ...
e9a0154726eacb3ca670e5bd  cguzman@ramos.net  berta.fisher@lindgren.net  Design for failure and graceful child.  513   ...
```

`randmsg`'s From/To/subject/body all came from maelsink's own Faker-style template functions (see the [Shell Functions Reference](/maelsink/docs/shell-functions-reference/)) — useful for populating an inbox with realistic-looking noise without writing any templates yourself.
