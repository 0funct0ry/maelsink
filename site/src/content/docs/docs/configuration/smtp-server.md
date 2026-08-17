---
title: SMTP Server
description: The smtp.* config keys in context — port, size limits, and STARTTLS/require modes.
---

The SMTP listener is what your application actually sends mail to. For the full
key/default/env/flag table, see [Runtime options](/maelsink/docs/configuration/runtime-options/) —
this page explains the `smtp.*` keys in practical terms.

## Host, port, domain

```sh
maelsink serve --smtp-host 0.0.0.0 --smtp-port 2525 --smtp-domain test.local
```

`smtp.domain` is the hostname maelsink announces in its SMTP `EHLO`/`HELO` banner and
uses when generating `Message-ID` headers — it doesn't need to resolve to anything real.

## Message size limits

`smtp.max_message_size_mb` (default `25`) caps the size of an incoming message,
including attachments. Messages over this limit are rejected during the SMTP `DATA`
phase rather than silently truncated.

## STARTTLS and TLS enforcement

Three related settings control transport security on the SMTP port:

- `smtp.tls_cert` / `smtp.tls_key` — paths to a certificate/key pair. If set, the server
  advertises STARTTLS. See [TLS certificates](/maelsink/docs/configuration/tls-certificates/) for
  generating a local dev cert.
- `smtp.require_starttls` — reject any client that doesn't upgrade to TLS via STARTTLS
  before authenticating or sending mail.
- `smtp.require_tls` — require TLS from the start of the connection (implicit TLS),
  rather than allowing a plaintext-then-STARTTLS upgrade.

Both `require_*` flags need `smtp.tls_cert`/`smtp.tls_key` configured — enabling either
one without a cert/key pair will fail to start.

## SMTP AUTH

If you want to require authenticated senders (e.g. to simulate a real relay's auth
gate), see [Password files](/maelsink/docs/configuration/password-files/) for `smtp.auth.file`,
and the `smtp.auth.accept_any` / `MAELSINK_SMTP_AUTH` shortcuts for CI use.
