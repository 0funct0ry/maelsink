---
title: CLI Reference
description: Every maelsink command, generated directly from its real --help output.
---

Every page in this section is generated straight from `maelsink <command>
--help` as a build step (`site/scripts/gen-cli-docs.mjs`), so the flags and
defaults you see always match the actual binary — never hand-transcribed.

## Commands

- [`serve`](/maelsink/docs/cli-reference/serve/) — start the SMTP server, Web UI, and REST API
- [`send`](/maelsink/docs/cli-reference/send/) — send a test message over SMTP
- [`list`](/maelsink/docs/cli-reference/list/) — list/filter captured messages
- [`get`](/maelsink/docs/cli-reference/get/) — show one message's detail
- [`delete`](/maelsink/docs/cli-reference/delete/) — delete one message
- [`clear`](/maelsink/docs/cli-reference/clear/) — delete all messages
- [`export`](/maelsink/docs/cli-reference/export/) — download a message as `.eml`
- [`shell`](/maelsink/docs/cli-reference/shell/) — interactive/scriptable REPL
- [`compose`](/maelsink/docs/cli-reference/compose/) — start the local browser composing UI
- [`config init`](/maelsink/docs/cli-reference/config-init/) — write a default `maelsink.yaml`
- [`config show`](/maelsink/docs/cli-reference/config-show/) — print the fully resolved config
- [`config validate`](/maelsink/docs/cli-reference/config-validate/) — validate a config file
- [`auth adduser`](/maelsink/docs/cli-reference/auth-adduser/) — add/update a Basic Auth user
- [`auth removeuser`](/maelsink/docs/cli-reference/auth-removeuser/) — remove a Basic Auth user
- [`version`](/maelsink/docs/cli-reference/version/) — print build provenance

See [Runtime Options](/maelsink/docs/configuration/runtime-options/) for how
every flag interacts with `maelsink.yaml` and `MAELSINK_*` environment
variables.
