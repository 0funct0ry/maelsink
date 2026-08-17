---
title: "compose"
description: "Start the maelsink compose browser-based playground."
---

Start the maelsink compose browser-based playground.

This page is generated directly from `maelsink compose --help` — the flag list below is always in sync with the binary; it is never hand-maintained.

```
Starts maelsink compose: a small local web server serving a standalone
single-page app that gives every capability of "maelsink shell" a visual,
point-and-click front end. It is a pure client of a target maelsink
instance's REST API and SMTP port — it starts no database and never talks
to storage directly. Useful for trying maelsink for the first time, or for
any target (e.g. a headless Docker deployment) with no local terminal
attached.

Usage:
  maelsink compose [flags]

Flags:
  -A, --api-addr string            target maelsink REST API base URL (default "http://localhost:9090")
  -C, --api-ca-cert string         path to a CA cert to trust for the target API's TLS
  -k, --api-insecure-skip-verify   skip TLS verification when calling the target API (local/CI use only)
  -P, --api-pass string            basic-auth password for the target API
  -u, --api-user string            basic-auth username for the target API (if fronted by auth)
  -h, --help                       help for compose
  -L, --listen string              compose server listen address (default ":8090")
  -o, --open                       automatically open the compose UI in a browser on startup
  -S, --smtp-addr string           target maelsink SMTP address (host:port) (default "127.0.0.1:1025")
  -W, --smtp-pass string           target SMTP AUTH password
  -U, --smtp-user string           target SMTP AUTH username

Global Flags:
  -c, --config string       path to maelsink.yaml
  -j, --log-file string     log file path (empty = stdout only)
  -F, --log-format string   log format (text|json)
  -l, --log-level string    log level (debug|info|warn|error)
```
