---
title: Advanced Search Patterns
description: The full SQLite FTS5 query syntax supported by the Web UI's search bar, including boolean operators, column scoping, and proximity search.
---

The Web UI's search bar transparently exposes SQLite's FTS5 query grammar. Every pattern on this page is real FTS5 syntax, not a maelsink-specific extension, and works identically whether typed into the search bar or passed as the `q` query parameter to `GET /api/v1/messages`.

The indexed columns are `subject`, `from_addr`, `to_addrs`, and `text_body`. See [Filters and Search](/maelsink/docs/usage/filters-and-search/) for basic keyword and phrase search.

An optional fixture-data generator (`scripts/search.py`) can populate a local `maelsink serve` instance with sample messages for experimenting with these patterns:

```bash
maelsink serve                       # in one terminal
python3 scripts/search.py            # in another; sends sample emails via SMTP
```

## Boolean operators: AND, OR, NOT

FTS5 supports explicit, uppercase boolean operators between terms:

```
invoice AND acme             matches messages mentioning both terms
acme OR globex                matches messages mentioning either term
certificate NOT urgent        matches messages mentioning the first term but not the second
```

FTS5 does not support a `-token` shorthand for exclusion. A bare `-word` is interpreted as column-filter syntax and fails to parse. maelsink's API classifies any FTS5 syntax failure as a search-query error and returns a `400 invalid_query` response with a generic message rather than the underlying SQLite error text. `NOT` must always be spelled out.

## Column-scoped search

Prefixing a term with one of the four indexed column names and a colon scopes the match to that field:

```
subject:migration
from_addr:globex
to_addrs:pixelforge
text_body:configuration
```

`from_addr` and `to_addrs` are tokenized on punctuation such as `.` and `@`, so `globex` matches `globex.io` (tokenized as `globex` plus `io`), while `acme` does not match `acmecorp.com` (tokenized as the single token `acmecorp`). Searching a sender's company name as it appears in prose and searching their email domain are different queries with different results.

## Prefix wildcard

A trailing `*` matches any token starting with that prefix:

```
invoic*      matches invoice, invoiced, invoicing, and other forms
config*      matches configure, configuration, and other forms
```

This is useful for word-form variants, such as singular and plural forms or different verb tenses, without enumerating each one.

## Proximity search: NEAR

`NEAR(term1 term2, N)` matches when both terms appear within `N` tokens of each other, in either order, anywhere in the row:

```
NEAR(refund order, 5)
```

Reducing `N` tightens the match to require the terms to appear closer together, excluding messages where the terms are only loosely related.

## Grouping with parentheses

Boolean operators can be combined with explicit precedence using parentheses:

```
(receipt OR invoice) AND acme
```

This pattern, matching either of two related terms combined with a required term, is a common candidate for a saved search, since it is easy to get the operator precedence or exact spelling wrong when retyping it.

## Quoting hyphenated and apostrophized terms

A bare hyphen or apostrophe inside an unquoted query term is not treated as a literal character by the FTS5 parser; it is interpreted as syntax and causes a parse error:

```
multi-factor       →  400 invalid_query (bare hyphen breaks the parser)
"multi-factor"     →  matches correctly
multi factor       →  also matches correctly, as two unquoted tokens

can't              →  400 invalid_query (bare apostrophe starts an unterminated quote)
"can't"            →  matches correctly
```

A query term containing a hyphen, apostrophe, or other punctuation should be wrapped in double quotes. This has no effect when the term does not require it, and avoids a parse error when it does. Any malformed FTS5 syntax, including these punctuation cases, surfaces as the same generic `400 invalid_query` response.

## Unicode and diacritics

FTS5's default tokenizer folds diacritics, so accented and unaccented spellings match the same content:

```
cafe
café
```

Both queries return the same results, which is useful when a keyboard or locale does not make typing an accented character convenient, or when a sender's own capitalization or accent usage is uncertain.

## Numbers and alphanumeric codes

Order numbers, invoice identifiers, ticket references, and one-time codes are indexed and searchable like any other token, including as a column-scoped search when the field containing the code is known:

```
INV-2024-00987
482913
TCK-10432
```
