---
title: Filters and Search
description: Filter and search captured mail with maelsink list and the equivalent GET /api/v1/messages query parameters.
---

Every message maelsink accepts over SMTP is immediately queryable through `maelsink list` and the underlying `GET /api/v1/messages` REST endpoint. Both accept the same filters — `list` is a thin client over that endpoint.

## `maelsink list` flags

```
maelsink list [--q string] [--from string] [--to string] [--subject string]
              [--cc string] [--bcc string] [--limit int] [--offset int]
              [--since string] [--until string] [--sort string]
```

| Flag | Meaning |
|---|---|
| `--q` | Full-text search query |
| `--from` | Substring match against the `From` address |
| `--to` | Substring match against the `To` address |
| `--subject` | Substring match against the subject |
| `--cc` | Substring match against `Cc` |
| `--bcc` | Substring match against `Bcc` |
| `--limit` | Max messages to return (`0` = server default) |
| `--offset` | Pagination offset |
| `--since` | Only messages received at/after this RFC3339 timestamp |
| `--until` | Only messages received at/before this RFC3339 timestamp |
| `--sort` | `received_at_desc` (default) or `received_at_asc` |

`list` also takes the standard API client flags (`--api`, `--api-key`) and `--format table|json|<Go template>`.

## Live examples

Against a local `maelsink serve` with a couple of test messages sent:

```
$ maelsink send --to alice@example.com --from bob@example.com \
    --subject "Password reset" --text "Click here to reset"
message sent

$ maelsink send --to carol@example.com --from bob@example.com \
    --subject "Weekly digest" --text "Hello"
message sent
```

Filter by sender substring:

```
$ maelsink list --from=bob
ID                        FROM             TO                 SUBJECT         SIZE  ATTACHMENTS  RECEIVED
5c02cbc79b40326a4aca5376  bob@example.com  carol@example.com  Weekly digest   170   0            2026-08-17T09:24:13Z
bed302ae42d8d1b913c1c999  bob@example.com  alice@example.com  Password reset  185   0            2026-08-17T09:24:13Z
```

Full-text query (matches subject/body content):

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
This page's examples ran against a real local `maelsink serve` instance (SQLite, ports `1025`/`9090`) started for this build — the table output above is exactly what the CLI printed, not a hand-written mockup.
:::

## REST equivalent

`maelsink list` is a thin client of `GET /api/v1/messages`, which accepts the same filters as query parameters, plus a few only exposed at the REST layer (not yet surfaced as `list` flags): `tag`, `tag_mode`, `read`, `has_attachments`, and `parse_warning`. See [Advanced Search Patterns](/maelsink/docs/usage/advanced-search-patterns/) and [Tagging Messages](/maelsink/docs/usage/tagging-messages/) for those.

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
