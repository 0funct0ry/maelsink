---
title: "clear"
description: "Delete all messages via the REST API."
---

Delete all messages via the REST API.

This page is generated directly from `maelsink clear --help` — the flag list below is always in sync with the binary; it is never hand-maintained.

```
Thin REST API client: deletes every stored message. Prompts for confirmation unless --yes is given.

Usage:
  maelsink clear [flags]

Flags:
      --api string       maelsink REST API base URL (default "http://127.0.0.1:9090")
      --api-key string   REST API bearer key (if api.auth.enabled)
  -h, --help             help for clear
  -y, --yes              skip the confirmation prompt

Global Flags:
  -c, --config string       path to maelsink.yaml
  -j, --log-file string     log file path (empty = stdout only)
  -F, --log-format string   log format (text|json)
  -l, --log-level string    log level (debug|info|warn|error)
```
