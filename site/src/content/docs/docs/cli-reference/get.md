---
title: "get"
description: "Show full message detail via the REST API."
---

Show full message detail via the REST API.

This page is generated directly from `maelsink get --help` — the flag list below is always in sync with the binary; it is never hand-maintained.

```
Thin REST API client: fetches and prints one message by id.

Usage:
  maelsink get <id> [flags]

Flags:
      --api string       maelsink REST API base URL (default "http://127.0.0.1:9090")
      --api-key string   REST API bearer key (if api.auth.enabled)
      --format string    output format: table|json, or a Go template (e.g. '{{.ID}}: {{.Subject}}') (default "table")
  -h, --help             help for get

Global Flags:
  -c, --config string       path to maelsink.yaml
  -j, --log-file string     log file path (empty = stdout only)
  -F, --log-format string   log format (text|json)
  -l, --log-level string    log level (debug|info|warn|error)
```
