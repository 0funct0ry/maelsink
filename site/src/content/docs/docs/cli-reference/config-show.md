---
title: "config show"
description: "Print the fully-resolved configuration (defaults + file + env + flags) that `serve` would use."
---

Print the fully-resolved configuration (defaults + file + env + flags) that `serve` would use.

This page is generated directly from `maelsink config show --help`. The flag list below is always in sync with the binary and is never hand-maintained.

```
Loads maelsink.yaml (or --config), layers MAELSINK_* env vars and any flags given here, and prints the result as YAML.

Usage:
  maelsink config show [flags]

Flags:
  -k, --api-auth-api-key string                REST API bearer key
  -y, --api-auth-enabled                       require a bearer API key on /api/v1
  -B, --api-base-path string                   REST API reverse-proxy base path
  -A, --api-host string                        REST API listen host (default "127.0.0.1")
  -o, --api-port int                           REST API listen port (default 9090)
  -d, --db string                              path to the SQLite database file (default "./maelsink.db")
  -u, --headless                               shorthand for --web-enabled=false (headless mode)
  -h, --help                                   help for show
  -g, --retention-max-age-hours int            max message age in hours (0 = unlimited)
  -M, --retention-max-messages int             max stored messages (0 = unlimited)
  -i, --retention-sweep-interval-minutes int   retention sweeper interval in minutes (default 5)
  -T, --server-shutdown-timeout-seconds int    graceful shutdown timeout in seconds (default 15)
  -Y, --smtp-auth-accept-any                   accept any AUTH PLAIN/LOGIN credentials, including none (test/CI use only)
  -I, --smtp-auth-allow-insecure               permit plaintext AUTH PLAIN/LOGIN without STARTTLS/TLS (RFC 4954; local/CI use only)
  -a, --smtp-auth-enabled                      require AUTH PLAIN/LOGIN on the SMTP server
  -f, --smtp-auth-file string                  path to htpasswd-style multi-user credential file for SMTP AUTH (disabled if empty)
  -W, --smtp-auth-password string              SMTP AUTH password
  -U, --smtp-auth-username string              SMTP AUTH username
  -m, --smtp-domain string                     HELO/EHLO advertised domain (default "maelsink.local")
  -H, --smtp-host string                       SMTP listen host (default "127.0.0.1")
  -s, --smtp-max-message-size-mb int           max accepted message size in MB (default 25)
  -p, --smtp-port int                          SMTP listen port (default 1025)
  -R, --smtp-require-starttls                  require STARTTLS before MAIL FROM/AUTH
  -S, --smtp-require-tls                       require implicit TLS from connect (disables STARTTLS on this listener)
  -C, --smtp-tls-cert string                   path to PEM certificate enabling SMTP STARTTLS (both cert+key required together; disabled if empty)
  -K, --smtp-tls-key string                    path to PEM private key enabling SMTP STARTTLS (both cert+key required together; disabled if empty)
  -x, --storage-attachments-disk-path string   directory for on-disk attachment storage (default "./attachments")
  -n, --storage-attachments-store-on-disk      store attachments on disk instead of as SQLite BLOBs
  -r, --storage-driver string                  storage driver (default "sqlite")
  -L, --web-auth-file string                   path to htpasswd-style basic-auth file for the Web UI (disabled if empty)
  -b, --web-base-path string                   Web UI reverse-proxy base path
  -O, --web-cors-origins strings               allowed CORS origins for the Web UI server
  -e, --web-enabled                            enable the Web UI server (default true)
  -w, --web-host string                        Web UI listen host (default "127.0.0.1")
  -P, --web-port int                           Web UI listen port (default 8080)
  -q, --web-tls-cert string                    path to PEM certificate for Web UI HTTPS (both cert+key required together; disabled if empty)
  -z, --web-tls-key string                     path to PEM private key for Web UI HTTPS (both cert+key required together; disabled if empty)

Global Flags:
  -c, --config string       path to maelsink.yaml
  -j, --log-file string     log file path (empty = stdout only)
  -F, --log-format string   log format (text|json)
  -l, --log-level string    log level (debug|info|warn|error)
```
