---
title: TLS Certificates
description: Configuring web.tls.cert/key and smtp.tls_cert/tls_key for local dev, plus the reverse-proxy alternative.
---

maelsink can terminate TLS itself on either the Web UI port or the SMTP port, using a
supplied PEM certificate/key pair. There is no built-in CA or auto-generated certificate;
the files must be supplied.

## Generating a local dev cert

A self-signed cert is enough for local dev/CI — you're not trying to fool a real client,
just get TLS negotiation working:

```sh
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout dev.key -out dev.crt -days 365 \
  -subj "/CN=maelsink.local"
```

This produces `dev.crt` and `dev.key` in the current directory, valid for 365 days, with
no passphrase (`-nodes`) so maelsink can read the key without prompting.

## Wiring it up

For the SMTP listener (enables STARTTLS):

```sh
maelsink serve --smtp-tls-cert ./dev.crt --smtp-tls-key ./dev.key
```

For the Web UI (serves the UI and REST API over HTTPS):

```sh
maelsink serve --web-tls-cert ./dev.crt --web-tls-key ./dev.key
```

Both cert and key must be supplied together — setting only one is treated as
misconfiguration. See [Runtime options](/maelsink/docs/configuration/runtime-options/) for the
corresponding `smtp.tls_cert`/`smtp.tls_key` and `web.tls.cert`/`web.tls.key` config
keys and their env vars, and
[SMTP server](/maelsink/docs/configuration/smtp-server/) for `smtp.require_starttls`/
`smtp.require_tls`.

## What a self-signed cert means for clients

- **Browsers** hitting the Web UI over HTTPS will show a certificate-warning
  interstitial ("Your connection is not private" or similar) — click through it for
  local dev, or import the cert into your OS/browser trust store if you want to avoid
  the warning.
- **Mail clients / SMTP libraries** connecting over STARTTLS or implicit TLS will
  usually refuse the connection by default unless you explicitly disable certificate
  verification. Most SMTP libraries expose exactly this as a debug/insecure flag —
  maelsink's own client tooling does too (`--smtp-tls-insecure-skip-verify` on `send`
  and the shell's SMTP-backed builtins).

## Alternative: terminate TLS at a reverse proxy

Rather than configuring TLS in maelsink itself, you can run maelsink in plain HTTP/SMTP
mode behind a reverse proxy (nginx, Caddy, HAProxy, a cloud load balancer) that
terminates TLS and forwards plaintext to maelsink. This is often simpler in CI/container
environments where the proxy already manages certificates. See
[Running maelsink behind HAProxy](/maelsink/docs/guides/haproxy/) for a worked example — note that
proxying the SMTP port requires TCP-mode (not HTTP-mode) proxying, since SMTP isn't
HTTP.
