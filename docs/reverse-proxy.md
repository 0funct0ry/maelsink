# Running maelsink behind a reverse proxy

maelsink's Web UI can be hosted at a subpath (e.g.
`https://tools.example.com/maelsink/`) behind a reverse proxy, per
[SPEC.md §3.4](../internal-docs/SPEC.md).

Two ways to configure the subpath:

- **Explicit** — set `web.base_path` (config file), `MAELSINK_WEB_BASE_PATH`
  (env var), or `--web-base-path` (CLI flag) to the subpath, e.g.
  `/maelsink`. This always wins.
- **Zero-config fallback** — leave `web.base_path` unset and have the proxy
  send an `X-Forwarded-Prefix` header with the subpath. maelsink resolves the
  base path per request from that header, including for the `/ws` endpoint.

Either way, the embedded SPA's `<base href>` and all asset/API/WebSocket URLs
are generated relative to the resolved base path — no rebuild required to
move maelsink to a different subpath.

## HAProxy

```haproxy
frontend fe_tools
    bind *:443 ssl crt /etc/haproxy/certs/tools.pem
    acl is_maelsink path_beg /maelsink
    use_backend be_maelsink if is_maelsink

backend be_maelsink
    mode http
    option forwardfor
    http-request set-header X-Forwarded-Prefix /maelsink
    http-request set-header X-Forwarded-Proto https if { ssl_fc }
    # WebSocket upgrade (required for the live-update /ws endpoint)
    http-request set-header Connection upgrade if { req.hdr(Upgrade) -i websocket }
    timeout tunnel 1h
    server maelsink1 127.0.0.1:8080 check
```

Start maelsink without `--web-base-path` and let the `X-Forwarded-Prefix`
header above drive the subpath, or set `--web-base-path=/maelsink`
explicitly for the same result without relying on the proxy header.

## nginx

```nginx
location /maelsink/ {
    proxy_pass http://127.0.0.1:8080/maelsink/;
    proxy_set_header X-Forwarded-Prefix /maelsink;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host  $host;

    # WebSocket upgrade for /ws — required, easy to forget
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

As with HAProxy, either rely on `X-Forwarded-Prefix` above, or pass
`--web-base-path=/maelsink` to the maelsink process and drop the header.

## Verifying the setup

1. Start maelsink: `maelsink serve --web-base-path=/maelsink` (or with the
   zero-config fallback via your proxy's headers instead).
2. Load `https://your-proxy/maelsink/` — the SPA should load and its assets
   should resolve under `/maelsink/...`.
3. Confirm the live inbox updates (WebSocket) work — the browser dev tools'
   Network tab should show a successful `wss://your-proxy/maelsink/ws`
   upgrade.
4. Confirm REST calls succeed at `https://your-proxy/maelsink/api/v1/...`.
