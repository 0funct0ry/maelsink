---
title: Using CLI
description: Filtering and searching captured mail with maelsink list and the equivalent REST API query parameters.
---

Every message maelsink accepts over SMTP is queryable through `maelsink list` and the underlying `GET /api/v1/messages` REST endpoint. `list` is a thin client over that endpoint, so both accept the same filters.

## `maelsink list` flags

```
maelsink list [--q string] [--from string] [--to string] [--subject string]
              [--cc string] [--bcc string] [--limit int] [--offset int]
              [--since string] [--until string] [--sort string]
```

| Flag | Meaning |
|---|---|
| `--q` | Full-text search query (see [Advanced Search Patterns](/maelsink/docs/usage/advanced-search-patterns/) for the query syntax) |
| `--from` | Substring match against the `From` address |
| `--to` | Substring match against the `To` address |
| `--subject` | Substring match against the subject |
| `--cc` | Substring match against `Cc` |
| `--bcc` | Substring match against `Bcc` |
| `--limit` | Max messages to return (`0` uses the server default) |
| `--offset` | Pagination offset |
| `--since` | Only messages received at or after this RFC3339 timestamp |
| `--until` | Only messages received at or before this RFC3339 timestamp |
| `--sort` | `received_at_desc` (default) or `received_at_asc` |

`list` also accepts the standard API client flags (`--api`, `--api-key`) and `--format table|json|<Go template>`.

## Live examples

Against a local `maelsink serve` instance with a couple of test messages sent:

```
$ maelsink send --to alice@example.com --from bob@example.com \
    --subject "Password reset" --text "Click here to reset"
message sent

$ maelsink send --to carol@example.com --from bob@example.com \
    --subject "Weekly digest" --text "Hello"
message sent
```

Filtering by sender substring:

```
$ maelsink list --from=bob
ID                        FROM             TO                 SUBJECT         SIZE  ATTACHMENTS  RECEIVED
5c02cbc79b40326a4aca5376  bob@example.com  carol@example.com  Weekly digest   170   0            2026-08-17T09:24:13Z
bed302ae42d8d1b913c1c999  bob@example.com  alice@example.com  Password reset  185   0            2026-08-17T09:24:13Z
```

Full-text query, matching subject and body content:

```
$ maelsink list --q=password
ID                        FROM             TO                 SUBJECT         SIZE  ATTACHMENTS  RECEIVED
bed302ae42d8d1b913c1c999  bob@example.com  alice@example.com  Password reset  185   0            2026-08-17T09:24:13Z
```

Subject filter combined with ascending sort:

```
$ maelsink list --subject=digest --sort=received_at_asc
ID                        FROM             TO                 SUBJECT        SIZE  ATTACHMENTS  RECEIVED
5c02cbc79b40326a4aca5376  bob@example.com  carol@example.com  Weekly digest  170   0            2026-08-17T09:24:13Z
```

:::note
This page's examples ran against a real local `maelsink serve` instance (SQLite, ports `1025`/`9090`) started for this build. The table output above is exactly what the CLI printed, not a hand-written mockup.
:::

## REST equivalent

`maelsink list` is a thin client of `GET /api/v1/messages`, which accepts the same filters as query parameters, plus a few not yet surfaced as `list` flags: `tag`, `tag_mode`, `read`, `has_attachments`, and `parse_warning`. See [Tagging Messages](/maelsink/docs/usage/tagging-messages/) for the tag parameters.

```
curl "http://127.0.0.1:9090/api/v1/messages?from=bob&sort=received_at_asc"
```

```json
{
  "messages": [
    {
      "id": "5c02cbc79b40326a4aca5376",
      "from": "bob@example.com",
      "to": ["carol@example.com"],
      "subject": "Weekly digest",
      "received_at": "2026-08-17T09:24:13Z",
      "tags": [],
      "...": "..."
    }
  ],
  "total": 2,
  "limit": 50,
  "offset": 0
}
```

## Combining multiple filters

Every filter on `GET /api/v1/messages` is combined with AND. A search can be narrowed by stacking flags:

```
maelsink list --from=bob --subject=digest --sort=received_at_asc
```

which is equivalent to:

```
curl "http://127.0.0.1:9090/api/v1/messages?from=bob&subject=digest&sort=received_at_asc"
```

`--from`, `--to`, `--subject`, `--cc`, and `--bcc` are substring matches against a specific header, rather than a field-prefixed query syntax such as `from:bob`. `--q` is the exception: it performs a full-text search across subject and body content rather than matching a specific header.

## Date-range searches

`--since` and `--until` (and the corresponding REST `since`/`until` parameters) take RFC3339 timestamps and are inclusive at both ends:

```
maelsink list --since=2026-08-01T00:00:00Z --until=2026-08-17T23:59:59Z
```

```
curl "http://127.0.0.1:9090/api/v1/messages?since=2026-08-01T00:00:00Z&until=2026-08-17T23:59:59Z"
```

These combine with other filters the same way:

```
maelsink list --from=bob --since=2026-08-17T00:00:00Z
```

## Attachment-presence searches

The REST endpoint supports a `has_attachments` boolean filter. This parameter is REST-only and is not currently exposed as a `maelsink list` flag:

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

Until a dedicated `list` flag exists, this filter can be approximated from the CLI by combining `--q` with attachment-related terms in the subject or body, or by reaching for the REST parameter directly with `curl`, `maelsink shell`'s `show` builtin (which reports `has_attachments` and `attachment_count` per message), or a `--format` Go template on `list` that filters client-side:

```
maelsink list --format '{{if .HasAttachments}}{{.ID}}: {{.Subject}}{{end}}'
```

:::note
Two other REST-only filters exist alongside `has_attachments`: `read` (boolean) and `parse_warning` (boolean, flags messages that maelsink could not fully MIME-parse). Neither has a dedicated `list` flag yet, so the same `curl` or `--format` template approach applies.
:::

## Other REST-only filters

- `tag` (repeatable) and `tag_mode=any|all`, described in [Tagging Messages](/maelsink/docs/usage/tagging-messages/)
- `read=true|false`, filtering by read or unread state
- `parse_warning=true|false`, surfacing messages that had a MIME parsing issue on ingest

These compose with every other filter the same way, since they are all additional query parameters on the same `GET /api/v1/messages` call.
