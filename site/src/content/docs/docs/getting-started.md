---
title: Getting Started
description: A guide to installing maelsink, sending a test email, and viewing it in the inbox.
---

This page describes the fastest path to a running maelsink instance and a captured test
email. Every command below has been run against a real build.

![Running `maelsink serve`, sending a test message, and listing it](/maelsink/recordings/quickstart.gif)

## 1. Get the binary

Download a release for your platform from
[GitHub Releases](https://github.com/0funct0ry/maelsink/releases/latest), or install via
Homebrew, Scoop, or Docker — see [Installation](/maelsink/docs/installation/) for all four paths.
For this walkthrough we'll assume you have a `maelsink` binary on your `PATH` (or you can
build one yourself — see [Building from Source](/maelsink/docs/installation/building-from-source/)).

## 2. Run it

```sh
maelsink serve
```

By default this starts three listeners:

```
SMTP     -> 127.0.0.1:1025
Web UI   -> http://127.0.0.1:8080/
REST API -> http://127.0.0.1:9090/api/v1
```

Leave it running. Open [http://127.0.0.1:8080/](http://127.0.0.1:8080/) in a browser to
see the (empty, for now) inbox.

## 3. Send a test email

maelsink ships a built-in sendmail-equivalent client, so you don't need a separate mail
client to try it. In a second terminal:

```sh
maelsink send \
  --from me@example.com \
  --to test@example.com \
  --subject "Hello from maelsink" \
  --text "This is a test message."
```

You should see:

```
message sent
```

## 4. See it appear

Refresh the Web UI — the message shows up instantly (it's pushed over WebSocket the
moment it's stored). You can also confirm it landed via the REST API:

```sh
curl -s http://127.0.0.1:9090/api/v1/messages
```

```json
{"messages":[{"id":"6c5370232fe28b4d5735589e","from":"me@example.com","to":["test@example.com"],"cc":[],"bcc":[],"subject":"Hello from maelsink","size_bytes":185,"has_attachments":false,"attachment_count":0,"received_at":"2026-08-17T09:20:45Z","parse_warning":false,"read":false,"tags":[],"preview":"This is a test message."}],"total":1,"limit":50,"offset":0}
```

That's it — any app configured to send SMTP mail to `127.0.0.1:1025` will show up the
same way. No relaying, no real mailbox, nothing leaves your machine.

## Where to next

- [Features](/maelsink/docs/features/) for the full capability map.
- [Configuration → Runtime options](/maelsink/docs/configuration/runtime-options/) to change ports,
  storage location, or anything else.
- [Integration testing](/maelsink/docs/integration-testing/) to run maelsink as an ephemeral SMTP
  sink in CI.
