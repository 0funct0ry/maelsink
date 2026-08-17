---
title: Integration Testing
description: Running maelsink as an ephemeral SMTP sink in CI, with health-check readiness and REST assertions.
---

maelsink is well suited to running as a short-lived SMTP sink inside CI: start it, run
your app's test suite against it, assert on what got captured via the REST API, then
tear it down.

![Headless health check, sending a signup-confirmation message, and asserting on it via the REST API](/maelsink/recordings/ci-smoke-test.gif)

## Headless mode

For CI you almost never need the Web UI — run with `-u/--headless` to skip starting it
and keep only the SMTP server and REST API:

```sh
maelsink serve --headless --db ./ci.db --smtp-port 1026 --api-port 9091
```

## Readiness: poll `/api/v1/health`

Don't assume the server is listening the instant the process starts — poll the health
endpoint until it returns `200`:

```sh
for i in $(seq 1 20); do
  if curl -sf http://127.0.0.1:9091/api/v1/health > /dev/null; then
    echo "maelsink is ready"
    break
  fi
  sleep 0.5
done
```

A healthy response looks like:

```json
{"db":"ok","smtp":"listening","status":"ok"}
```

## Asserting on captured messages

Once your app under test has sent mail, query `/api/v1/messages` with filters to assert
on what arrived:

```sh
curl -s "http://127.0.0.1:9091/api/v1/messages?subject=Welcome&to=newuser@example.com"
```

See [Filters and Search](/maelsink/docs/usage/filters-and-search/) for the full query parameter
reference, and the [REST API Reference](/maelsink/docs/rest-api-reference/) for response shapes.

## Worked example: a CI job

```sh
#!/usr/bin/env bash
set -euo pipefail

DB=$(mktemp -u).db
maelsink serve --headless --db "$DB" --smtp-port 1026 --api-port 9091 &
MAELSINK_PID=$!
trap 'kill $MAELSINK_PID 2>/dev/null' EXIT

# Wait for readiness
for i in $(seq 1 20); do
  curl -sf http://127.0.0.1:9091/api/v1/health > /dev/null && break
  sleep 0.5
done

# Point the app under test at 127.0.0.1:1026 for SMTP, run its test suite here.
# e.g.: MY_APP_SMTP_HOST=127.0.0.1 MY_APP_SMTP_PORT=1026 npm test

# Assert the welcome email was captured
COUNT=$(curl -s "http://127.0.0.1:9091/api/v1/messages?subject=Welcome" | jq '.total')
if [ "$COUNT" -lt 1 ]; then
  echo "expected at least one welcome email, got $COUNT"
  exit 1
fi

echo "integration check passed"
```

The `trap` ensures maelsink is killed even if an assertion fails partway through.

## Authentication in CI

If your app under test requires SMTP AUTH to be configured, see
[Password files](/maelsink/docs/configuration/password-files/)'s `smtp.auth.accept_any` and
`MAELSINK_SMTP_AUTH` shortcuts — both avoid needing to manage a credential file for a
throwaway CI run.
