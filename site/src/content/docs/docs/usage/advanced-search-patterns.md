---
title: Advanced Search Patterns
description: Field-scoped queries, combining filters, date-range searches, and attachment-presence filtering against maelsink's message store.
---

Beyond a single `--from` or `--q`, the REST API (and by extension `maelsink list`, `maelsink shell`'s `list` builtin, and the Web UI/Composer) support combining filters and a couple of query parameters that aren't yet surfaced as `list`/`show` flags.

## Combining multiple filters

Every filter on `GET /api/v1/messages` is AND-combined. Narrow a search by stacking flags:

```
maelsink list --from=bob --subject=digest --sort=received_at_asc
```

is equivalent to:

```
curl "http://127.0.0.1:9090/api/v1/messages?from=bob&subject=digest&sort=received_at_asc"
```

Because `--from`/`--to`/`--subject`/`--cc`/`--bcc` are substring matches (not exact-match or field-prefixed query syntax like `from:bob`), each one scopes the search to that specific header rather than searching everywhere — that's the "field-scoped" part. `--q` is the odd one out: it's a full-text search across subject/body content, not a specific header.

## Date-range searches

`--since` / `--until` (and the REST `since`/`until` params) take RFC3339 timestamps and are inclusive at both ends:

```
maelsink list --since=2026-08-01T00:00:00Z --until=2026-08-17T23:59:59Z
```

```
curl "http://127.0.0.1:9090/api/v1/messages?since=2026-08-01T00:00:00Z&until=2026-08-17T23:59:59Z"
```

Combine with other filters the same way:

```
maelsink list --from=bob --since=2026-08-17T00:00:00Z
```

## Attachment-presence searches

The REST endpoint supports a real `has_attachments` boolean filter — this is a REST-only parameter (checked directly against `internal/api/handlers.go`, which parses it via `parseOptionalBoolQuery`), not currently exposed as a `maelsink list` flag:

```
curl "http://127.0.0.1:9090/api/v1/messages?has_attachments=true"
```

```json
{
  "messages": [
    {
      "id": "627751956cc6fe4055a6bccb",
      "from": "app@example.com",
      "to": ["dev@example.com"],
      "subject": "With attachment",
      "has_attachments": true,
      "attachment_count": 1,
      "...": "..."
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

This was verified live: sending a message with `maelsink send --attach ./note.txt ...` and then querying `?has_attachments=true` returned exactly that message.

From `maelsink list`, there's no dedicated flag for this yet, so the closest you can do from the CLI today is combine `--q` with attachment-related terms in the subject/body (e.g. `--q=invoice`), or reach for the REST parameter directly with `curl`, `maelsink shell`'s `show` builtin (which reports `has_attachments`/`attachment_count` per message), or a `--format` Go template on `list` that filters client-side:

```
maelsink list --format '{{if .HasAttachments}}{{.ID}}: {{.Subject}}{{end}}'
```

:::note
Two other REST-only filters exist alongside `has_attachments`: `read` (boolean) and `parse_warning` (boolean, flags messages maelsink couldn't fully MIME-parse). Same story — reach for `curl` or a `--format` template until a dedicated `list` flag exists.
:::

## Other REST-only filters

- `tag` (repeatable) and `tag_mode=any|all` — see [Tagging Messages](/maelsink/docs/usage/tagging-messages/).
- `read=true|false` — filter by read/unread state.
- `parse_warning=true|false` — surface messages that had a MIME parsing issue on ingest.

These compose with every other filter the same way, since they're all just additional query parameters on the same `GET /api/v1/messages` call.
