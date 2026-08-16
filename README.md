# maelsink

A single-binary fake SMTP server for local development and CI. It accepts
any mail over SMTP and lets you inspect it — it never relays mail anywhere.

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
