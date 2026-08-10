# Deployment topologies

maelsink supports three common deployment shapes.

## a) All-in-one local dev

Run the default `maelsink serve` (or bare `maelsink`) with no flags: all
three listeners bind on `127.0.0.1` — SMTP on `1025`, the Web UI on `8080`,
and the REST API on `9090`. Point your app under test at
`localhost:1025` for SMTP and open `http://localhost:8080` to inspect mail.

```bash
maelsink serve
```

## b) Headless CI usage

For CI pipelines or automated tests, run maelsink with `--headless`
(equivalent to `--web-enabled=false`): only the SMTP and REST API listeners
bind — the Web UI server is not constructed or bound at all. Tests interact
with maelsink purely through `/api/v1` (list/get/delete messages), typically
in an ephemeral container started fresh per test run.

```bash
maelsink serve --headless --db /tmp/maelsink.db
```

The startup banner omits the Web UI line in this mode, and the Web UI port
is not merely returning 404 — nothing is listening there at all.

## c) Shared team instance behind a reverse proxy subpath

For a long-running shared instance reachable by a team, run maelsink behind
a reverse proxy at a subpath (e.g. `https://tools.example.com/maelsink/`),
using `--web-base-path` or the zero-config `X-Forwarded-Prefix` fallback.
See [reverse-proxy.md](reverse-proxy.md) for concrete HAProxy and nginx
configs.

```bash
maelsink serve --web-host 0.0.0.0 --web-base-path=/maelsink
```

Bind `--web-host`/`--api-host` to `0.0.0.0` (or the container's interface)
so the proxy can reach maelsink, and keep the proxy itself as the only
public-facing entry point.
