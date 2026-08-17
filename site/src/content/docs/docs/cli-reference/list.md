---
title: "list"
description: "List messages via the REST API."
---

List messages via the REST API.

This page is generated directly from `maelsink list --help` — the flag list below is always in sync with the binary; it is never hand-maintained.

```
Thin REST API client: lists stored messages in table or JSON format.

--format also accepts a Go template, docker-CLI-style, executed once per
message: maelsink list --format '{{.ID}}: {{.From}} -> {{.Subject}}'

Usage:
  maelsink list [flags]

Flags:
      --api string       maelsink REST API base URL (default "http://127.0.0.1:9090")
      --api-key string   REST API bearer key (if api.auth.enabled)
      --bcc string       filter by bcc address substring
      --cc string        filter by cc address substring
      --format string    output format: table|json, or a Go template (e.g. '{{.ID}}: {{.Subject}}') (default "table")
      --from string      filter by from address substring
  -h, --help             help for list
      --limit int        max messages to return (0 = server default)
      --offset int       pagination offset
      --q string         full-text search query
      --since string     only messages received at/after this RFC3339 timestamp
      --sort string      sort order: received_at_desc|received_at_asc
      --subject string   filter by subject substring
      --to string        filter by to address substring
      --until string     only messages received at/before this RFC3339 timestamp

Global Flags:
  -c, --config string       path to maelsink.yaml
  -j, --log-file string     log file path (empty = stdout only)
  -F, --log-format string   log format (text|json)
  -l, --log-level string    log level (debug|info|warn|error)
```
