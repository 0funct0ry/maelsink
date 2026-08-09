# maelsink Web UI

Vite + React + TypeScript + Tailwind CSS single-page app, embedded into the
`maelsink` binary at build time (see `internal/webui`).

```bash
npm ci
npm run dev      # Vite dev server (UI development only, never shipped)
npm run build    # -> ../internal/webui/dist, embedded via go:embed
npm test         # component/unit tests
```

See `internal-docs/SPEC.md` §8 and `internal-docs/STYLE_GUIDE.md` for the
design system and architecture this app builds against.
