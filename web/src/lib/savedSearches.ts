// Client-only (localStorage) CRUD for the sidebar's "Saved searches" list
// (M6.1). Deliberately not server-persisted: SPEC.md has no multi-device
// sync requirement for the Web UI today, so a small per-browser list is the
// simplest thing that satisfies MOCKUP.html's saved-searches affordance.
// Revisit as a server-persisted table if multi-device sync is ever needed.
import type { ListMessagesParams } from './apiTypes'

export interface SavedSearch {
  name: string
  query: ListMessagesParams
}

const STORAGE_KEY = 'maelsink.saved_searches'

/** Coerces a legacy single-tag string (pre-M8.2) into the current
 * string[] shape, so old saved searches keep working after the sidebar's
 * tag filter became multi-select. */
function normalizeQuery(query: ListMessagesParams): ListMessagesParams {
  const legacyTag = query.tag as unknown
  if (typeof legacyTag === 'string') {
    return { ...query, tag: [legacyTag] }
  }
  return query
}

function readAll(): SavedSearch[] {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return (parsed as SavedSearch[]).map((s) => ({ ...s, query: normalizeQuery(s.query) }))
  } catch {
    return []
  }
}

function writeAll(searches: SavedSearch[]): void {
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(searches))
}

/** Returns every saved search, in insertion order. */
export function listSavedSearches(): SavedSearch[] {
  return readAll()
}

/** Saves (or overwrites, by name) a named search. */
export function saveSearch(name: string, query: ListMessagesParams): SavedSearch[] {
  const trimmed = name.trim()
  if (!trimmed) return readAll()
  const existing = readAll().filter((s) => s.name !== trimmed)
  const updated = [...existing, { name: trimmed, query }]
  writeAll(updated)
  return updated
}

/** Removes a saved search by name. */
export function deleteSavedSearch(name: string): SavedSearch[] {
  const updated = readAll().filter((s) => s.name !== name)
  writeAll(updated)
  return updated
}
