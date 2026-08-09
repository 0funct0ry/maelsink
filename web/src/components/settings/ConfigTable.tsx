import { useEffect, useMemo, useState } from 'react'
import { Search, Info } from 'lucide-react'
import { getInfo } from '../../lib/uiApiClient'
import type { UiInfo } from '../../lib/apiTypes'

interface ConfigRow {
  label: string
  value: string
}

export default function ConfigTable() {
  const [info, setInfo] = useState<UiInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    getInfo()
      .then((i) => {
        if (!cancelled) setInfo(i)
      })
      .catch(() => {
        if (!cancelled) setError('Failed to load config.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Only fields actually returned by /ui-api/v1/info are shown here — the Go
  // backend (internal/config) has no per-key provenance/dump endpoint yet,
  // so this is intentionally a small, real subset rather than a fabricated
  // full config listing.
  const rows: ConfigRow[] = useMemo(() => {
    if (!info) return []
    return [
      { label: 'SMTP Host', value: info.smtp.host },
      { label: 'SMTP Port', value: String(info.smtp.port) },
      { label: 'API Auth Enabled', value: info.auth_enabled ? 'true' : 'false' },
    ]
  }, [info])

  const filteredRows = rows.filter((r) => r.label.toLowerCase().includes(filter.toLowerCase()))

  return (
    <div className="rounded-lg border border-border bg-surface p-4 shadow-sm">
      <div className="mb-3 flex items-center gap-2">
        <Info className="h-4 w-4 text-text-tertiary" aria-hidden="true" />
        <h2 className="text-sm font-semibold text-text-primary">Config</h2>
      </div>

      <div className="relative mb-3 flex h-8 items-center gap-2 rounded-md border border-border-soft bg-surface px-2.5">
        <Search className="h-3.5 w-3.5 flex-none text-text-tertiary" aria-hidden="true" />
        <input
          type="text"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="Filter config fields..."
          aria-label="Filter config fields"
          className="w-full bg-transparent text-sm text-text-primary placeholder:text-text-tertiary focus:outline-none"
        />
      </div>

      {loading && (
        <div className="space-y-2" data-testid="config-loading">
          <div className="h-4 w-full animate-pulse rounded-sm bg-surface-2" />
          <div className="h-4 w-full animate-pulse rounded-sm bg-surface-2" />
          <div className="h-4 w-2/3 animate-pulse rounded-sm bg-surface-2" />
        </div>
      )}

      {!loading && error && <p className="text-sm text-danger">{error}</p>}

      {!loading && !error && info && (
        <table className="w-full text-sm">
          <tbody>
            {filteredRows.map((row) => (
              <tr
                key={row.label}
                className="border-b border-border-soft last:border-b-0 hover:bg-surface-2"
              >
                <td className="w-1/2 py-2 pr-4 font-mono font-medium text-text-primary">
                  {row.label}
                </td>
                <td className="py-2 font-mono text-text-primary">{row.value}</td>
              </tr>
            ))}
            {filteredRows.length === 0 && (
              <tr>
                <td colSpan={2} className="py-2 text-text-tertiary">
                  No matching fields.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      )}
    </div>
  )
}
