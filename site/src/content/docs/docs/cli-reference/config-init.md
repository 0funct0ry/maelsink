---
title: "config init"
description: "Write a maelsink.yaml scaffolded with the built-in defaults."
---

Write a maelsink.yaml scaffolded with the built-in defaults.

This page is generated directly from `maelsink config init --help` — the flag list below is always in sync with the binary; it is never hand-maintained.

```
Writes maelsink's built-in default configuration to ./maelsink.yaml, refusing to overwrite an existing file unless --force is given.

Usage:
  maelsink config init [flags]

Flags:
  -f, --force   overwrite an existing maelsink.yaml
  -h, --help    help for init

Global Flags:
  -c, --config string       path to maelsink.yaml
  -j, --log-file string     log file path (empty = stdout only)
  -F, --log-format string   log format (text|json)
  -l, --log-level string    log level (debug|info|warn|error)
```
