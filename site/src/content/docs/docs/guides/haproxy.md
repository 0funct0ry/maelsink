---
title: Running maelsink behind HAProxy
description: Reverse-proxying maelsink's Web UI/API with HAProxy at a subpath, with TLS terminated at the proxy.
---

If maelsink is one service among several behind a shared HAProxy front end, you'll
typically want TLS terminated at the proxy and maelsink itself reachable at a subpath
like `/maelsink/`.

## Configure `base_path`

Tell maelsink's Web UI (and its mounted `/api/v1`) to serve everything under a prefix,
so generated links and asset paths match where the proxy is routing requests:

```sh
maelsink serve --headless=false --web-base-path /maelsink --web-host 127.0.0.1 --web-port 8080
```

See [Web UI and API server](/maelsink/docs/configuration/web-ui-and-api-server/) for how
`web.base_path`/`api.base_path` work, and
[Runtime options](/maelsink/docs/configuration/runtime-options/) for the full config table.

## Minimal `haproxy.cfg`

```
global
    log stdout format raw local0

defaults
    mode http
    timeout connect 5s
    timeout client  30s
    timeout server  30s
    log global

frontend https_in
    bind *:443 ssl crt /etc/haproxy/certs/example.com.pem
    # Route anything under /maelsink/ to the maelsink backend.
    acl is_maelsink path_beg /maelsink
    use_backend maelsink_web if is_maelsink

backend maelsink_web
    server maelsink1 127.0.0.1:8080 check
```

TLS is terminated at the `https_in` frontend (via the `ssl crt` bind option); HAProxy
forwards plain HTTP to maelsink on `127.0.0.1:8080`, so maelsink itself needs no
`web.tls.*` configuration in this setup. See
[TLS certificates](/maelsink/docs/configuration/tls-certificates/) if you'd rather have maelsink
terminate TLS itself instead of relying on the proxy.

## Proxying the SMTP port

SMTP isn't HTTP, so it can't go through an `http`-mode HAProxy frontend/backend — it
needs a `tcp`-mode passthrough:

```
frontend smtp_in
    bind *:1025
    mode tcp
    default_backend maelsink_smtp

backend maelsink_smtp
    mode tcp
    server maelsink1 127.0.0.1:1025 check
```

If you need STARTTLS to be visible end-to-end (rather than terminated at the proxy),
keep this frontend in `tcp` mode and configure `smtp.tls_cert`/`smtp.tls_key` on
maelsink itself — see [SMTP server](/maelsink/docs/configuration/smtp-server/).
