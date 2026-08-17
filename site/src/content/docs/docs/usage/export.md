---
title: Export
description: Every way to export captured mail as .eml files — the CLI, the shell's export builtin, and the Composer/Web UI, single message or bulk.
---

Every stored message can be exported as a standard `.eml` file (raw RFC 5322 source) — the same bytes maelsink received over SMTP, unmodified.

## `maelsink export` (CLI)

```
maelsink export <id> [-o/--output string]
```

Writes one message to `-o <path>`, or `./<id>.eml` if omitted:

```
$ maelsink export 5c02cbc79b40326a4aca5376
./5c02cbc79b40326a4aca5376.eml

$ cat 5c02cbc79b40326a4aca5376.eml
From: bob@example.com
To: carol@example.com
Subject: Weekly digest
Date: Mon, 17 Aug 2026 09:24:13 +0000
MIME-Version: 1.0
Content-Type: text/plain; charset=utf-8

Hello
```

Live-verified — the file above is exactly what a real `maelsink export` call wrote against a local test server. This maps to `GET /api/v1/messages/{id}/export`, which sets `Content-Disposition` so the CLI/browser gets a sensible filename.

## The shell's `export` builtin

The shell's `export` builtin is considerably richer, supporting bulk export by filter:

```
export [<id>...] [--all] [--zip] [-o/--out string]
       [-q/--query string] [--from string] [--to string] [--subject string]
       [--since string] [--until string] [--sort string]
```

- **One id**: `export <id>` writes a single `.eml`, same as the CLI command.
- **Several ids, no `--zip`**: writes one `.eml` per id into the `--out` directory (default `.`).
- **`--zip`**: bundles the exported message(s) into one `.zip` at `--out` (default `export.zip` when combined with `--all`).
- **`--all`**: ignores any explicit ids and instead exports every message matching the filter flags (`--query`/`--from`/`--to`/`--subject`/`--since`/`--until`/`--sort` — the same shape as `list`'s filters), server-side, as one `.zip` via `GET /api/v1/messages/export`.

Live example — sending two messages with `randmsg` and bulk-exporting everything:

```
$ maelsink shell -e 'randmsg' -e 'randmsg' -e 'export --all --out bulk.zip'
1/1 sent
1/1 sent
wrote bulk.zip

$ unzip -l bulk.zip
Archive:  bulk.zip
  Length      Date    Time    Name
---------  ---------- -----   ----
     1358  00-00-1980 00:00   4a448083cfa103f0d762bd3c.eml
     1267  00-00-1980 00:00   381b4f540025cda8fbac3576.eml
---------                     -------
     2625                     2 files
```

Live-verified end to end — the zip really contained one `.eml` per message, each a valid RFC 5322 document.

## Exporting from the Composer / Web UI

- **Web UI message detail screen**: an Export action on a single message triggers the same `GET /api/v1/messages/{id}/export` download the CLI uses.
- **Composer's API Explorer screen** (`web-compose/src/screens/ApiExplorerScreen.tsx`): the `export` endpoint card exposes `GET /api/v1/messages/export` directly, letting you build a filtered bulk export request (same query params as `list`) and see/download the raw response without leaving the browser.

## Output format

Every export path produces the identical format: raw RFC 5322 source (`.eml`), individually for single-message export, or bundled into a `.zip` of `.eml` files for bulk export. There is no other export format (no `.mbox`, no JSON export of message contents) — `.eml` is it.
