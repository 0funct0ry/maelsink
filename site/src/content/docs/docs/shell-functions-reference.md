---
title: "Shell Functions Reference"
description: "Every template function available to maelsink shell templates, generated from the template function registry."
---

This page is generated from `internal/shell/tmpl`'s registry (`go run ./tools/docgen`) — descriptions and signatures are always in sync with the engine; they are never hand-maintained.

:::caution
Functions marked **unsafe** below (`env`, `expandenv`, `getHostByName`) are gated behind the shell's `--template-unsafe-funcs` / `-Z` flag (or the `shell.template_unsafe_funcs` config key). They are disabled by default because they can read host environment variables or perform DNS lookups from inside a template.
:::

## identifiers

### `uuid`

`uuid()` -> string

Random UUIDv4, seeded from the session's PRNG.

### `uuidv7`

`uuidv7()` -> string

Time-ordered UUIDv7. NOT reproducible under a fixed --seed: its rand_a field derives from a package-level wall-clock counter, not the seeded entropy source.

### `ulid`

`ulid()` -> string

Lexicographically-sortable ULID. NOT reproducible under a fixed --seed: its 48-bit timestamp component is real wall-clock time by design.

### `nanoid`

`nanoid([size])` -> string

URL-safe random ID, default length 21.

### `objectid`

`objectid()` -> string

24-hex-char MongoDB-style ObjectID (4-byte timestamp + 5 random + 3-byte counter).

### `ksuid`

`ksuid()` -> string

27-char base62 K-Sortable ID (4-byte timestamp + 16 random bytes).

### `messageID`

`messageID([domain])` -> string

RFC 5322 email Message-Id value, e.g. <hex@domain>. domain defaults to "maelsink.local".

## generate

### `randInt`

`randInt(min, max int)` -> int

Random int in [min,max].

### `randFloat`

`randFloat(min, max float64 [, decimals int])` -> float64

Random float64 in [min,max], optionally rounded to `decimals` places.

### `randBool`

`randBool()` -> bool

Random true/false.

### `randString`

`randString(n int [, charset string])` -> string

Random string of length n from charset (default alphanumeric).

### `randBytes`

`randBytes(n int)` -> []byte

n random bytes.

### `pick`

`pick(a, b, c, ...)` -> any

One of its comma-separated arguments, chosen at random.

### `oneOf`

`oneOf(a, b, c, ... | "a,b,c")` -> any

One of its arguments, chosen at random. Accepts either several comma-separated arguments ({{ oneOf "a" "b" "c" }}) or a single string split on "," ({{ oneOf "a,b,c" }}).

### `shuffle`

`shuffle(list)` -> list

Returns list with its elements in random order.

### `regex`

`regex(pattern)` -> string

A string matching the given RE2 pattern. Accepts a bare, unquoted pattern in template source (e.g. {{ regex [a-z]{2,4} }}) as well as the normal quoted form.

### `fName`

`fName()` -> string

Random full name.

### `fFirstName`

`fFirstName()` -> string

Random first name.

### `fLastName`

`fLastName()` -> string

Random last name.

### `fUsername`

`fUsername()` -> string

Random username.

### `fPhone`

`fPhone()` -> string

Random phone number.

### `fAddress`

`fAddress()` -> string

Random street address.

### `fStreet`

`fStreet()` -> string

Random street name.

### `fCity`

`fCity()` -> string

Random city name.

### `fState`

`fState()` -> string

Random US state.

### `fZip`

`fZip()` -> string

Random ZIP/postal code.

### `fCountry`

`fCountry()` -> string

Random country name.

### `fDomain`

`fDomain()` -> string

Random domain name.

### `fURL`

`fURL()` -> string

Random URL.

### `fIPv4`

`fIPv4()` -> string

Random IPv4 address.

### `fIPv6`

`fIPv6()` -> string

Random IPv6 address.

### `fMAC`

`fMAC()` -> string

Random MAC address.

### `fUserAgent`

`fUserAgent()` -> string

Random browser User-Agent string.

### `fCompany`

`fCompany()` -> string

Random company name.

### `fJobTitle`

`fJobTitle()` -> string

Random job title.

### `fWord`

`fWord()` -> string

A single random word.

### `fSentence`

`fSentence()` -> string

A random ~10-word sentence.

### `fParagraph`

`fParagraph([n])` -> string

n random paragraphs (default 1), separated by blank lines.

### `fSubject`

`fSubject()` -> string

A random ~6-word email-subject-like sentence.

### `fHTMLBody`

`fHTMLBody([paragraphs])` -> string

Random HTML body: `paragraphs` <p> blocks (default 3).

### `fTextBody`

`fTextBody([paragraphs])` -> string

Random plain-text body: `paragraphs` blocks (default 3).

### `fCreditCard`

`fCreditCard([type])` -> dict

dict{number,type,cvv,exp} — Luhn-valid test card.

### `fTransaction`

`fTransaction()` -> dict

dict{id,amount,currency,status,timestamp,merchant}.

### `fProduct`

`fProduct()` -> dict

dict{sku,name,category,price,currency,qty}.

### `fOrder`

`fOrder([items])` -> dict

dict{id,items,total,currency,created} — `items` fProduct line items (default 1-3).

### `fInvoice`

`fInvoice([items])` -> dict

dict{invoiceNumber,order,subtotal,tax,total,currency,issued,dueDate,billTo}.

### `fPNG`

`fPNG([w] [h])` -> string

Generates a PNG image (default 64x64) and returns its path.

### `fJPEG`

`fJPEG([w] [h])` -> string

Generates a JPEG image (default 64x64) and returns its path.

