// Detects whether a rendered /compose-api/v1/render response is an HTML
// message, and extracts just the markup — so the Composer's Preview pane
// can show an actual rendered rendering instead of raw markup as text.
//
// For "eml" format, `rendered` is the full raw RFC 5322 document (headers +
// blank line + body): look for a Content-Type: text/html header and split
// off the body. For "json" format, `rendered` is the whole re-marshaled
// cliclient.MessageSpec document: parse it and pull out the `html` field.

import type { TemplateFormat } from './composeApi'

function extractEmlHtmlBody(raw: string): string | null {
  const separatorMatch = raw.match(/\r?\n\r?\n/)
  if (!separatorMatch || separatorMatch.index == null) return null

  const headerBlock = raw.slice(0, separatorMatch.index)
  const body = raw.slice(separatorMatch.index + separatorMatch[0].length)

  const isHTML = headerBlock
    .split(/\r?\n/)
    .some((line) => /^content-type\s*:/i.test(line) && /text\/html/i.test(line))

  return isHTML ? body : null
}

function extractJSONHtmlBody(raw: string): string | null {
  try {
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed.html === 'string' && parsed.html.trim() !== '') {
      return parsed.html
    }
  } catch {
    // not valid JSON (e.g. mid-edit) — no HTML preview to show
  }
  return null
}

export function extractHtmlPreview(format: TemplateFormat, rendered: string): string | null {
  if (!rendered) return null
  return format === 'eml' ? extractEmlHtmlBody(rendered) : extractJSONHtmlBody(rendered)
}
