---
title: Filters and Search
description: Using the Web UI's search bar and sidebar filters to find captured messages.
---

The Web UI's top search bar (`web/src/components/inbox/SearchBar.tsx`) is a single free-text field. It debounces input for 300ms and sends the query to `GET /api/v1/messages` as the `q` parameter, which the REST API forwards directly into SQLite's FTS5 `MATCH` operator.

This means the search bar is not a simple substring filter; it exposes the full SQLite FTS5 query grammar. See [Advanced Search Patterns](/maelsink/docs/usage/advanced-search-patterns/) for the complete syntax, including boolean operators, column scoping, and proximity search.

The indexed columns are `subject`, `from_addr`, `to_addrs`, and `text_body`. An unscoped query matches across all four.

## Basic keyword search

A single word matches any of the four indexed columns, case-insensitively:

```
invoice
```

This matches messages where the term appears anywhere in the subject, sender, recipient, or body, not only the subject.

## Phrase search

Multiple words without quotes are implicitly combined with `AND` across the entire row, which can match unrelated messages that each contain one of the words. Quoting the phrase requires the words to appear adjacent and in order:

```
database migration          →  matches any message containing both words anywhere
"database migration"        →  matches only messages containing the exact phrase
```

## Combining search with sidebar filters

The search bar's `q` parameter is one field of the message list's overall query object. The sidebar's mailbox filters (unread, has attachments, parse warnings) and tag selection merge into the same query, and a saved search snapshots the entire combination, not just the search text. For example:

- A boolean text query combined with a tag filter, such as an invoice-or-receipt search scoped to a specific vendor's tag.
- A prefix search combined with the attachments filter, to find messages that reference an invoice and actually have one attached.
- A proximity search combined with the unread filter, to find recent, untriaged messages matching a pattern.

Saving a search preserves this combination under a name in the sidebar, so it does not need to be reconstructed by hand.

## Equivalent access from the CLI and shell

The same filtering is available outside the Web UI. See [Using CLI](/maelsink/docs/usage/using-cli/) for `maelsink list` and the REST API, and [Using Shell](/maelsink/docs/usage/using-shell/) for the `list` builtin in `maelsink shell`.
