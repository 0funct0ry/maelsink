---
title: "auth removeuser"
description: "Remove a Web UI Basic Auth user from an htpasswd-style credential file."
---

Remove a Web UI Basic Auth user from an htpasswd-style credential file.

This page is generated directly from `maelsink auth removeuser --help`. The flag list below is always in sync with the binary and is never hand-maintained.

```
Removes a user's entry from the --web-auth-file htpasswd-style credential
file. Works standalone against just the file — no running maelsink server
is required.

  maelsink auth removeuser bob --web-auth-file /data/webauth.htpasswd

  docker exec my-maelsink maelsink auth removeuser bob --web-auth-file /data/webauth.htpasswd

Errors if the username or the file itself doesn't exist.

Usage:
  maelsink auth removeuser <username> [flags]

Flags:
  -h, --help                   help for removeuser
  -L, --web-auth-file string   path to the htpasswd-style basic-auth file (default "maelsink.htpasswd")

Global Flags:
  -c, --config string       path to maelsink.yaml
  -j, --log-file string     log file path (empty = stdout only)
  -F, --log-format string   log format (text|json)
  -l, --log-level string    log level (debug|info|warn|error)
```
