---
title: "shell"
description: "Start an interactive maelsink shell — a REPL client of the REST API and SMTP port."
---

Start an interactive maelsink shell — a REPL client of the REST API and SMTP port.

This page is generated directly from `maelsink shell --help`. The flag list below is always in sync with the binary and is never hand-maintained.

```
Starts maelsink's interactive shell: a readline-style REPL with alias/
abbreviation/template expansion, a builtin command table (list/show/delete/
clear/export/send/stats/health/...), and non-interactive scripting modes
(-e/--execute, --script). It is a pure client of the /api/v1 REST surface
and SMTP port — it never talks to storage directly.

Usage:
  maelsink shell [flags]

Flags:
  -G, --abbr-trigger-key string         space|tab|enter|none (default "space")
  -A, --api string                      maelsink REST API base URL (default "http://127.0.0.1:9090")
  -k, --api-key string                  REST API bearer key (if api.auth.enabled)
  -W, --auth-pass string                SMTP AUTH password
  -U, --auth-user string                SMTP AUTH username
  -L, --color string                    auto|always|never (default "auto")
  -x, --command-prefix string           builtin prefix: "", ".", ":" or "/"
  -c, --config string                   path to maelsink.yaml
  -E, --editor string                   "" = $VISUAL, $EDITOR, vi/notepad
  -e, --execute stringArray             run a command and exit (repeatable)
  -Q, --exit-on-error                   abort a script on first failure
  -f, --format string                   session default output format: table|json|yaml (default "table")
  -h, --help                            help for shell
  -Y, --history-file string             "" = platform default
  -z, --history-size int                max history lines kept (default 5000)
  -N, --no-template                     shorthand for --template-enabled=false
  -R, --prompt string                   prompt template (default "maelsink{{ if not .connected }} (offline){{ end }}> ")
  -s, --script string                   run a file of commands and exit
  -S, --seed int                        template PRNG seed (0 = random per session)
  -X, --sh-enabled                      allow the sh builtin (default true)
  -H, --smtp-host string                default SMTP target for the send builtin (default "127.0.0.1")
  -p, --smtp-port int                   default SMTP target for the send builtin (default 1025)
      --smtp-tls string                 default transport security for the send builtins: none|starttls|implicit (default "none")
      --smtp-tls-insecure-skip-verify   accept a self-signed/dev SMTP TLS certificate without verification (local/CI use only)
  -t, --template-enabled                enable {{ }} template expansion (default true)
  -Z, --template-unsafe-funcs           re-enable env/expandenv/getHostByName
```
