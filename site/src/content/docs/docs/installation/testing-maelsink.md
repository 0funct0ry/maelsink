---
title: Testing maelsink
description: Verify your maelsink install or build actually works.
---

Once you've installed or built maelsink, use these checks to confirm it's working before
you build anything on top of it. (This page is about verifying *your* install — see
[Integration testing](/maelsink/docs/integration-testing/) for using maelsink as a sink inside
someone else's CI pipeline.)

## Check the version

```sh
maelsink version
```

```
maelsink version 1.2.3 (commit abcdef0, built 2026-01-01T00:00:00Z)
```

Or as JSON, useful for scripting:

```sh
maelsink version --json
```

```json
{"version":"1.2.3","commit":"abcdef0","build_date":"2026-01-01T00:00:00Z","go":"go1.26.4"}
```

## Run it and hit the health endpoint

```sh
maelsink serve &
curl -s http://127.0.0.1:9090/api/v1/health
```

```json
{"db":"ok","smtp":"listening","status":"ok"}
```

A `200` response with `"status":"ok"` means the SMTP listener is up and the database
connection is healthy — everything is working end-to-end.

## Run the test suite (source builds only)

If you built from source (see [Building from Source](/maelsink/docs/installation/building-from-source/)),
you can also run maelsink's own test suite from the repo root:

```sh
make test
```

This runs `go test -v -race ./...` across every package. This confirms the *source tree*
is healthy, which is only relevant if you're building from source or contributing —
release/Homebrew/Scoop/Docker installs don't need this step.
