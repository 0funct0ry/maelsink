---
title: Installation
description: Four ways to install maelsink — binary download, Homebrew, Scoop, or Docker.
---

There are four supported ways to get maelsink running. Pick whichever fits your platform
and workflow.

## Binary download

Grab a per-OS/arch tarball from the
[latest GitHub Release](https://github.com/0funct0ry/maelsink/releases/latest), extract
it, and put the `maelsink` binary on your `PATH`.

## Homebrew (macOS/Linux)

```sh
brew tap 0funct0ry/maelsink   # or: brew install 0funct0ry/maelsink/maelsink
brew install maelsink
```

## Scoop (Windows)

```powershell
scoop bucket add maelsink https://github.com/0funct0ry/scoop-maelsink
scoop install maelsink
```

## Docker

```sh
docker run --rm -p 1025:1025 -p 8080:8080 -p 9090:9090 \
  ghcr.io/0funct0ry/maelsink:latest
```

Images are published to GHCR as `ghcr.io/0funct0ry/maelsink`, tagged both `:latest` and
with the exact release version (e.g. `:1.2.3`).

## Next steps

- Building from source instead? See [Building from Source](/maelsink/docs/installation/building-from-source/).
- Verify your install works: [Testing maelsink](/maelsink/docs/installation/testing-maelsink/).
- Then head to [Getting Started](/maelsink/docs/getting-started/).
