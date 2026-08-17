---
title: "config validate"
description: "Validate a maelsink.yaml (and the layered config it produces) without starting any server."
---

Validate a maelsink.yaml (and the layered config it produces) without starting any server.

This page is generated directly from `maelsink config validate --help` — the flag list below is always in sync with the binary; it is never hand-maintained.

```
Loads and validates a config file (--config, defaults to ./maelsink.yaml), reporting any errors, without starting the server.

Usage:
  maelsink config validate [flags]

Flags:
  -h, --help   help for validate

Global Flags:
  -c, --config string       path to maelsink.yaml
  -j, --log-file string     log file path (empty = stdout only)
  -F, --log-format string   log format (text|json)
  -l, --log-level string    log level (debug|info|warn|error)
```
