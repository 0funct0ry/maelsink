---
title: SMTP Sessions
description: Inspecting the raw SMTP protocol transcript of every connection maelsink accepts, in the Web UI and through the REST API.
---

Every connection accepted on maelsink's SMTP listener is recorded as a session, independent of whether it produces a stored message. A session captures the full raw protocol transcript, line by line, so a rejected, aborted, or malformed SMTP conversation can be inspected exactly as it happened, not only the message that resulted from it.

## What a session records

A session record consists of:

- `id` — a 24-character hex identifier, using the same scheme as message IDs, including unambiguous prefix resolution
- `client_ip` and `client_helo` — the connecting client's address and the hostname it announced in `HELO`/`EHLO`
- `started_at` and `ended_at` — timestamps bounding the connection; `ended_at` is unset while the session is still open
- `status` — empty while the session is in progress, then one of `completed`, `rejected`, `aborted`, or `timeout` once the connection closes
- `message_id` — set only if the session produced a message that was successfully stored
- a transcript: an ordered list of lines, each tagged `C` (sent by the client) or `S` (sent by the server)

`AUTH` command arguments are redacted in the stored transcript (for example, `AUTH PLAIN [REDACTED]`), so credentials submitted during a session are never persisted or displayed.

## Viewing sessions in the Web UI

The Sessions screen, reachable from the list icon in the top bar, shows every recorded session: relative start time, the client's IP and HELO hostname, a status badge, and a link to the resulting message when one exists. An empty status renders as an "In progress" badge; `aborted` and `timeout` render as a warning; `rejected` renders as an error. A session can be deleted individually from this list, or all sessions cleared at once.

Opening a session shows its detail screen: the same metadata (client IP, HELO, start and end time, status), a link to the resulting message if one was stored, and the protocol transcript itself, rendered as `C:`/`S:`-prefixed lines in a monospace panel.

The two recordings below show the same underlying connection: a raw `telnet` session against the SMTP port, followed by that session's transcript appearing in the Web UI as it happens.

![A telnet session against maelsink's SMTP port, sending a message by hand](/maelsink/recordings/telnet-maelsink-session.gif)

![The same telnet session's transcript appearing live in the session detail screen](/maelsink/recordings/session-progress.gif)

Because `session.line` events are published as each line is captured, the transcript in an open session's detail screen fills in live rather than only appearing once the connection closes.

## Realtime updates

The Web UI receives session activity over the same `/ws` WebSocket connection used for message updates. Five event types cover the session lifecycle:

| Event | Payload | Fires when |
|---|---|---|
| `session.started` | `{"id", "client_ip", "started_at"}` | A connection is accepted |
| `session.line` | `{"session_id", "direction", "line", "position"}` | Each transcript line is captured |
| `session.completed` | `{"id", "status", "message_id"}` | The connection closes |
| `session.deleted` | `{"id"}` | A session is deleted |
| `sessions.cleared` | `{}` | All sessions are cleared |

A client that only needs the resulting messages, not the raw protocol exchange, can ignore `session.*` events entirely and rely on `message.created` instead.

## REST API

Sessions are also available through the REST API, without requiring a WebSocket connection:

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/v1/sessions` | List sessions, with `status`, `client_ip`, `since`, `until`, `limit`, `offset`, and `sort` filters |
| `GET` | `/api/v1/sessions/{id}` | Get a session, including its full transcript; `{id}` may be an unambiguous prefix |
| `DELETE` | `/api/v1/sessions/{id}` | Delete a single session |
| `DELETE` | `/api/v1/sessions` | Delete every session; requires `?confirm=true` |

```
curl "http://127.0.0.1:9090/api/v1/sessions/5c02cbc79b40326a4aca5376"
```

```json
{
  "id": "5c02cbc79b40326a4aca5376",
  "client_ip": "127.0.0.1",
  "client_helo": "client.example.com",
  "started_at": "2026-08-17T09:14:58Z",
  "ended_at": "2026-08-17T09:15:00Z",
  "status": "completed",
  "message_id": "bed302ae42d8d1b913c1c999",
  "transcript": [
    { "direction": "C", "line": "MAIL FROM:<alice@example.com>", "position": 3 }
  ]
}
```

See [REST API Reference](/maelsink/docs/rest-api-reference/) for the complete request and response shapes.

## No CLI equivalent yet

There is currently no `maelsink` subcommand or shell builtin for sessions. Session data is accessible only through the Web UI and the REST API described above.
