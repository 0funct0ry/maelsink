---
title: "auth adduser"
description: "Add (or update) a Web UI Basic Auth user in an htpasswd-style credential file."
---

Add (or update) a Web UI Basic Auth user in an htpasswd-style credential file.

This page is generated directly from `maelsink auth adduser --help` — the flag list below is always in sync with the binary; it is never hand-maintained.

```
Adds a new user to the --web-auth-file htpasswd-style credential file, or
updates an existing user's password in place. Works standalone against just
the file — no running maelsink server is required.

Fully interactive (prompts for both username and password, password masked,
no shell history/ps exposure), writing to ./maelsink.htpasswd by default:

  maelsink auth adduser

Interactive password only (username given as a positional arg):

  maelsink auth adduser bob --web-auth-file /data/webauth.htpasswd

Non-interactive, for Docker/CI (preferred: --password-stdin):

  echo "$PASSWORD" | maelsink auth adduser bob --web-auth-file /data/webauth.htpasswd --password-stdin

  docker exec my-maelsink maelsink auth adduser bob --web-auth-file /data/webauth.htpasswd --password-stdin <<< "$PASSWORD"

  docker run --rm -v $(pwd)/webauth.htpasswd:/data/webauth.htpasswd maelsink \
    auth adduser bob --web-auth-file /data/webauth.htpasswd --password-stdin <<< "$PASSWORD"

--password takes the value directly as a flag; it is provided for
convenience but is visible in shell history and "ps" output on most
systems, so --password-stdin is the recommended non-interactive path.

Usage:
  maelsink auth adduser [username] [flags]

Flags:
  -h, --help                   help for adduser
      --password string        password value (visible in shell history/ps — prefer --password-stdin)
      --password-stdin         read the password from stdin (recommended for scripted/Docker use)
  -L, --web-auth-file string   path to the htpasswd-style basic-auth file (default "maelsink.htpasswd")

Global Flags:
  -c, --config string       path to maelsink.yaml
  -j, --log-file string     log file path (empty = stdout only)
  -F, --log-format string   log format (text|json)
  -l, --log-level string    log level (debug|info|warn|error)
```
