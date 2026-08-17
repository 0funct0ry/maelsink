---
title: "export"
description: "Download a message as a .eml file via the REST API."
---

Download a message as a .eml file via the REST API.

This page is generated directly from `maelsink export --help`. The flag list below is always in sync with the binary and is never hand-maintained.

```
Thin REST API client: writes a message's raw source to -o <path>, or ./<id>.eml if omitted.

Usage:
  maelsink export <id> [flags]

Flags:
      --api string       maelsink REST API base URL (default "http://127.0.0.1:9090")
      --api-key string   REST API bearer key (if api.auth.enabled)
  -h, --help             help for export
  -o, --output string    output file path (default ./<id>.eml)

Global Flags:
  -c, --config string       path to maelsink.yaml
  -j, --log-file string     log file path (empty = stdout only)
  -F, --log-format string   log format (text|json)
  -l, --log-level string    log level (debug|info|warn|error)
```
