---
title: "delete"
description: "Delete one message via the REST API."
---

Delete one message via the REST API.

This page is generated directly from `maelsink delete --help` — the flag list below is always in sync with the binary; it is never hand-maintained.

```
Thin REST API client: deletes a single message by id.

Usage:
  maelsink delete <id> [flags]

Flags:
      --api string       maelsink REST API base URL (default "http://127.0.0.1:9090")
      --api-key string   REST API bearer key (if api.auth.enabled)
  -h, --help             help for delete

Global Flags:
  -c, --config string       path to maelsink.yaml
  -j, --log-file string     log file path (empty = stdout only)
  -F, --log-format string   log format (text|json)
  -l, --log-level string    log level (debug|info|warn|error)
```
