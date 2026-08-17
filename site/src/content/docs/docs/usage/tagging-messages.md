---
title: Tagging Messages
description: The X-Tag header convention for grouping test emails, tag-based filtering, and the REST tag management endpoints.
---

maelsink derives tags from a plain SMTP header, so any app under test — or any of maelsink's own sending tools — can tag messages without a separate API call.

## The `X-Tag` header convention

Any message ingested over SMTP with one or more `X-Tag` header lines gets those values recorded as its `tags` (confirmed in `internal/smtp/mime.go`'s `extractTags`, which "collects every `X-Tag` header value, preserving order and duplicates"). Multiple `X-Tag` lines on one message all become separate tags:

```
X-Tag: smoke
X-Tag: release
```

- The match is on the header name only (case-insensitively) — the value is used verbatim as the tag name.
- Duplicate `X-Tag` values are preserved as duplicates in the header, but each distinct value becomes one tag.
- `internal/store.Message.Tags` documents the intent directly: "Apps under test set X-Tag to group related test emails without [needing a separate API call]".

### Setting tags with `maelsink send`

`maelsink send`'s `--file` JSON spec has a first-class `tags` field (`cliclient.MessageSpec.Tags`) that becomes one `X-Tag` header per entry when the message is built:

```json
{
  "from": "bob@example.com",
  "to": ["dana@example.com"],
  "subject": "Tagged smoke test",
  "text": "hi",
  "tags": ["smoke", "release"]
}
```

```
$ maelsink send --file tagged.json
message sent
```

Verified live — after sending the spec above, `GET /api/v1/messages?tag=smoke` returned exactly that message with `"tags": ["smoke", "release"]`, and `GET /api/v1/tags` listed both `smoke` and `release` with `count: 1`.

There's no `--tag` flag on the top-level `maelsink send` (its flag set has no `tags` field — only `--file`'s JSON spec exposes it). If you're not using `--file`, you can still tag a message by adding the header yourself via `--raw`:

```
printf 'From: bob@example.com\r\nTo: dana@example.com\r\nSubject: tagged\r\nX-Tag: smoke\r\n\r\nhi\r\n' | maelsink send --raw
```

## Tag-based filtering and search

`GET /api/v1/messages` (and therefore the Web UI/Composer) supports two tag query parameters:

- `tag` — repeatable; filter to messages carrying at least one of the given tag(s)
- `tag_mode` — `any` (default) or `all`, controlling whether a message must match at least one or every listed `tag`

```
curl "http://127.0.0.1:9090/api/v1/messages?tag=smoke"
curl "http://127.0.0.1:9090/api/v1/messages?tag=smoke&tag=release&tag_mode=all"
```

`maelsink list` does not currently expose `--tag`/`--tag-mode` flags — reach for `curl`, `maelsink shell`'s `list` builtin (whose `--ids` and filter flags are the same shape), or a `--format` Go template against `list`'s JSON output.

## REST tag management endpoints

Tags also have their own dedicated resource, independent of any one message:

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/v1/tags` | List every known tag (name, color, message count, last used) |
| `POST` | `/api/v1/tags` | Create a tag (`{"name": string}`) |
| `PATCH` | `/api/v1/tags/{name}` | Rename/update a tag |
| `DELETE` | `/api/v1/tags/{name}` | Delete a tag (untags its messages, doesn't delete them) |
| `DELETE` | `/api/v1/tags/{name}/messages` | Delete a tag **and** every message still carrying it |

Live example:

```
$ curl -s http://127.0.0.1:9090/api/v1/tags
[
  {"name":"release","color":"lime","count":1,"last_used":"2026-08-17T09:24:28Z"},
  {"name":"smoke","color":"lime","count":1,"last_used":"2026-08-17T09:24:28Z"}
]
```

Per-message tags can also be edited directly without resending the message, via `PATCH /api/v1/messages/{id}/tags` with body `{"tags": [...]}` — useful from the Web UI when you want to re-tag something after the fact.