### `fGIF`

`fGIF([w] [h])` -> string

Generates a GIF image (default 64x64) and returns its path.

### `fCSV`

`fCSV([rows] [cols])` -> string

Generates a CSV file (default 10x5) and returns its path.

### `fZIP`

`fZIP([files...])` -> string

Bundles the given paths (or one generated file) into a .zip and returns its path.

### `fBinary`

`fBinary(size)` -> string

Writes size (e.g. "2MB", "512KB", or a plain byte count) of pseudo-random bytes and returns the path.

### `fPDF`

`fPDF([pages])` -> string

Generates an N-page PDF (default 1) and returns its path.

### `fXLSX`

`fXLSX([rows] [cols])` -> string

Generates a workbook (default 10x5) and returns its path.

### `fDOCX`

`fDOCX([paragraphs])` -> string

Generates a minimal .docx (default 3 paragraphs) and returns its path.

## string

### `upper`

`upper(s)`

Uppercases s.

### `lower`

`lower(s)`

Lowercases s.

### `title`

`title(s)`

Title-cases s (capitalizes each word).

### `trim`

`trim(s)`

Removes leading/trailing whitespace from s.

### `trimPrefix`

`trimPrefix(prefix, s)`

Removes prefix from s if present.

### `trimSuffix`

`trimSuffix(suffix, s)`

Removes suffix from s if present.

### `trunc`

`trunc(n, s)`

Truncates s to n characters (negative n truncates from the left).

### `replace`

`replace(old, new, s)`

Replaces every occurrence of old with new in s.

### `contains`

`contains(substr, s)`

Reports whether s contains substr.

### `hasPrefix`

`hasPrefix(prefix, s)`

Reports whether s starts with prefix.

### `hasSuffix`

`hasSuffix(suffix, s)`

Reports whether s ends with suffix.

### `add`

`add(a, b, ...)`

Sums its integer arguments.

### `join`

`join(sep, list)`

Joins list's elements with sep.

### `split`

`split(sep, s)`

Splits s on sep, returning a dict of _0, _1, ...

### `default`

`default(fallback, value)`

Returns value, or fallback if value is empty.

### `ternary`

`ternary(truthy, falsy, cond)`

Returns truthy if cond is true, else falsy.

### `list`

`list(a, b, ...)`

Builds a list from its arguments.

### `dict`

`dict(k1, v1, k2, v2, ...)`

Builds a dict (map[string]any) from alternating keys/values.

### `env` **(unsafe)**

`env(name)` -> string

The value of environment variable `name`. Removed by default; restored only under --template-unsafe-funcs.

Requires `--template-unsafe-funcs` / `-Z`.

### `expandenv` **(unsafe)**

`expandenv(s)` -> string

Expands $VAR references in s from the environment. Removed by default; restored only under --template-unsafe-funcs.

Requires `--template-unsafe-funcs` / `-Z`.

### `getHostByName` **(unsafe)**

`getHostByName(host)` -> string

Resolves host to an IP address (performs a DNS lookup). Removed by default; restored only under --template-unsafe-funcs.

Requires `--template-unsafe-funcs` / `-Z`.

## date

### `now`

The current time.Time.

### `date`

`date(fmt, t)`

Formats t using a reference-time layout string.

## encoding

### `b64enc`

`b64enc(s)`

Base64-encodes s.

### `b64dec`

`b64dec(s)`

Base64-decodes s.

## email

### `quotedPrintable`

`quotedPrintable(s)` -> string

Quoted-printable encodes s (RFC 2045).

### `mimeWord`

`mimeWord(s)` -> string

RFC 2047 encoded-word (UTF-8 Q-encoding) for non-ASCII email header values.

### `rfc2822Date`

`rfc2822Date([time])` -> string

Formats the given time (default now) as RFC 1123Z, for email Date headers.

### `fEmail`

`fEmail([domain])` -> string

Random email address, optionally on the given domain.

### `fileOf`

`fileOf(path)` -> string

Validates path exists and returns it unchanged (passthrough for chaining into an email's attachments).

### `attach`

`attach(path...)` -> string

Joins multiple file paths with "::" for send --attach's email-attachment chaining convention.

## files

### `readFile`

`readFile(path)` -> string

Returns the file's contents as a string.

### `glob`

`glob(pattern)` -> []string

Returns matching file paths.

### `basename`

`basename(path)` -> string

Returns the final path element.

### `dirname`

`dirname(path)` -> string

Returns all but the final path element.

### `ext`

`ext(path)` -> string

Returns the file extension, including the leading dot.

## ansi

### `ansi`

`ansi(code, text)` -> string

Wraps text in the given SGR escape code(s) (e.g. "1;32"), resetting after. For most use, prefer the bare color/style functions below with an explicit {{ reset }}.

### `reset`

`reset()` -> string

The bare ANSI reset sequence — pair with any color/style function below, e.g. {{ blue }}text{{ reset }}.

### `bold`

`bold()` -> string

The bare ANSI bold sequence.

### `dim`

`dim()` -> string

The bare ANSI dim sequence.

### `red`

`red()` -> string

The bare ANSI red foreground sequence.

### `green`

`green()` -> string

The bare ANSI green foreground sequence.

### `yellow`

`yellow()` -> string

The bare ANSI yellow foreground sequence.

### `blue`

`blue()` -> string

The bare ANSI blue foreground sequence.

### `magenta`

`magenta()` -> string

The bare ANSI magenta foreground sequence.

### `cyan`

`cyan()` -> string

The bare ANSI cyan foreground sequence.

### `white`

`white()` -> string

The bare ANSI white foreground sequence.

