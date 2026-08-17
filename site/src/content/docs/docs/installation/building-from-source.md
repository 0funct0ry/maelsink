---
title: Building from Source
description: Build maelsink from source with Go 1.26.4, for readers who need a non-release build.
---

If you need a build that isn't covered by [Installation](/maelsink/docs/installation/)'s four
release paths — a specific commit, a local patch, or a platform without a published
release — build from source.

## Prerequisites

- Go **1.26.4** (the toolchain version this repo is pinned to)
- `git`
- Node.js and `npm`, only if you want the embedded web UIs built from source too (see
  below) — the Go binary itself builds without them once assets are already present.

## Clone and build

```sh
git clone https://github.com/0funct0ry/maelsink.git
cd maelsink
make build
```

`make build` runs three steps in order: `build-web`, `build-web-compose`, then
`build-go`. The resulting binary is at `bin/maelsink`.

## What each build target does

- **`build-web`** — builds the main product web UI (React/Vite) under `web/`, if that
  directory exists, and its output is embedded into the Go binary via `go:embed`.
- **`build-web-compose`** — builds the `maelsink compose` playground UI under
  `web-compose/`, likewise embedded via `go:embed`.
- **`build-go`** — compiles the Go binary itself with `CGO_ENABLED=0`, embedding
  version/commit/build-date via `-ldflags`. This target depends on
  `ensure-web-embed`/`ensure-web-compose-embed`, which only rebuild the web assets if
  `internal/webui/dist/index.html` / `internal/compose/dist/index.html` are missing —
  so if you've already built the web assets once, `make build-go` alone is fast.

If you only care about the Go binary and don't want to touch the frontend toolchains at
all, run:

```sh
make build-go
```

This still requires the embedded asset directories to exist at least once (they're
checked into build output, not source control) — the `ensure-web-embed` /
`ensure-web-compose-embed` prerequisites will build them automatically on first run if
`web/` and `web-compose/` are present.

## Site build (this docs/marketing site)

The `site/` directory (this Astro + Starlight project) has its own build, wired via the
`site-dev`/`site-build` Makefile targets — it is independent of `make build` and not
required to build the `maelsink` binary itself.

## Verifying your build

Once `bin/maelsink` exists, see
[Testing maelsink](/maelsink/docs/installation/testing-maelsink/) to confirm it works.
