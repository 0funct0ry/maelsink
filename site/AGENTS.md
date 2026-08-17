## Development

When starting the dev server, use background mode:

```
astro dev --background
```

Manage the background server with `astro dev stop`, `astro dev status`, and `astro dev logs`.

## Copy style

All prose on this site (marketing page, docs shell, footers — any user-facing text,
not just the Hero) should be **precise, documentation-flavored, and formal** —
not playful startup marketing copy. Concretely:

- Full grammatical sentences only — no sentence fragments (e.g. not "One binary, a
  realtime web UI, and a REST API your test suite can talk to"; instead "It provides
  a REST API that can be used by test suites and other development tools.")
- Plain declarative statements over wordplay, puns, or cleverness (no "fake SMTP
  server for your real development flow"-style contrasts)
- Technical/precise verbs over colloquial ones (e.g. "intercepts" not "catches";
  "non-production environments" not "dev, staging, and CI")
- Third-person, spec/documentation register ("Maelsink is..." / "It intercepts...
  and provides...") rather than second-person ad copy or exclamatory tone
- No idioms, em-dash asides, or informal filler ("no surprises", "no real
  recipients" style phrasing)
- `maelsink` stays lowercase everywhere it appears as the product/brand name,
  including mid-sentence — capitalize only when grammar requires it (start of a
  sentence or heading), e.g. "Maelsink is a lightweight email capture service..."
  but "...built with maelsink" mid-sentence

Reference example (target tone):
> Maelsink is a lightweight email capture service for non-production environments.
> It intercepts outgoing application emails for inspection through a realtime web
> UI and provides a REST API that can be used by test suites and other development
> tools.

## Documentation

Full documentation: https://docs.astro.build

Consult these guides before working on related tasks:

- [Adding pages, dynamic routes, or middleware](https://docs.astro.build/en/guides/routing/)
- [Working with Astro components](https://docs.astro.build/en/basics/astro-components/)
- [Using React, Vue, Svelte, or other framework components](https://docs.astro.build/en/guides/framework-components/)
- [Adding or managing content](https://docs.astro.build/en/guides/content-collections/)
- [Adding styles or using Tailwind](https://docs.astro.build/en/guides/styling/)
- [Supporting multiple languages](https://docs.astro.build/en/guides/internationalization/)
