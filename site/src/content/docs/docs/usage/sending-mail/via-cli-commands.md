---
title: Sending Mail via CLI Commands
description: Using the top-level maelsink send command with flags, attachments, JSON message specs, and raw RFC5322 piping.
---

`maelsink send` is a sendmail-equivalent SMTP client for scripting and CI: build a message from flags, a JSON spec file, or feed it a complete raw message on stdin.

![`maelsink send --help`, then sending to multiple recipients with an attachment](/maelsink/recordings/send-command.gif)

:::note
The top-level `maelsink send` command is deliberately simple — it has **no** `--count`, `--concurrency`, `--template`, `--dir`, `--recursive`, `--eml`, `--body-file`, `--json`, or `--dry-run` flags (verified directly against `cmd/send.go`'s flag registration). Those richer flags belong to `maelsink shell`'s `send` **builtin**, covered in [Sending Mail via Shell](/maelsink/docs/usage/sending-mail/via-shell/) — don't confuse the two.
:::

## Flags

```
maelsink send [--to strings] [--cc strings] [--bcc strings] [--from string]
              [--subject string] [--text string] [--html string]
              [--attach strings] [--raw] [--file string]
              [--smtp-host string] [--smtp-port int]
              [--auth-user string] [--auth-pass string]
              [--smtp-tls none|starttls|implicit]
              [--smtp-tls-insecure-skip-verify]
```

`--to`/`--cc`/`--bcc`/`--attach` are all repeatable (`stringArray`). None of `send`'s flags have single-letter shorthands.

## Composing from flags

```
$ maelsink send --to alice@example.com --from bob@example.com \
    --subject "Password reset" --text "Click here to reset"
message sent
```

`--html` sends an HTML body instead of (or alongside) `--text` — giving both produces a `multipart/alternative` body.

## Attachments

`--attach` (repeatable) attaches a file by path, base64-encoding it into a `multipart/mixed` envelope:

```
$ echo "hello attachment content" > note.txt
$ maelsink send --to dev@example.com --from app@example.com \
    --subject "With attachment" --text "see attached" --attach ./note.txt
message sent
```

Verified live — `GET /api/v1/messages?has_attachments=true` returned this message with `"attachment_count": 1`.

## `--file`: a JSON message spec

`--file` reads a JSON document shaped like `cliclient.MessageSpec` — the same shape used internally for `--to`/`--cc`/`--bcc`/`--subject`/`--text`/`--html`/`--attach`, plus a `tags` field (see [Tagging Messages](/maelsink/docs/usage/tagging-messages/)):

```json
{
  "from": "bob@example.com",
  "to": ["dana@example.com"],
  "subject": "Tagged smoke test",
  "text": "hi",
  "tags": ["smoke", "release"]
}
```

```
$ maelsink send --file tagged.json
message sent
```

Live-verified: the message arrived with `"tags": ["smoke", "release"]` in the API response. `attachments` in the JSON spec takes a list of `{"path": "...", "filename": "..."}` objects, same as `--attach`.

## `--raw`: piping a full RFC 5322 message via stdin

`--raw` reads a complete message — headers and all — from stdin and sends it verbatim. The envelope (`MAIL FROM`/`RCPT TO`) is derived from the message's own `From`/`To`/`Cc`/`Bcc` headers, so there's no separate `--to`/`--from` needed (and they're ignored in this mode):

```
$ printf 'From: raw@example.com\r\nTo: dev@example.com\r\nSubject: Raw RFC5322 test\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nThis is a raw message piped via --raw.\r\n' \
  | maelsink send --raw
message sent
```

Live-verified — the message arrived with subject `Raw RFC5322 test` and `From: raw@example.com`, both taken from the piped headers.

## Pointing at a different server / transport security

`--smtp-host`/`--smtp-port` override the default `127.0.0.1:1025` target. `--smtp-tls none|starttls|implicit` selects transport security (default `none`), with `--smtp-tls-insecure-skip-verify` to accept a self-signed/dev cert when testing against `smtp.tls_cert`/`smtp.tls_key`. `--auth-user`/`--auth-pass` supply SMTP AUTH PLAIN credentials when the target has `smtp.auth.enabled=true`.

```
maelsink send --smtp-host 127.0.0.1 --smtp-port 1025 \
  --to dev@example.com --from app@example.com --subject hi --text hi
```
