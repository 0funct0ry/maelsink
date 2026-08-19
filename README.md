# maelsink

A single-binary fake SMTP server for local development and CI. It accepts
any mail over SMTP and lets you inspect it — it never relays mail anywhere.

## Install

Pick one of four ways to get `maelsink`, all built from the same tagged
release so every install path ships with a working, embedded Web UI:

1. **Download a binary** from the [GitHub Releases](https://github.com/0funct0ry/maelsink/releases) page — pick the archive matching your OS/arch, extract, run.
2. **Homebrew** (macOS/Linux):
   ```bash
   brew install 0funct0ry/maelsink/maelsink
   ```
3. **Scoop** (Windows):
   ```powershell
   scoop bucket add maelsink https://github.com/0funct0ry/scoop-maelsink
   scoop install maelsink
   ```
4. **Docker**:
   ```bash
   docker run --rm -p 1025:1025 -p 8080:8080 -p 9090:9090 \
     -v maelsink-data:/data ghcr.io/0funct0ry/maelsink:latest
   ```
   Or via Compose: `docker compose up` (see [docker-compose.yml](docker-compose.yml)).

`go install` is deliberately not offered — it skips the frontend build step,
so a binary built that way would ship without a working Web UI.

Releases (binaries/Homebrew/Scoop) are cut via a manually triggered GitHub
Actions workflow after a tag is pushed; the Docker image is published as a
separate manual step.

## Prerequisites

- Go 1.26+

## Build & run

```bash
make build
make run
```

This starts maelsink with its default configuration:

| Service  | Default address                |
|----------|---------------------------------|
| SMTP     | `127.0.0.1:1025`                |
| Web UI   | `http://127.0.0.1:8080/`        |
| REST API | `http://127.0.0.1:9090/api/v1`  |

Point your application's SMTP client at `localhost:1025` and send mail —
it'll show up in maelsink.

By default (no `-d`/`--db` given, and no `storage.path` set via config file or
env var) maelsink stores messages in a transient **in-memory** SQLite
database — nothing is written to disk, and all messages are lost when the
process exits. The startup log states which storage mode is active. Pass
`--db <path>` (e.g. `--db ./maelsink.db`) for a persistent, file-backed
database; passing `--db` with no value at all (or `--db ""` explicitly)
falls back to the default file, `./maelsink.db`, in the current directory.

By default maelsink only listens on `127.0.0.1` (loopback) — it's a
development tool and isn't meant to be exposed on a network. To bind
elsewhere (e.g. `0.0.0.0` inside a container), set `--smtp-host`,
`--web-host`, `--api-host` (or the matching `MAELSINK_*_HOST` env vars /
config keys) explicitly, and enable `smtp.auth`/`api.auth` first if you do.

## Configuration

Generate a default config file, then inspect the effective (fully resolved)
config at any time:

```bash
./bin/maelsink config init    # writes ./maelsink.yaml
./bin/maelsink config show    # prints the resolved config as YAML
```

Configuration can also be set via `MAELSINK_*` environment variables or CLI
flags — run `./bin/maelsink --help` or `./bin/maelsink serve --help` for the
full flag list.

## Other useful commands

```bash
make test   # run the test suite
make fmt    # format Go source
make vet    # run go vet
```

## Documentation

- [docs/cli-smtp-tls.md](docs/cli-smtp-tls.md) — using `maelsink send`/`shell` against a TLS/auth-hardened SMTP listener
- [docs/deployment.md](docs/deployment.md) — deployment guide
- [docs/reverse-proxy.md](docs/reverse-proxy.md) — running the Web UI behind a reverse proxy
