---
title: Password Files
description: The htpasswd-style web.auth.file/smtp.auth.file multi-user credential stores, and CI shortcuts.
---

Both the Web UI and the SMTP server support multi-user authentication backed by a
plain-text, htpasswd-style credential file — one username per line, bcrypt-hashed
password.

## File format

Each line is `username:bcrypt-hash`, for example:

```
alice:$2a$10$yxAVtZUprPAt0XuzZ4RQdewQjnlaav4jJnEiAc72nB6c1HnkHKsQu
```

Don't hand-edit this file — use `maelsink auth adduser`/`auth removeuser` (below) so the
hash is generated correctly.

## Managing users

```sh
maelsink auth adduser
```

Fully interactive — prompts for both username and password (password input is masked,
so it never touches shell history or `ps` output), writing to `./maelsink.htpasswd` by
default.

Username as a positional arg, password still prompted:

```sh
maelsink auth adduser bob --web-auth-file /data/webauth.htpasswd
```

Non-interactive, for Docker/CI (`--password-stdin` is the recommended path — `--password`
also exists but is visible in shell history/`ps`):

```sh
echo "$PASSWORD" | maelsink auth adduser bob --web-auth-file /data/webauth.htpasswd --password-stdin
```

```sh
docker exec my-maelsink maelsink auth adduser bob --web-auth-file /data/webauth.htpasswd --password-stdin <<< "$PASSWORD"
```

Removing a user:

```sh
maelsink auth removeuser bob --web-auth-file /data/webauth.htpasswd
```

`removeuser` errors if the username or the file itself doesn't exist. Both `adduser` and
`removeuser` work standalone against just the file — no running maelsink server is
required, so you can prepare a credential file before `serve` ever starts.

## Wiring the file into `serve`

```sh
maelsink serve --web-auth-file /data/webauth.htpasswd
```

```sh
maelsink serve --smtp-auth-file /data/smtpauth.htpasswd --smtp-auth-enabled
```

`web.auth.file` gates the Web UI (and its mounted `/api/v1` and `/ui-api` routes) behind
HTTP Basic Auth. `smtp.auth.file` is checked during SMTP AUTH when `smtp.auth.enabled`
is `true`. See [Runtime options](/maelsink/docs/configuration/runtime-options/) for both keys'
env vars and flags.

## CI shortcuts

For ephemeral CI runs where you don't want to manage a credential file at all:

- **`smtp.auth.accept_any`** (`-Y, --smtp-auth-accept-any`) — accept any username/password
  combination on SMTP AUTH without checking a file. Useful when your app under test
  requires SMTP AUTH to be configured, but you don't care what the credentials actually
  are.
- **`MAELSINK_SMTP_AUTH`** — an env-only override (no YAML key, no CLI flag) that takes a
  space-separated list of `user:pass` pairs, parsed directly by `serve` and bypassing the
  config/viper layer entirely:

  ```sh
  MAELSINK_SMTP_AUTH="alice:secret1 bob:secret2" maelsink serve --smtp-auth-enabled
  ```

  This is convenient for a single CI job that needs one or two fixed credentials without
  writing a file to disk first.
