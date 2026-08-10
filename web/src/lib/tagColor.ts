// Deterministic hash-tag-name -> fixed palette color mapping, shared by
// MessageRow's tag chips and the sidebar's tags nav (M6.1) so the same tag
// always renders with the same color everywhere in the UI. The palette
// reuses Tailwind's standard color scale (not ad-hoc hex values) at a fixed
// shade pair (dot + text) chosen to read clearly against the app's light
// surface tokens.
export interface TagColor {
  dot: string
  text: string
  /** Soft badge background, paired with `text` (M6.1's tag-badge treatment
   * in the Inbox row and Message Detail — the sidebar's tag nav keeps the
   * plain dot+text-on-transparent style since it already sits on a card). */
  bg: string
}

const PALETTE: TagColor[] = [
  { dot: 'bg-indigo-500', text: 'text-indigo-700', bg: 'bg-indigo-100' },
  { dot: 'bg-emerald-500', text: 'text-emerald-700', bg: 'bg-emerald-100' },
  { dot: 'bg-amber-500', text: 'text-amber-700', bg: 'bg-amber-100' },
  { dot: 'bg-rose-500', text: 'text-rose-700', bg: 'bg-rose-100' },
  { dot: 'bg-cyan-500', text: 'text-cyan-700', bg: 'bg-cyan-100' },
  { dot: 'bg-fuchsia-500', text: 'text-fuchsia-700', bg: 'bg-fuchsia-100' },
  { dot: 'bg-lime-600', text: 'text-lime-700', bg: 'bg-lime-100' },
  { dot: 'bg-orange-500', text: 'text-orange-700', bg: 'bg-orange-100' },
]

/** A small, stable string hash (FNV-1a-ish) — not cryptographic, just
 * deterministic across runs/sessions for a given tag string. */
function hashString(s: string): number {
  let h = 2166136261
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

/** Returns the same TagColor for the same tag name every time. */
export function tagColor(tag: string): TagColor {
  const idx = hashString(tag) % PALETTE.length
  return PALETTE[idx]
}
