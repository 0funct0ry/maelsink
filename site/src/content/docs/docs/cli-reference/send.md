---
title: "send"
description: "Compose and send a test message to a maelsink instance over SMTP."
---

Compose and send a test message to a maelsink instance over SMTP.

This page is generated directly from `maelsink send --help`. The flag list below is always in sync with the binary and is never hand-maintained.

```
A sendmail-equivalent SMTP client for scripting/CI: send via flags, a raw RFC 5322 message on stdin (--raw), or a JSON message spec (--file).

Usage:
  maelsink send [flags]

Flags:
      --attach stringArray              path to a file to attach (repeatable)
      --auth-pass string                SMTP AUTH password
      --auth-user string                SMTP AUTH username
      --bcc stringArray                 bcc address (repeatable)
      --cc stringArray                  cc address (repeatable)
      --file string                     path to a JSON message spec to send
      --from string                     from address
  -h, --help                            help for send
      --html string                     HTML body
      --raw                             read a full RFC 5322 message from stdin and send it verbatim
      --smtp-host string                SMTP server host (default "127.0.0.1")
      --smtp-port int                   SMTP server port (default 1025)
      --smtp-tls string                 transport security: none|starttls|implicit (default "none")
      --smtp-tls-insecure-skip-verify   accept a self-signed/dev SMTP TLS certificate without verification (local/CI use only)
      --subject string                  message subject
      --text string                     plain text body
      --to stringArray                  recipient address (repeatable)
```
