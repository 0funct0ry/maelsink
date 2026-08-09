// Package uiapi will hold the Web UI's internal-only /ui-api/v1/* endpoints
// (SPEC.md §2.3 point 4) — e.g. the empty-state SMTP connection info and any
// other UI-specific data that has no business living in the stable /api/v1
// surface (internal/api). No endpoints exist yet as of M5.0; this package
// exists so the boundary is in place before M6.0 needs it.
package uiapi
