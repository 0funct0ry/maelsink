---
title: Web UI and API Server
description: The web.* and api.* config keys in context — ports, base paths, and the ui-api vs api/v1 split.
---

maelsink runs the Web UI and the REST API as two separately-configurable listeners
(`web.*` and `api.*`), though the REST API is also mounted read-through on the Web UI
port. For the full key/default/env/flag table, see
[Runtime options](/maelsink/docs/configuration/runtime-options/) — this page explains what those
keys mean in practice.

## Ports and hosts

By default the Web UI listens on `127.0.0.1:8080` and the REST API on
`127.0.0.1:9090`. Change either independently:

```sh
maelsink serve --web-port 8888 --api-port 9999
```

To disable the Web UI entirely and run headless (SMTP + REST API only — useful for CI),
use `-u/--headless`. This is distinct from `web.enabled`/`-e/--web-enabled`, which
governs the same thing at the config layer.

## `base_path`, for reverse proxies

`web.base_path` and `api.base_path` prefix every route the respective server serves,
without changing the listen host/port. Set this when maelsink sits behind a reverse
proxy at a subpath rather than its own subdomain — e.g. `web.base_path: /maelsink` makes
the UI reachable at `https://your-proxy/maelsink/` instead of `/`. See
[Running maelsink behind HAProxy](/maelsink/docs/guides/haproxy/) for a full worked example.

## The `/ui-api` vs `/api/v1` split

Two API surfaces are served on the Web UI port:

- **`/api/v1/*`** — the stable, public REST API. This is the same surface exposed on the
  dedicated API port (`api.port`), and the one documented in the
  [REST API Reference](/maelsink/docs/rest-api-reference/). Build integrations against this.
- **`{base_path}/ui-api/v1/*`** — internal endpoints used only by the bundled Web UI
  itself (session info, UI-specific config, the `/ws` WebSocket feed). Not a supported
  integration surface — it can change between releases without notice.

## CORS

`web.cors_origins` (a list) controls which origins may make cross-origin requests to the
Web UI's APIs. Leave it empty (the default) unless you're serving a separate frontend
that needs to call maelsink directly from the browser.

## API auth

Setting `api.auth.enabled: true` requires every `/api/v1/*` request (on either the API
port or the Web UI port) to carry `Authorization: Bearer <api.auth.api_key>`. This is
independent of the Web UI's own Basic Auth (`web.auth.file`) — see
[Password files](/maelsink/docs/configuration/password-files/).
