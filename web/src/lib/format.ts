const UNITS = ['B', 'KB', 'MB', 'GB', 'TB']

/** Human-readable byte size, e.g. formatBytes(1536) === "1.5 KB". */
export function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B'
  const exp = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), UNITS.length - 1)
  const value = bytes / 1024 ** exp
  const precision = exp === 0 ? 0 : value < 10 ? 1 : 0
  return `${value.toFixed(precision)} ${UNITS[exp]}`
}

const RELATIVE_STEPS: Array<[number, string]> = [
  [60, 's'],
  [60, 'm'],
  [24, 'h'],
  [7, 'd'],
  [4.345, 'w'],
]

/** Coarse relative-time label, e.g. "2m ago", "just now", "3d ago". */
export function formatRelativeTime(iso: string, now: Date = new Date()): string {
  const then = new Date(iso)
  let deltaSeconds = Math.max(0, (now.getTime() - then.getTime()) / 1000)

  if (deltaSeconds < 5) return 'just now'

  let unit = 's'
  let value = deltaSeconds
  for (const [divisor, label] of RELATIVE_STEPS) {
    if (value < divisor) {
      unit = label
      break
    }
    value /= divisor
    unit = label
  }
  return `${Math.floor(value)}${unit} ago`
}

/** Exact, locale-formatted timestamp for a title/tooltip. */
export function formatExactTime(iso: string): string {
  return new Date(iso).toLocaleString()
}

/**
 * Renders a list of addresses as "first, +N more" once it exceeds `max`
 * entries (default 1: show the first, badge the rest).
 */
export function truncateList(items: string[], max = 1): { shown: string[]; more: number } {
  if (items.length <= max) return { shown: items, more: 0 }
  return { shown: items.slice(0, max), more: items.length - max }
}
