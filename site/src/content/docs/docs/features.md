---
title: Features
description: A map of everything maelsink can do, with links into the guide for each capability.
---

maelsink combines an SMTP sink, a web UI, and a REST API in one binary. This page lists
every capability, with each bullet linking to the guide that covers it in depth.

## Core sink

- **SMTP sink** — accepts any mail sent to it over SMTP and stores it in SQLite; never
  relays anywhere. See [Configuration → SMTP server](/maelsink/docs/configuration/smtp-server/).
- **TLS and auth on the SMTP port** — STARTTLS/implicit TLS and SMTP AUTH, including
  multi-user htpasswd-style credential files. See
  [Configuration → TLS certificates](/maelsink/docs/configuration/tls-certificates/) and
  [Configuration → Password files](/maelsink/docs/configuration/password-files/).

## Web UI and REST API

- **Web UI** — a live inbox with real-time updates over WebSocket, message detail view,
  settings, tags, and session history. See
  [Configuration → Web UI and API server](/maelsink/docs/configuration/web-ui-and-api-server/).
- **REST API** — a stable `/api/v1/*` surface for listing, filtering, retrieving, tagging,
  and deleting messages, plus stats and health checks. See
  [REST API Reference](/maelsink/docs/rest-api-reference/).
- **Filters and search** — filter messages by sender, recipient, subject, date range, and
  full-text query. See [Usage → Filters and Search](/maelsink/docs/usage/filters-and-search/) and
  [Usage → Advanced Search Patterns](/maelsink/docs/usage/advanced-search-patterns/) for the Web UI,
  or [Usage → Using CLI](/maelsink/docs/usage/using-cli/) for the CLI and REST API.
- **Tagging** — organize captured messages with tags. See
  [Usage → Tagging Messages](/maelsink/docs/usage/tagging-messages/).
- **Deleting messages** — remove one message or clear the whole inbox. See
  [Usage → Deleting Messages](/maelsink/docs/usage/deleting-messages/).
- **Export** — download any message as a `.eml` file, individually or in bulk. See
  [Usage → Export](/maelsink/docs/usage/export/).

## Sending mail

- **Programmatically** — worked examples in Node, Python, Go, Ruby, PHP, Java, .NET, and
  popular frameworks. See
  [Usage → Sending Mail → Programmatically](/maelsink/docs/usage/sending-mail/programmatically/).
- **Via CLI** — the built-in `maelsink send` sendmail-equivalent client. See
  [Usage → Sending Mail → Via CLI Commands](/maelsink/docs/usage/sending-mail/via-cli-commands/).
- **Via Shell** — generate and send test messages from an interactive shell. See
  [Usage → Sending Mail → Via Shell](/maelsink/docs/usage/sending-mail/via-shell/).
- **Via Composer UI** — a point-and-click browser front end for the same capabilities.
  See [Usage → Sending Mail → Via Composer UI](/maelsink/docs/usage/sending-mail/via-composer-ui/).

## Interactive shell

- **`maelsink shell`** — a full interactive shell for exploring and scripting against a
  maelsink instance, with builtins, templating, and history. See
  [Usage → Using Shell](/maelsink/docs/usage/using-shell/),
  [Shell Builtin Reference](/maelsink/docs/shell-builtin-reference/), and
  [Shell Functions Reference](/maelsink/docs/shell-functions-reference/).

## Composer

- **`maelsink compose`** — a standalone browser-based playground giving every shell
  capability a visual front end, useful for headless deployments with no local terminal.
  See [Usage → Using Composer](/maelsink/docs/usage/using-composer/).

## Operations

- **Configuration** — a layered config system (defaults, YAML file, environment
  variables, CLI flags). See [Configuration → Runtime options](/maelsink/docs/configuration/runtime-options/).
- **Integration testing** — run maelsink as an ephemeral SMTP sink in CI. See
  [Integration Testing](/maelsink/docs/integration-testing/).
- **Reverse proxying** — run maelsink behind a proxy like HAProxy. See
  [Guides → Running maelsink behind HAProxy](/maelsink/docs/guides/haproxy/).
