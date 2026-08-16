// Persistence for the Vars screen (SPEC.md §7.7.1): session variables live
// entirely in browser localStorage, never on the compose backend — a
// restart or reload of the backend loses no user data since it never held
// any. Failures (private mode, disabled storage) degrade to an empty map
// rather than throwing.

const STORAGE_KEY = 'maelsink-compose-vars'

export type VarsMap = Record<string, string>

export function readVars(): VarsMap {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object') return parsed as VarsMap
    return {}
  } catch {
    return {}
  }
}

export function writeVars(vars: VarsMap): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(vars))
  } catch {
    // localStorage unavailable — the in-memory store still reflects the
    // change for the current session, it just won't survive a reload.
  }
}
