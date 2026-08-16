// Persistence for named Composer drafts (SPEC.md §7.7.4.1) — distinct from
// Vars (lib/varsStorage.ts): a draft is a saved template body/format pair, a
// var is a substitution value shared across every draft. Lives entirely in
// browser localStorage, same as vars.

import type { AttachmentInput, TemplateFormat } from './composeApi'

const STORAGE_KEY = 'maelsink-compose-drafts'

export interface Draft {
  format: TemplateFormat
  content: string
  // attachments only applies when format is "eml" (see composeApi.ts's
  // AttachmentInput doc) — omitted (or empty) for older drafts/json drafts.
  attachments?: AttachmentInput[]
}

export type DraftsMap = Record<string, Draft>

export function readDrafts(): DraftsMap {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object') return parsed as DraftsMap
    return {}
  } catch {
    return {}
  }
}

export function writeDrafts(drafts: DraftsMap): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(drafts))
  } catch {
    // localStorage unavailable — the in-memory store still reflects the
    // change for the current session, it just won't survive a reload.
  }
}
