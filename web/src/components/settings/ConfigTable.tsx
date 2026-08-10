import { useEffect, useMemo, useState } from 'react'
import { Search, Info } from 'lucide-react'
import { getConfig } from '../../lib/uiApiClient'
import Badge from '../common/Badge'
import type { ConfigEntry, ConfigSource } from '../../lib/apiTypes'

type SourceFilter = 'all' | ConfigSource['layer']

const SOURCE_FILTERS: Array<{ value: SourceFilter; label: string }> = [
  { value: 'all', label: 'All' },
  { value: 'default', label: 'Default' },
  { value: 'file', label: 'File' },
  { value: 'env', label: 'Env' },
  { value: 'flag', label: 'Flag' },
]

const SOURCE_LABEL: Record<ConfigSource['layer'], string> = {
  default: 'Default',
  file: 'Config file',
  env: 'Environment variable',
  flag: 'CLI flag',
}

function sourceBadgeVariant(layer: ConfigSource['layer']) {
  return `source-${layer}` as const
}

function formatValue(value: unknown): string {
  if (Array.isArray(value)) return value.length === 0 ? '[]' : value.join(', ')
  if (typeof value === 'boolean') return value ? 'true' : 'false'
  return String(value)
}

/**
 * Full per-section config dump backed by GET /ui-api/v1/config (M6.1):
 * every non-secret key's resolved value plus real provenance (which layer
 * resolved it, and that layer's origin), grouped by section, with source
 * filter chips. Replaces the earlier 3-hardcoded-row placeholder.
 */
export default function ConfigTable() {
  const [entries, setEntries] = useState<ConfigEntry[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState('')
  const [sourceFilter, setSourceFilter] = useState<SourceFilter>('all')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    getConfig()
      .then((e) => {
        if (!cancelled) setEntries(e)
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

  const filteredEntries = useMemo(() => {
    if (!entries) return []
    return entries.filter((e) => {
      if (sourceFilter !== 'all' && e.source?.layer !== sourceFilter) return false
      if (filter && !e.key.toLowerCase().includes(filter.toLowerCase())) return false
      return true
    })
  }, [entries, filter, sourceFilter])

  const bySection = useMemo(() => {
    const groups = new Map<string, ConfigEntry[]>()
    for (const e of filteredEntries) {
      const list = groups.get(e.section) ?? []
      list.push(e)
      groups.set(e.section, list)
    }
    return groups
  }, [filteredEntries])

  return (
    <div className="rounded-lg border border-border bg-surface p-4 shadow-sm">
      <div className="mb-3 flex items-center gap-2">
        <Info className="h-4 w-4 text-text-tertiary" aria-hidden="true" />
        <h2 className="text-sm font-semibold text-text-primary">Config</h2>
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-1.5">
        {SOURCE_FILTERS.map((f) => (
          <button
            key={f.value}
            type="button"
            onClick={() => setSourceFilter(f.value)}
            className={`rounded-full border px-2.5 py-1 text-[12px] font-medium transition-colors ${
              sourceFilter === f.value
                ? 'border-accent bg-accent-soft text-accent'
                : 'border-border-soft bg-bg text-text-secondary hover:bg-surface-2'
            }`}
          >
            {f.label}
          </button>
        ))}
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

      {!loading && !error && entries && (
        <div className="space-y-4">
          {Array.from(bySection.entries()).map(([section, rows]) => (
            <div key={section}>
              <h3 className="mb-1.5 text-[11px] font-semibold uppercase tracking-[0.04em] text-text-tertiary">
                {section}
              </h3>
              <table className="w-full text-sm">
                <tbody>
                  {rows.map((row) => (
                    <tr key={row.key} className="border-b border-border-soft last:border-b-0 hover:bg-surface-2">
                      <td className="w-1/3 py-2 pr-4 font-mono font-medium text-text-primary">{row.key}</td>
                      <td className="w-1/3 py-2 pr-4 font-mono text-text-primary">{formatValue(row.value)}</td>
                      <td className="py-2 pr-4">
                        <Badge variant={sourceBadgeVariant(row.source.layer)}>{SOURCE_LABEL[row.source.layer]}</Badge>
                      </td>
                      <td className="py-2 font-mono text-[12px] text-text-tertiary">{row.source.origin}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
          {filteredEntries.length === 0 && <p className="py-2 text-text-tertiary">No matching fields.</p>}
        </div>
      )}
    </div>
  )
}
