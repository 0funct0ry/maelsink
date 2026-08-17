---
title: Deleting Messages
description: Delete a single message, clear the whole inbox, and the non-interactive confirmation gate on destructive CLI commands.
---

maelsink offers three ways to delete messages: one at a time via `maelsink delete`, everything at once via `maelsink clear`, and the shell's `delete`/`clear` builtins (thin wrappers over the same REST endpoints).

## Deleting a single message

```
maelsink delete <id>
```

Live example:

```
$ maelsink send --to alice@example.com --from bob@example.com --subject "Password reset" --text hi
message sent

$ maelsink list
ID                        FROM             TO                  SUBJECT         ...
bed302ae42d8d1b913c1c999  bob@example.com  alice@example.com   Password reset  ...

$ maelsink delete bed302ae42d8d1b913c1c999
deleted bed302ae42d8d1b913c1c999
```

This maps directly to `DELETE /api/v1/messages/{id}`. There's no confirmation prompt for single-message delete — it's a targeted, low-blast-radius operation.

## Clearing everything

```
maelsink clear
```

Unlike `delete`, `clear` removes every stored message (`DELETE /api/v1/messages?confirm=true`), so it prompts for confirmation by default:

```
$ maelsink clear
This will delete 3 messages. Continue? [y/N] y
All messages deleted.
```

### Skipping the prompt with `--yes`/`-y`

For scripts and CI, pass `--yes` (or `-y`) to skip the interactive prompt entirely:

```
$ maelsink clear --yes
All messages deleted.
```

Verified live — after sending a message and running `maelsink clear --yes`, `GET /api/v1/messages` immediately reported `"total": 0`.

:::caution
`clear` is irreversible: it deletes every message in the store, not just the ones matching a filter. There's no `--from`/`--subject` scoping on the top-level `clear` command — for a filtered bulk delete, use the shell's richer `export --all` to snapshot before you clear, or delete a curated list of IDs individually.
:::

## The `--all` flag (shell only)

The top-level `maelsink delete` command deletes exactly one message by ID and has no `--all` flag. Bulk delete-by-filter lives on the shell's `delete` builtin instead, where `--all` (`internal/shell/builtin/delete.go`) is documented as "delete every message (same as clear)" and shares its implementation with the `clear` builtin (`internal/shell/builtin/clear.go`) — both route through the same `runClear` helper, which:

- in **non-interactive** mode (`-e`/`--script`/piped stdin), hard-errors immediately unless `-y`/`--yes` is given — there is no prompt to answer;
- in **interactive** mode, fetches the current message count via `stats` and shows the same `"This will delete N messages. Continue? [y/N]"` prompt as the top-level `clear` command, unless `-y`/`--yes` is passed to skip it.

```
maelsink shell -e 'delete --all --yes'
maelsink shell -e 'clear --yes'
```

Verified live: `maelsink shell --api ... --smtp-port ... -e 'clear --yes'` printed `all messages deleted` and the message count immediately dropped to 0.

## Summary

| Command | Scope | Confirmation |
|---|---|---|
| `maelsink delete <id>` | One message | None needed |
| `maelsink clear` | Every message | Interactive prompt, or `--yes`/`-y` to skip |
| shell `delete --all` | Every message | Prompt if interactive; requires `-y`/`--yes` if non-interactive |
| shell `clear` | Every message | Same as `delete --all` (shares implementation) |
