# CLI/shell SMTP client TLS & auth

`maelsink send` and every shell builtin that sends mail (`send`, `intmsg`,
`randmsg`, `weirdmsg`, `blast`, `deluge`) can now speak TLS to the SMTP
listener, so the CLI can exercise a `maelsink serve` instance started with
STARTTLS/implicit-TLS or SMTP AUTH hardening (see
[SPEC.md §4](../internal-docs/SPEC.md) for the server-side flags this
client-side support pairs with).

## `--smtp-tls` modes

| Value | Behavior |
|---|---|
| `none` (default) | Plain TCP, no transport security — unchanged from before this support existed. |
| `starttls` | Dials plain TCP, then issues `STARTTLS` after `EHLO` and upgrades the connection before `AUTH`/`MAIL FROM`. Matches a server started with `--smtp-require-starttls`. |
| `implicit` | Dials straight into TLS, no `STARTTLS` negotiation. Matches a server started with `--smtp-require-tls`. |

`--smtp-tls-insecure-skip-verify` accepts a self-signed/dev certificate
without verifying it against a CA. This is expected routine use when the
target is your own `maelsink serve` instance running with
`--smtp-tls-cert`/`--smtp-tls-key` — **never use it against a real
CA-issued certificate in a shared or production-adjacent environment**, since
it disables the check that the server is who it claims to be.

Both flags are available on:

- `maelsink send --smtp-tls <mode> [--smtp-tls-insecure-skip-verify]`
- `maelsink shell --smtp-tls <mode> [--smtp-tls-insecure-skip-verify]` (sets
  the session-wide default for every mail-sending builtin)
- Per-invocation, on each of `send`/`intmsg`/`randmsg`/`weirdmsg`/`blast`/
  `deluge` inside the shell, overriding the session default for just that
  call — exactly like those builtins' existing `--smtp-host`/`--auth-user`
  overrides.

## Worked examples

### `implicit` against `--smtp-require-tls`

```bash
maelsink serve --smtp-tls-cert cert.pem --smtp-tls-key key.pem --smtp-require-tls
```

```bash
maelsink send --smtp-tls implicit --smtp-tls-insecure-skip-verify \
  --from a@b.com --to c@d.com --subject hi --text "hello over implicit TLS"
```

### `starttls` against `--smtp-require-starttls`

```bash
maelsink serve --smtp-tls-cert cert.pem --smtp-tls-key key.pem --smtp-require-starttls
```

```bash
maelsink send --smtp-tls starttls --smtp-tls-insecure-skip-verify \
  --from a@b.com --to c@d.com --subject hi --text "hello over STARTTLS"
```

### AUTH over TLS

The server's RFC 4954 guard rejects `AUTH` sent over a plaintext connection
unless the server was started with `--smtp-auth-allow-insecure`. So once
`--smtp-auth-enabled` is on, `--auth-user`/`--auth-pass` on the CLI need
`--smtp-tls starttls` or `--smtp-tls implicit` to succeed:

```bash
maelsink serve --smtp-tls-cert cert.pem --smtp-tls-key key.pem \
  --smtp-require-tls --smtp-auth-enabled \
  --smtp-auth-username alice --smtp-auth-password s3cret
```

```bash
maelsink send --smtp-tls implicit --smtp-tls-insecure-skip-verify \
  --auth-user alice --auth-pass s3cret \
  --from a@b.com --to c@d.com --subject hi --text "authenticated send"
```

### From `maelsink shell`

```bash
maelsink shell --smtp-tls implicit --smtp-tls-insecure-skip-verify \
  --auth-user alice --auth-pass s3cret
```

sets the session default for every builtin; a single invocation can still
override it, e.g. `send --smtp-tls starttls ...` for just that one send.
